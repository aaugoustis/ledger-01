package ledger

import (
	"context"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/franco-roura/ledger-01/backend/internal/db/dbq"
	"github.com/google/uuid"
)

type Wallet struct {
	ID               uuid.UUID
	Name             string
	Currency         string
	MinorUnits       int
	InitialBalance   int64
	Balance          int64
	Hidden           bool
	Position         int32
	Archived         bool
	TransactionCount int64
	CreatedAt        time.Time
}

// walletRow is satisfied by both ListWalletsRow and GetWalletRow (same columns).
func walletFrom(id uuid.UUID, name, currency string, minorUnits int16, initial, movement int64, hidden bool,
	position int32, archivedAt *time.Time, count int64, createdAt time.Time,
) Wallet {
	return Wallet{
		ID:               id,
		Name:             name,
		Currency:         strings.TrimSpace(currency),
		MinorUnits:       int(minorUnits),
		InitialBalance:   initial,
		Balance:          initial + movement,
		Hidden:           hidden,
		Position:         position,
		Archived:         archivedAt != nil,
		TransactionCount: count,
		CreatedAt:        createdAt,
	}
}

// ListWallets returns the household's wallets with their balances as of today (the caller's
// local date): upcoming rows don't count yet.
func (s *Service) ListWallets(ctx context.Context, a Actor, includeHidden, includeArchived bool, today time.Time) ([]Wallet, error) {
	rows, err := s.q.ListWallets(ctx, dbq.ListWalletsParams{
		HouseholdID: a.HouseholdID, IncludeHidden: includeHidden, IncludeArchived: includeArchived, Today: today,
	})
	if err != nil {
		return nil, err
	}
	out := make([]Wallet, len(rows))
	for i, r := range rows {
		out[i] = walletFrom(r.ID, r.Name, r.Currency, r.MinorUnits, r.InitialBalanceMinor, r.MovementMinor,
			r.Hidden, r.Position, r.ArchivedAt, r.TransactionCount, r.CreatedAt)
	}
	return out, nil
}

// GetWallet returns a wallet with its balance as of today in UTC.
func (s *Service) GetWallet(ctx context.Context, a Actor, id uuid.UUID) (Wallet, error) {
	return s.GetWalletOn(ctx, a, id, Today())
}

// GetWalletOn returns a wallet with its balance as of today (the caller's local date).
func (s *Service) GetWalletOn(ctx context.Context, a Actor, id uuid.UUID, today time.Time) (Wallet, error) {
	r, err := s.q.GetWallet(ctx, dbq.GetWalletParams{ID: id, HouseholdID: a.HouseholdID, Today: today})
	if err != nil {
		return Wallet{}, dbErr(err, "wallet")
	}
	return walletFrom(r.ID, r.Name, r.Currency, r.MinorUnits, r.InitialBalanceMinor, r.MovementMinor,
		r.Hidden, r.Position, r.ArchivedAt, r.TransactionCount, r.CreatedAt), nil
}

func cleanName(field, name string, max int) (string, error) {
	n := strings.TrimSpace(name)
	if n == "" || utf8.RuneCountInString(n) > max {
		return "", invalid(field, "must be 1-%d characters", max)
	}
	return n, nil
}

func (s *Service) CreateWallet(ctx context.Context, a Actor, name, currency string, initialBalance int64) (Wallet, error) {
	name, err := cleanName("name", name, 60)
	if err != nil {
		return Wallet{}, err
	}
	currency = strings.ToUpper(strings.TrimSpace(currency))
	if _, err := s.q.GetCurrency(ctx, currency); err != nil {
		return Wallet{}, invalid("currency", "unknown currency %q", currency)
	}
	if err := checkAmount("initial_balance_minor", abs(initialBalance), true); err != nil {
		return Wallet{}, err
	}
	pos, err := s.q.NextWalletPosition(ctx, a.HouseholdID)
	if err != nil {
		return Wallet{}, err
	}
	w, err := s.q.CreateWallet(ctx, dbq.CreateWalletParams{
		ID: newID(), HouseholdID: a.HouseholdID, Name: name, Currency: currency, InitialBalanceMinor: initialBalance, Position: pos,
	})
	if err != nil {
		return Wallet{}, dbErr(err, "wallet")
	}
	return s.GetWallet(ctx, a, w.ID)
}

type WalletPatch struct {
	Name           *string
	Hidden         *bool
	Position       *int32
	InitialBalance *int64
}

func (s *Service) UpdateWallet(ctx context.Context, a Actor, id uuid.UUID, p WalletPatch) (Wallet, error) {
	if p.Name != nil {
		n, err := cleanName("name", *p.Name, 60)
		if err != nil {
			return Wallet{}, err
		}
		p.Name = &n
	}
	if p.InitialBalance != nil {
		if err := checkAmount("initial_balance_minor", abs(*p.InitialBalance), true); err != nil {
			return Wallet{}, err
		}
	}
	if _, err := s.q.UpdateWallet(ctx, dbq.UpdateWalletParams{
		ID: id, HouseholdID: a.HouseholdID, Name: p.Name, Hidden: p.Hidden, Position: p.Position, InitialBalanceMinor: p.InitialBalance,
	}); err != nil {
		return Wallet{}, dbErr(err, "wallet")
	}
	return s.GetWallet(ctx, a, id)
}

func (s *Service) ArchiveWallet(ctx context.Context, a Actor, id uuid.UUID) error {
	n, err := s.q.ArchiveWallet(ctx, dbq.ArchiveWalletParams{ID: id, HouseholdID: a.HouseholdID})
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// DeleteWallet hard-deletes a wallet that has no transactions.
func (s *Service) DeleteWallet(ctx context.Context, a Actor, id uuid.UUID) error {
	n, err := s.q.DeleteEmptyWallet(ctx, dbq.DeleteEmptyWalletParams{ID: id, HouseholdID: a.HouseholdID})
	if err != nil {
		return err
	}
	if n == 0 {
		if _, err := s.GetWallet(ctx, a, id); err != nil {
			return err
		}
		return conflict("wallet has transactions; archive it instead")
	}
	return nil
}

// MaxAmount caps any single amount (in minor units) well inside int64.
const MaxAmount = 1_000_000_000_000_000

func checkAmount(field string, v int64, allowZero bool) error {
	if v < 0 || (!allowZero && v == 0) {
		return invalid(field, "must be greater than zero")
	}
	if v > MaxAmount {
		return invalid(field, "is too large")
	}
	return nil
}

func abs(v int64) int64 {
	if v < 0 {
		return -v
	}
	return v
}
