package ledger

import (
	"context"
	"fmt"
	"net/mail"
	"strings"
	"time"

	"github.com/franco-roura/ledger-01/backend/internal/db/dbq"
	"github.com/google/uuid"
)

// Member is a person in the household, or (Former) someone who entered rows here and left.
type Member struct {
	UserID    uuid.UUID
	Email     string
	Name      string
	AvatarURL string
	Role      string
	Former    bool
}

// Invite is a pending invite, accepted on that email's next sign-in.
type Invite struct {
	Email     string
	CreatedAt time.Time
}

// HouseholdMembers is everything Settings → Household lists.
type HouseholdMembers struct {
	Members []Member
	Former  []Member
	Invites []Invite
}

// MembershipSummary is one household the caller belongs to (for the switcher).
type MembershipSummary struct {
	ID          uuid.UUID
	Name        string
	Role        string
	MemberCount int
	Active      bool
}

// Household returns the caller's active household.
func (s *Service) Household(ctx context.Context, a Actor) (Household, error) {
	h, err := s.q.GetHousehold(ctx, a.HouseholdID)
	if err != nil {
		return Household{}, dbErr(err, "household")
	}
	return toHousehold(h, a.Role), nil
}

// UpdateHousehold renames the household and/or changes its base currency. Owner only.
func (s *Service) UpdateHousehold(ctx context.Context, a Actor, name, baseCurrency *string) (Household, error) {
	if err := requireOwner(a); err != nil {
		return Household{}, err
	}
	if name != nil {
		n := strings.TrimSpace(*name)
		if n == "" || len(n) > 80 {
			return Household{}, invalid("name", "must be 1-80 characters")
		}
		name = &n
	}
	if baseCurrency != nil {
		c := strings.ToUpper(strings.TrimSpace(*baseCurrency))
		if _, err := s.q.GetCurrency(ctx, c); err != nil {
			return Household{}, invalid("base_currency", "unknown currency %q", c)
		}
		baseCurrency = &c
	}
	h, err := s.q.UpdateHousehold(ctx, dbq.UpdateHouseholdParams{ID: a.HouseholdID, Name: name, BaseCurrency: baseCurrency})
	if err != nil {
		return Household{}, dbErr(err, "household")
	}
	return toHousehold(h, a.Role), nil
}

// Members lists the household's members, former creators and pending invites.
func (s *Service) Members(ctx context.Context, a Actor) (HouseholdMembers, error) {
	var out HouseholdMembers
	ms, err := s.q.ListHouseholdMembers(ctx, a.HouseholdID)
	if err != nil {
		return out, err
	}
	for _, m := range ms {
		out.Members = append(out.Members, Member{UserID: m.ID, Email: m.Email, Name: m.Name, AvatarURL: m.AvatarUrl, Role: m.Role})
	}
	fs, err := s.q.ListFormerCreators(ctx, a.HouseholdID)
	if err != nil {
		return out, err
	}
	for _, f := range fs {
		out.Former = append(out.Former, Member{UserID: f.ID, Email: f.Email, Name: f.Name, AvatarURL: f.AvatarUrl, Former: true})
	}
	is, err := s.q.ListHouseholdInvites(ctx, a.HouseholdID)
	if err != nil {
		return out, err
	}
	for _, i := range is {
		out.Invites = append(out.Invites, Invite{Email: i.Email, CreatedAt: i.CreatedAt})
	}
	return out, nil
}

func normalizeEmail(email string) (string, error) {
	e := strings.ToLower(strings.TrimSpace(email))
	addr, err := mail.ParseAddress(e)
	if err != nil || addr.Address != e || len(e) > 254 {
		return "", invalid("email", "must be an email address")
	}
	return e, nil
}

