package ledger

import (
	"context"
	"errors"
	"strings"

	"github.com/franco-roura/ledger-01/backend/internal/db/dbq"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Profile is what an identity provider tells us about a person.
type Profile struct {
	// Stable subject id from the provider ("dev:<email>" for the dev login).
	Subject   string
	Email     string
	Name      string
	AvatarURL string
}

type User struct {
	ID        uuid.UUID
	Email     string
	Name      string
	AvatarURL string
}

// Household is the caller's active household, with their role in it.
type Household struct {
	ID           uuid.UUID
	Name         string
	BaseCurrency string
	Role         string
}

func toUser(u dbq.User) User {
	return User{ID: u.ID, Email: u.Email, Name: u.Name, AvatarURL: u.AvatarUrl}
}

func toHousehold(h dbq.Household, role string) Household {
	return Household{ID: h.ID, Name: h.Name, BaseCurrency: strings.TrimSpace(h.BaseCurrency), Role: role}
}

// EnsureUser finds or creates the user for a verified profile, accepts any pending invites for
// their email (the last one joined becomes their active household) and, when they belong to no
// household, gives them one (as its owner) seeded with default categories. Returns the caller as
// an Actor.
//
// Two first sign-ins can race (two devices or tabs). The loser's insert hits the unique index
// only after the winner has committed, so retrying finds the winner's user.
func (s *Service) EnsureUser(ctx context.Context, p Profile) (Actor, error) {
	var conf *ConflictError
	for attempt := 0; ; attempt++ {
		actor, err := s.ensureUserOnce(ctx, p)
		if err == nil || !errors.As(err, &conf) || attempt == 2 {
			return actor, err
		}
	}
}

func (s *Service) ensureUserOnce(ctx context.Context, p Profile) (Actor, error) {
	var actor Actor
	err := s.inTx(ctx, func(q *dbq.Queries) error {
		u, err := q.GetUserByGoogleSub(ctx, p.Subject)
		if errors.Is(err, pgx.ErrNoRows) {
			// Same verified email signing in through another subject (e.g. dev login first).
			u, err = q.GetUserByEmail(ctx, p.Email)
		}
		switch {
		case err == nil:
			if _, err := q.UpdateUserFromGoogle(ctx, dbq.UpdateUserFromGoogleParams{
				ID: u.ID, Email: p.Email, Name: p.Name, AvatarUrl: p.AvatarURL,
			}); err != nil {
				return dbErr(err, "user")
			}
		case errors.Is(err, pgx.ErrNoRows):
			u, err = q.CreateUser(ctx, dbq.CreateUserParams{
				ID: newID(), GoogleSub: p.Subject, Email: p.Email, Name: p.Name, AvatarUrl: p.AvatarURL,
			})
			if err != nil {
				return dbErr(err, "user")
			}
		default:
			return err
		}

		joined, err := q.AcceptHouseholdInvites(ctx, dbq.AcceptHouseholdInvitesParams{Email: p.Email, UserID: u.ID})
		if err != nil {
			return err
		}
		if len(joined) > 0 {
			last := joined[len(joined)-1]
			if err := q.SetActiveHousehold(ctx, dbq.SetActiveHouseholdParams{HouseholdID: &last, ID: u.ID}); err != nil {
				return err
			}
		}

		m, err := q.GetActiveMembership(ctx, u.ID)
		if err == nil {
			actor = Actor{UserID: u.ID, HouseholdID: m.HouseholdID, Role: m.Role}
			return nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		h, err := createHousehold(ctx, q, u.ID, p.Name)
		if err != nil {
			return err
		}
		actor = Actor{UserID: u.ID, HouseholdID: h, Role: RoleOwner}
		return nil
	})
	return actor, err
}

// createHousehold gives a user a new household they own and makes it their active one.
func createHousehold(ctx context.Context, q *dbq.Queries, user uuid.UUID, userName string) (uuid.UUID, error) {
	name := "Household"
	if first, _, _ := strings.Cut(strings.TrimSpace(userName), " "); first != "" {
		name = first + "'s household"
	}
	h, err := q.CreateHousehold(ctx, dbq.CreateHouseholdParams{ID: newID(), Name: name, BaseCurrency: "USD"})
	if err != nil {
		return uuid.Nil, err
	}
	if err := q.AddHouseholdMember(ctx, dbq.AddHouseholdMemberParams{HouseholdID: h.ID, UserID: user, Role: RoleOwner}); err != nil {
		return uuid.Nil, err
	}
	if err := q.SetActiveHousehold(ctx, dbq.SetActiveHouseholdParams{HouseholdID: &h.ID, ID: user}); err != nil {
		return uuid.Nil, err
	}
	return h.ID, seedCategories(ctx, q, h.ID)
}

// Me returns the caller's user and active household.
func (s *Service) Me(ctx context.Context, a Actor) (User, Household, error) {
	u, err := s.q.GetUser(ctx, a.UserID)
	if err != nil {
		return User{}, Household{}, dbErr(err, "user")
	}
	h, err := s.Household(ctx, a)
	if err != nil {
		return User{}, Household{}, err
	}
	return toUser(u), h, nil
}

// UpdateProfile changes the caller's display name.
func (s *Service) UpdateProfile(ctx context.Context, a Actor, name *string) (User, error) {
	if name != nil {
		n := strings.TrimSpace(*name)
		if n == "" || len(n) > 80 {
			return User{}, invalid("name", "must be 1-80 characters")
		}
		name = &n
	}
	u, err := s.q.UpdateUserProfile(ctx, dbq.UpdateUserProfileParams{ID: a.UserID, Name: name})
	if err != nil {
		return User{}, dbErr(err, "user")
	}
	return toUser(u), nil
}

type Currency struct {
	Code       string
	Name       string
	Symbol     string
	MinorUnits int
}

func (s *Service) Currencies(ctx context.Context) ([]Currency, error) {
	rows, err := s.q.ListCurrencies(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]Currency, len(rows))
	for i, r := range rows {
		out[i] = Currency{Code: strings.TrimSpace(r.Code), Name: r.Name, Symbol: r.Symbol, MinorUnits: int(r.MinorUnits)}
	}
	return out, nil
}

type seedCategory struct {
	kind  dbq.CategoryKind
	name  string
	icon  string
	color string
}

var defaultCategories = []seedCategory{
	{dbq.CategoryKindExpense, "Food & Drink", "utensils", "cat-orange"},
	{dbq.CategoryKindExpense, "Groceries", "shopping-cart", "cat-amber"},
	{dbq.CategoryKindExpense, "Shopping", "shopping-bag", "cat-pink"},
	{dbq.CategoryKindExpense, "Transport", "bus", "cat-yellow"},
	{dbq.CategoryKindExpense, "Car", "car", "cat-sky"},
	{dbq.CategoryKindExpense, "Home", "house", "cat-brown"},
	{dbq.CategoryKindExpense, "Bills & Fees", "receipt", "cat-lime"},
	{dbq.CategoryKindExpense, "Utilities", "zap", "cat-cyan"},
	{dbq.CategoryKindExpense, "Healthcare", "heart-pulse", "cat-red"},
	{dbq.CategoryKindExpense, "Entertainment", "film", "cat-violet"},
	{dbq.CategoryKindExpense, "Travel", "plane", "cat-rose"},
	{dbq.CategoryKindExpense, "Education", "graduation-cap", "cat-blue"},
	{dbq.CategoryKindExpense, "Sport & Hobbies", "dumbbell", "cat-teal"},
	{dbq.CategoryKindExpense, "Beauty", "sparkles", "cat-pink"},
	{dbq.CategoryKindExpense, "Gifts", "gift", "cat-green"},
	{dbq.CategoryKindExpense, "Family & Personal", "users", "cat-indigo"},
	{dbq.CategoryKindExpense, "Work", "briefcase", "cat-slate"},
	{dbq.CategoryKindExpense, "Other", "ellipsis", "cat-slate"},
	{dbq.CategoryKindIncome, "Salary", "banknote", "cat-green"},
	{dbq.CategoryKindIncome, "Business", "briefcase", "cat-teal"},
	{dbq.CategoryKindIncome, "Dividends", "trending-up", "cat-lime"},
	{dbq.CategoryKindIncome, "Refund", "undo-2", "cat-pink"},
	{dbq.CategoryKindIncome, "Gifts", "gift", "cat-green"},
	{dbq.CategoryKindIncome, "Loan", "landmark", "cat-blue"},
	{dbq.CategoryKindIncome, "Other", "ellipsis", "cat-slate"},
}

func seedCategories(ctx context.Context, q *dbq.Queries, household uuid.UUID) error {
	pos := map[dbq.CategoryKind]int32{}
	for _, c := range defaultCategories {
		if _, err := q.CreateCategory(ctx, dbq.CreateCategoryParams{
			ID: newID(), HouseholdID: household, Kind: c.kind, Name: c.name, Icon: c.icon, Color: c.color, Position: pos[c.kind],
		}); err != nil {
			return err
		}
		pos[c.kind]++
	}
	return nil
}
