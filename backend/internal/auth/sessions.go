// Package auth implements Google sign-in, Postgres-backed sessions and the Connect interceptor
// that turns a session cookie into the caller's identity.
package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"net/http"
	"time"

	"github.com/franco-roura/ledger-01/backend/internal/db/dbq"
	"github.com/franco-roura/ledger-01/backend/internal/ledger"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// CookieName is the session cookie. It holds a random token; only its hash is stored.
const CookieName = "ledger_session"

// refreshEvery limits how often a session's sliding expiry is written back.
const refreshEvery = time.Hour

var ErrNoSession = errors.New("no valid session")

type Sessions struct {
	q   *dbq.Queries
	ttl time.Duration
	now func() time.Time
}

func NewSessions(q *dbq.Queries, ttl time.Duration) *Sessions {
	return &Sessions{q: q, ttl: ttl, now: time.Now}
}

func hashToken(token string) []byte {
	sum := sha256.Sum256([]byte(token))
	return sum[:]
}

// Create starts a session for the user and returns the cookie token and its expiry.
func (s *Sessions) Create(ctx context.Context, userID uuid.UUID, userAgent string) (string, time.Time, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", time.Time{}, err
	}
	token := base64.RawURLEncoding.EncodeToString(b)
	expires := s.now().Add(s.ttl)
	if len(userAgent) > 300 {
		userAgent = userAgent[:300]
	}
	err := s.q.CreateSession(ctx, dbq.CreateSessionParams{
		ID: uuid.Must(uuid.NewV7()), UserID: userID, TokenHash: hashToken(token), UserAgent: userAgent, ExpiresAt: expires,
	})
	return token, expires, err
}

// Resolve returns the actor for a session token and slides its expiry forward.
func (s *Sessions) Resolve(ctx context.Context, token string) (ledger.Actor, error) {
	if token == "" {
		return ledger.Actor{}, ErrNoSession
	}
	row, err := s.q.GetSessionByTokenHash(ctx, hashToken(token))
	if errors.Is(err, pgx.ErrNoRows) {
		return ledger.Actor{}, ErrNoSession
	}
	if err != nil {
		return ledger.Actor{}, err
	}
	if s.now().Sub(row.LastSeenAt) > refreshEvery {
		// Best effort: a failed refresh must not fail the request.
		_ = s.q.TouchSession(ctx, dbq.TouchSessionParams{ID: row.ID, ExpiresAt: s.now().Add(s.ttl)})
	}
	return ledger.Actor{UserID: row.UserID, HouseholdID: row.HouseholdID, Role: row.Role}, nil
}

// Revoke deletes the session behind a token.
func (s *Sessions) Revoke(ctx context.Context, token string) error {
	if token == "" {
		return nil
	}
	return s.q.DeleteSessionByTokenHash(ctx, hashToken(token))
}

// TokenFromHeader extracts the session token from a request's Cookie header.
func TokenFromHeader(h http.Header) string {
	c, err := (&http.Request{Header: h}).Cookie(CookieName)
	if err != nil {
		return ""
	}
	return c.Value
}

func setSessionCookie(w http.ResponseWriter, token string, expires time.Time, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    token,
		Path:     "/",
		Expires:  expires,
		MaxAge:   int(time.Until(expires).Seconds()),
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}

func clearSessionCookie(w http.ResponseWriter, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name: CookieName, Value: "", Path: "/", MaxAge: -1, HttpOnly: true, Secure: secure, SameSite: http.SameSiteLaxMode,
	})
}
