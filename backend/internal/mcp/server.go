// Package mcp serves the ledger to MCP clients (such as Claude) as read-only tools. It runs inside
// the API on POST /mcp, stateless and with plain JSON responses, so it works behind API Gateway
// like every other route. The caller is the OAuth access token's user (see internal/oauth).
package mcp

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/franco-roura/ledger-01/backend/internal/auth"
	"github.com/franco-roura/ledger-01/backend/internal/ledger"
	"github.com/franco-roura/ledger-01/backend/internal/money"
	"github.com/google/uuid"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// MaxBodyBytes caps a request, like the Connect APIs.
const MaxBodyBytes = 1 << 20

const instructions = `Ledger/01 is the user's personal finance ledger: wallets (accounts) holding expenses,
income and transfers, grouped by categories and labels. Everything here is read-only.

- Money is integer minor units plus an ISO 4217 currency; "display" is the same amount as a decimal string.
- Pass "today" as the user's local date (YYYY-MM-DD). Rows dated after it are upcoming and never count in totals.
- Merchants and descriptions live in the transaction note, so search notes for things like "pizza".
- Start with get_profile and list_categories to learn the base currency and map category names to ids.
- spending_summary totals in the base currency across wallets; use it to compare periods.`

// Server holds the ledger service and reference data shared by every request.
type Server struct {
	svc *ledger.Service
	log *slog.Logger

	mu         sync.Mutex
	minorUnits map[string]int
}

// New returns the http.Handler for /mcp. Put it behind oauth's bearer middleware.
func New(svc *ledger.Service, log *slog.Logger) http.Handler {
	s := &Server{svc: svc, log: log}
	srv := sdk.NewServer(&sdk.Implementation{Name: "ledger-01", Title: "Ledger/01", Version: "1"},
		&sdk.ServerOptions{Instructions: instructions})
	s.addTools(srv)
	return sdk.NewStreamableHTTPHandler(func(*http.Request) *sdk.Server { return srv }, &sdk.StreamableHTTPOptions{
		Stateless:           true,
		JSONResponse:        true,
		MaxRequestBodyBytes: MaxBodyBytes,
	})
}

// toolError turns a domain error into a message for the model, hiding unexpected ones.
func (s *Server) toolError(ctx context.Context, err error) error {
	var inv *ledger.InvalidError
	switch {
	case errors.As(err, &inv):
		return inv
	case errors.Is(err, ledger.ErrNotFound):
		return errors.New("not found (check the id with a list tool)")
	default:
		s.log.ErrorContext(ctx, "mcp tool failed", "err", err)
		return errors.New("internal error")
	}
}

// tool wraps a handler with the caller's identity and error mapping.
func tool[In, Out any](s *Server, fn func(context.Context, ledger.Actor, In) (Out, error)) sdk.ToolHandlerFor[In, Out] {
	return func(ctx context.Context, _ *sdk.CallToolRequest, in In) (*sdk.CallToolResult, Out, error) {
		var zero Out
		a, err := auth.ActorFrom(ctx)
		if err != nil {
			return nil, zero, err
		}
		out, err := fn(ctx, a, in)
		if err != nil {
			return nil, zero, s.toolError(ctx, err)
		}
		return nil, out, nil
	}
}

// Money is an amount as the API keeps it, plus a decimal string for reading.
type Money struct {
	Minor    int64  `json:"minor" jsonschema:"amount in the currency's minor units (cents)"`
	Currency string `json:"currency"`
	Display  string `json:"display" jsonschema:"the amount as a decimal string, e.g. 1234.50"`
}

func (s *Server) loadMinorUnits(ctx context.Context) (map[string]int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.minorUnits != nil {
		return s.minorUnits, nil
	}
	cs, err := s.svc.Currencies(ctx)
	if err != nil {
		return nil, err
	}
	m := make(map[string]int, len(cs))
	for _, c := range cs {
		m[c.Code] = c.MinorUnits
	}
	s.minorUnits = m
	return m, nil
}

// moneyFmt formats amounts once the currency table is loaded.
type moneyFmt map[string]int

func (s *Server) moneyFmt(ctx context.Context) (moneyFmt, error) {
	m, err := s.loadMinorUnits(ctx)
	return moneyFmt(m), err
}

func (m moneyFmt) of(minor int64, currency string) Money {
	units, ok := m[currency]
	if !ok {
		units = 2
	}
	return Money{Minor: minor, Currency: currency, Display: money.FormatMinor(minor, units)}
}

func parseDate(field, s string) (time.Time, error) {
	t, err := money.ParseDate(strings.TrimSpace(s))
	if err != nil {
		return time.Time{}, &ledger.InvalidError{Field: field, Msg: "must be a date like 2026-10-30"}
	}
	return t, nil
}

func parseToday(s string) (time.Time, error) {
	if strings.TrimSpace(s) == "" {
		return ledger.Today(), nil
	}
	return parseDate("today", s)
}

func parseIDs(field string, ss []string) ([]uuid.UUID, error) {
	out := make([]uuid.UUID, 0, len(ss))
	for _, s := range ss {
		id, err := uuid.Parse(strings.TrimSpace(s))
		if err != nil {
			return nil, &ledger.InvalidError{Field: field, Msg: fmt.Sprintf("%q is not an id; get ids from the list tools", s)}
		}
		out = append(out, id)
	}
	return out, nil
}

func parseBucket(s string) (ledger.Bucket, error) {
	switch s {
	case "", "month":
		return ledger.BucketMonth, nil
	case "week":
		return ledger.BucketWeek, nil
	case "day":
		return ledger.BucketDay, nil
	}
	return 0, &ledger.InvalidError{Field: "bucket", Msg: "must be day, week or month"}
}

// checkBuckets keeps series short enough to read.
func checkBuckets(from, to time.Time, b ledger.Bucket) error {
	days := int(to.Sub(from).Hours()/24) + 1
	n := map[ledger.Bucket]int{ledger.BucketDay: days, ledger.BucketWeek: days / 7, ledger.BucketMonth: days / 30}[b]
	if n > 400 {
		return &ledger.InvalidError{Field: "bucket", Msg: "too many buckets for this period; use a coarser bucket or a shorter period"}
	}
	return nil
}

func date(t time.Time) string { return money.FormatDate(t) }

func optDate(t *time.Time) string {
	if t == nil {
		return ""
	}
	return money.FormatDate(*t)
}