// InviteMember adds a pending invite for an email. Owner only. The email still has to be on the
// deployment's allowlist to sign in.
func (s *Service) InviteMember(ctx context.Context, a Actor, email string) (Invite, error) {
	if err := requireOwner(a); err != nil {
		return Invite{}, err
	}
	e, err := normalizeEmail(email)
	if err != nil {
		return Invite{}, err
	}
	member, err := s.q.IsHouseholdMemberEmail(ctx, dbq.IsHouseholdMemberEmailParams{HouseholdID: a.HouseholdID, Email: e})
	if err != nil {
		return Invite{}, err
	}
	if member {
		return Invite{}, conflict("%s is already a member", e)
	}
	inv, err := s.q.CreateHouseholdInvite(ctx, dbq.CreateHouseholdInviteParams{HouseholdID: a.HouseholdID, Email: e, InvitedBy: &a.UserID})
	if err != nil {
		return Invite{}, dbErr(err, "invite")
	}
	return Invite{Email: inv.Email, CreatedAt: inv.CreatedAt}, nil
}

// CancelInvite deletes a pending invite. Owner only.
func (s *Service) CancelInvite(ctx context.Context, a Actor, email string) error {
	if err := requireOwner(a); err != nil {
		return err
	}
	n, err := s.q.DeleteHouseholdInvite(ctx, dbq.DeleteHouseholdInviteParams{HouseholdID: a.HouseholdID, Email: strings.TrimSpace(email)})
	if err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("invite: %w", ErrNotFound)
	}
	return nil
}

// RemoveMember takes a member out of the household. Owner only; the owner can't be removed. Their
// rows stay, still attributed to them. Their sessions and tokens lose access on the next request.
func (s *Service) RemoveMember(ctx context.Context, a Actor, userID uuid.UUID) error {
	if err := requireOwner(a); err != nil {
		return err
	}
	if userID == a.UserID {
		return conflict("the owner can't be removed")
	}
	n, err := s.q.RemoveHouseholdMember(ctx, dbq.RemoveHouseholdMemberParams{HouseholdID: a.HouseholdID, UserID: userID})
	if err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("member: %w", ErrNotFound)
	}
	return nil
}

// LeaveHousehold takes the caller out of their active household. Owners can't leave. The caller
// moves to their oldest remaining household; with none left, they get a new one on next sign-in.
func (s *Service) LeaveHousehold(ctx context.Context, a Actor) error {
	if a.Role == RoleOwner {
		return conflict("the owner can't leave the household")
	}
	return s.inTx(ctx, func(q *dbq.Queries) error {
		n, err := q.RemoveHouseholdMember(ctx, dbq.RemoveHouseholdMemberParams{HouseholdID: a.HouseholdID, UserID: a.UserID})
		if err != nil {
			return err
		}
		if n == 0 {
			return fmt.Errorf("member: %w", ErrNotFound)
		}
		return q.SetActiveHousehold(ctx, dbq.SetActiveHouseholdParams{HouseholdID: nil, ID: a.UserID})
	})
}

// Households lists every household the caller belongs to.
func (s *Service) Households(ctx context.Context, a Actor) ([]MembershipSummary, error) {
	rows, err := s.q.ListUserHouseholds(ctx, a.UserID)
	if err != nil {
		return nil, err
	}
	out := make([]MembershipSummary, len(rows))
	for i, r := range rows {
		out[i] = MembershipSummary{ID: r.ID, Name: r.Name, Role: r.Role, MemberCount: int(r.MemberCount), Active: r.ID == a.HouseholdID}
	}
	return out, nil
}

// SwitchHousehold makes another of the caller's households the active one, for all their
// sessions and MCP tokens.
func (s *Service) SwitchHousehold(ctx context.Context, a Actor, householdID uuid.UUID) error {
	ok, err := s.q.IsHouseholdMember(ctx, dbq.IsHouseholdMemberParams{HouseholdID: householdID, UserID: a.UserID})
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("household: %w", ErrNotFound)
	}
	return s.q.SetActiveHousehold(ctx, dbq.SetActiveHouseholdParams{HouseholdID: &householdID, ID: a.UserID})
}
