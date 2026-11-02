// Package ledger holds the domain services: wallets, categories, labels, transactions and
// transfers. Every method takes the calling Actor and scopes all reads and writes to the
// actor's household. The Connect handlers in internal/rpc are thin adapters over this package.
package ledger

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/franco-roura/ledger-01/backend/internal/db/dbq"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Actor is the authenticated caller, resolved from the session (never from request data).
type Actor struct {
	UserID      uuid.UUID
	HouseholdID uuid.UUID
	// RoleOwner or RoleMember in HouseholdID.
	Role string
}

const (
	RoleOwner  = "owner"
	RoleMember = "member"
)

// requireOwner allows household administration (members, settings, restore) to owners only.
func requireOwner(a Actor) error {
	if a.Role != RoleOwner {
		return ErrForbidden
	}
	return nil
}

type Service struct {
	pool *pgxpool.Pool
	q    *dbq.Queries
}

func New(pool *pgxpool.Pool) *Service {
	return &Service{pool: pool, q: dbq.New(pool)}
}

// inTx runs fn inside a transaction, committing on success.
func (s *Service) inTx(ctx context.Context, fn func(q *dbq.Queries) error) error {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // no-op after commit
	if err := fn(s.q.WithTx(tx)); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func newID() uuid.UUID { return uuid.Must(uuid.NewV7()) }

// Today is the current date in UTC: the default for callers that don't send their local date.
// Rows dated after a caller's today are upcoming and never count in balances or totals.
func Today() time.Time {
	y, m, d := time.Now().UTC().Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

// orToday returns d, or Today() when d is zero.
func orToday(d time.Time) time.Time {
	if d.IsZero() {
		return Today()
	}
	return d
}

// --- errors ------------------------------------------------------------------------------------

// ErrNotFound means the entity doesn't exist in the caller's household.
var ErrNotFound = errors.New("not found")

// ErrForbidden means the caller's role doesn't allow the action.
var ErrForbidden = errors.New("only the household owner can do this")

// InvalidError is a validation failure on a specific field.
type InvalidError struct {
	Field string
	Msg   string
}

func (e *InvalidError) Error() string { return fmt.Sprintf("%s: %s", e.Field, e.Msg) }

func invalid(field, format string, args ...any) error {
	return &InvalidError{Field: field, Msg: fmt.Sprintf(format, args...)}
}

// ConflictError means the request is valid but clashes with existing state.
type ConflictError struct{ Msg string }

func (e *ConflictError) Error() string { return e.Msg }

func conflict(format string, args ...any) error {
	return &ConflictError{Msg: fmt.Sprintf(format, args...)}
}

// dbErr translates driver errors into domain errors.
func dbErr(err error, what string) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("%s: %w", what, ErrNotFound)
	}
	var pg *pgconn.PgError
	if errors.As(err, &pg) {
		switch pg.Code {
		case "23505": // unique_violation
			return conflict("%s already exists", what)
		case "23514": // check_violation
			return invalid(what, "violates constraint %s", pg.ConstraintName)
		case "23503": // foreign_key_violation
			return invalid(what, "references something that doesn't exist")
		}
	}
	return err
}

func uniqueIDs(ids []uuid.UUID) []uuid.UUID {
	seen := make(map[uuid.UUID]struct{}, len(ids))
	out := make([]uuid.UUID, 0, len(ids))
	for _, id := range ids {
		if _, ok := seen[id]; !ok {
			seen[id] = struct{}{}
			out = append(out, id)
		}
	}
	return out
}
