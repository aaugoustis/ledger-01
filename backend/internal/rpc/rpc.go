// Package rpc adapts the ledger domain services to the generated Connect service interfaces:
// it converts protobuf messages to domain types and domain errors to Connect codes.
package rpc

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"connectrpc.com/connect"
	"github.com/franco-roura/ledger-01/backend/internal/auth"
	"github.com/franco-roura/ledger-01/backend/internal/ledger"
	"github.com/franco-roura/ledger-01/backend/internal/money"
	"github.com/google/uuid"
)

// ReadMaxBytes caps every request except ImportService's (see ImportReadMaxBytes). Connect reads
// the body before the session interceptor runs, so the cap applies to anonymous callers too.
const ReadMaxBytes = 1 << 20

// Server implements every ledger.v1 service on top of one ledger.Service.
type Server struct {
	svc *ledger.Service
	log *slog.Logger
	// emailAllowed reports whether the deployment's allowlist lets an email sign in.
	emailAllowed func(string) bool
}

func New(svc *ledger.Service, log *slog.Logger, emailAllowed func(string) bool) *Server {
	return &Server{svc: svc, log: log, emailAllowed: emailAllowed}
}

func (s *Server) actor(ctx context.Context) (ledger.Actor, error) { return auth.ActorFrom(ctx) }

// fail converts a domain error into a Connect error, logging unexpected ones.
func (s *Server) fail(ctx context.Context, err error) error {
	var inv *ledger.InvalidError
	var conf *ledger.ConflictError
	var ce *connect.Error
	switch {
	case errors.As(err, &ce):
		return ce
	case errors.As(err, &inv):
		return connect.NewError(connect.CodeInvalidArgument, inv)
	case errors.As(err, &conf):
		return connect.NewError(connect.CodeFailedPrecondition, conf)
	case errors.Is(err, ledger.ErrNotFound):
		return connect.NewError(connect.CodeNotFound, err)
	case errors.Is(err, ledger.ErrForbidden):
		return connect.NewError(connect.CodePermissionDenied, err)
	case errors.Is(err, context.Canceled):
		return connect.NewError(connect.CodeCanceled, err)
	default:
		s.log.ErrorContext(ctx, "rpc failed", "err", err)
		return connect.NewError(connect.CodeInternal, errors.New("internal error"))
	}
}

func parseID(field, s string) (uuid.UUID, error) {
	id, err := uuid.Parse(s)
	if err != nil {
		return uuid.Nil, &ledger.InvalidError{Field: field, Msg: "must be a UUID"}
	}
	return id, nil
}

func parseOptID(field string, s *string) (*uuid.UUID, error) {
	if s == nil || *s == "" {
		return nil, nil
	}
	id, err := parseID(field, *s)
	return &id, err
}

func parseIDs(field string, ss []string) ([]uuid.UUID, error) {
	out := make([]uuid.UUID, 0, len(ss))
	for i, s := range ss {
		id, err := parseID(fmt.Sprintf("%s[%d]", field, i), s)
		if err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, nil
}

func idStrings(ids []uuid.UUID) []string {
	out := make([]string, len(ids))
	for i, id := range ids {
		out[i] = id.String()
	}
	return out
}

func parseDate(field, s string) (time.Time, error) {
	t, err := money.ParseDate(s)
	if err != nil {
		return time.Time{}, &ledger.InvalidError{Field: field, Msg: err.Error()}
	}
	return t, nil
}

// parseToday reads the caller's local date, defaulting to today in UTC.
func parseToday(s string) (time.Time, error) {
	if s == "" {
		return ledger.Today(), nil
	}
	return parseDate("today", s)
}
