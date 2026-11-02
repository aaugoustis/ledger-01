// Package oauth is the OAuth 2.1 authorization server that MCP clients (such as Claude) use to get
// read-only access to a household's ledger. Users sign in with the existing Google session and
// approve the client on the web app's consent page; the client then exchanges a one-time code
// (PKCE S256) for an opaque access token and a rotating refresh token. Only token hashes are stored.
package oauth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"net/url"
	"time"

	"github.com/franco-roura/ledger-01/backend/internal/db/dbq"
	"github.com/franco-roura/ledger-01/backend/internal/ledger"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ScopeRead is the only scope: read every ledger tool, change nothing.
const ScopeRead = "ledger:read"

const (
	requestTTL = 10 * time.Minute
	codeTTL    = 2 * time.Minute
	accessTTL  = time.Hour
	refreshTTL = 60 * 24 * time.Hour
	// How often a grant's last-used time is written back.
	touchEvery = time.Hour
)

// ErrInvalidGrant covers every unusable code or refresh token, so callers learn nothing more.
var ErrInvalidGrant = errors.New("invalid grant")

// ErrNoToken means the access token is missing, unknown, expired or revoked.
var ErrNoToken = errors.New("no valid access token")

type Store struct {
	pool *pgxpool.Pool
	q    *dbq.Queries
	now  func() time.Time
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool, q: dbq.New(pool), now: time.Now}
}

func (s *Store) inTx(ctx context.Context, fn func(q *dbq.Queries) error) error {
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

func hashToken(token string) []byte {
	sum := sha256.Sum256([]byte(token))
	return sum[:]
}

func newToken() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

// pkceMatches checks an S256 code challenge against the verifier.
func pkceMatches(challenge, verifier string) bool {
	if len(verifier) < 43 || len(verifier) > 128 {
		return false
	}
	sum := sha256.Sum256([]byte(verifier))
	return subtle.ConstantTimeCompare([]byte(base64.RawURLEncoding.EncodeToString(sum[:])), []byte(challenge)) == 1
}

type Client struct {
	ID           uuid.UUID
	Name         string
	RedirectURIs []string
	CreatedAt    time.Time
}

func (s *Store) RegisterClient(ctx context.Context, name string, redirectURIs []string) (Client, error) {
	c, err := s.q.CreateOAuthClient(ctx, dbq.CreateOAuthClientParams{
		ID: uuid.Must(uuid.NewV7()), Name: name, RedirectUris: redirectURIs,
	})
	if err != nil {
		return Client{}, err
	}
	return Client{ID: c.ID, Name: c.Name, RedirectURIs: c.RedirectUris, CreatedAt: c.CreatedAt}, nil
}

// Client returns a registered client, or ledger.ErrNotFound.
func (s *Store) Client(ctx context.Context, id uuid.UUID) (Client, error) {
	c, err := s.q.GetOAuthClient(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return Client{}, ledger.ErrNotFound
	}
	if err != nil {
		return Client{}, err
	}
	return Client{ID: c.ID, Name: c.Name, RedirectURIs: c.RedirectUris, CreatedAt: c.CreatedAt}, nil
}

// Request is a validated /oauth/authorize call waiting for the user's decision.
type Request struct {
	ClientID      uuid.UUID
	RedirectURI   string
	CodeChallenge string
	Scope         string
	Resource      string
	State         string
}

func (s *Store) CreateRequest(ctx context.Context, r Request) (uuid.UUID, error) {
	id := uuid.Must(uuid.NewV7())
	return id, s.q.CreateOAuthRequest(ctx, dbq.CreateOAuthRequestParams{
		ID: id, ClientID: r.ClientID, RedirectUri: r.RedirectURI, CodeChallenge: r.CodeChallenge,
		Scope: r.Scope, Resource: r.Resource, State: r.State, ExpiresAt: s.now().Add(requestTTL),
	})
}

// Pending is what the consent page shows.
type Pending struct {
	ID           uuid.UUID
	ClientName   string
	RedirectHost string
	Scope        string
}

// PendingRequest returns a request still waiting for a decision, or ledger.ErrNotFound.
func (s *Store) PendingRequest(ctx context.Context, id uuid.UUID) (Pending, error) {
	r, err := s.q.GetPendingOAuthRequest(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return Pending{}, ledger.ErrNotFound
	}
	if err != nil {
		return Pending{}, err
	}
	return Pending{ID: r.ID, ClientName: r.ClientName, RedirectHost: redirectHost(r.RedirectUri), Scope: r.Scope}, nil
}

func redirectHost(uri string) string {
	u, err := url.Parse(uri)
	if err != nil {
		return ""
	}
	return u.Host
}

// redirectWith adds query parameters to the client's redirect URI.
func redirectWith(uri string, params url.Values) string {
	u, err := url.Parse(uri)
	if err != nil {
		return uri
	}
	q := u.Query()
	for k, vs := range params {
		for _, v := range vs {
			q.Set(k, v)
		}
	}
	u.RawQuery = q.Encode()
	return u.String()
}

// Approve records the user's consent and returns the client's redirect URI carrying a one-time code.
func (s *Store) Approve(ctx context.Context, userID, requestID uuid.UUID, issuer string) (string, error) {
	code := newToken()
	r, err := s.q.ApproveOAuthRequest(ctx, dbq.ApproveOAuthRequestParams{
		ID: requestID, UserID: &userID, CodeHash: hashToken(code), ExpiresAt: s.now().Add(codeTTL),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ledger.ErrNotFound
	}
	if err != nil {
		return "", err
	}
	p := url.Values{"code": {code}, "iss": {issuer}}
	if r.State != "" {
		p.Set("state", r.State)
	}
	return redirectWith(r.RedirectUri, p), nil
}

// Deny drops the request and returns the client's redirect URI carrying access_denied.
func (s *Store) Deny(ctx context.Context, requestID uuid.UUID, issuer string) (string, error) {
	r, err := s.q.DeleteOAuthRequest(ctx, requestID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ledger.ErrNotFound
	}
	if err != nil {
		return "", err
	}
	p := url.Values{"error": {"access_denied"}, "iss": {issuer}}
	if r.State != "" {
		p.Set("state", r.State)
	}
	return redirectWith(r.RedirectUri, p), nil
}

// Tokens is a token endpoint response.
type Tokens struct {
	Access    string
	Refresh   string
	ExpiresIn time.Duration
	Scope     string
}

func (s *Store) issue(ctx context.Context, q *dbq.Queries, grantID uuid.UUID, scope string) (Tokens, error) {
	t := Tokens{Access: newToken(), Refresh: newToken(), ExpiresIn: accessTTL, Scope: scope}
	now := s.now()
	if err := q.CreateOAuthToken(ctx, dbq.CreateOAuthTokenParams{
		TokenHash: hashToken(t.Access), GrantID: grantID, Kind: "access", ExpiresAt: now.Add(accessTTL),
	}); err != nil {
		return Tokens{}, err
	}
	if err := q.CreateOAuthToken(ctx, dbq.CreateOAuthTokenParams{
		TokenHash: hashToken(t.Refresh), GrantID: grantID, Kind: "refresh", ExpiresAt: now.Add(refreshTTL),
	}); err != nil {
		return Tokens{}, err
	}
	return t, nil
}

// ExchangeCode turns an approved code into tokens. The code is spent even when the exchange fails.
func (s *Store) ExchangeCode(ctx context.Context, clientID uuid.UUID, code, redirectURI, verifier string) (Tokens, error) {
	if code == "" {
		return Tokens{}, ErrInvalidGrant
	}
	r, err := s.q.ConsumeOAuthCode(ctx, hashToken(code))
	if errors.Is(err, pgx.ErrNoRows) {
		return Tokens{}, ErrInvalidGrant
	}
	if err != nil {
		return Tokens{}, err
	}
	if r.ClientID != clientID || r.UserID == nil || r.RedirectUri != redirectURI || !pkceMatches(r.CodeChallenge, verifier) {
		return Tokens{}, ErrInvalidGrant
	}
	var out Tokens
	err = s.inTx(ctx, func(q *dbq.Queries) error {
		grantID := uuid.Must(uuid.NewV7())
		if err := q.CreateOAuthGrant(ctx, dbq.CreateOAuthGrantParams{
			ID: grantID, ClientID: r.ClientID, UserID: *r.UserID, Scope: r.Scope, Resource: r.Resource,
		}); err != nil {
			return err
		}
		out, err = s.issue(ctx, q, grantID, r.Scope)
		return err
	})
	return out, err
}

// Refresh rotates a refresh token. Presenting a token that was already rotated means it leaked,
// so the whole grant is revoked.
func (s *Store) Refresh(ctx context.Context, clientID uuid.UUID, refresh string) (Tokens, error) {
	if refresh == "" {
		return Tokens{}, ErrInvalidGrant
	}
	var out Tokens
	err := s.inTx(ctx, func(q *dbq.Queries) error {
		g, err := q.UseOAuthRefreshToken(ctx, hashToken(refresh))
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrInvalidGrant
		}
		if err != nil {
			return err
		}
		if g.ClientID != clientID {
			return ErrInvalidGrant
		}
		out, err = s.issue(ctx, q, g.GrantID, g.Scope)
		return err
	})
	if errors.Is(err, ErrInvalidGrant) {
		if grantID, gerr := s.q.GetOAuthRefreshGrant(ctx, hashToken(refresh)); gerr == nil {
			if _, rerr := s.q.RevokeOAuthGrant(ctx, grantID); rerr != nil {
				return Tokens{}, rerr
			}
		}
	}
	return out, err
}

// Access is what an access token grants.
type Access struct {
	Actor    ledger.Actor
	GrantID  uuid.UUID
	ClientID uuid.UUID
	Scope    string
}

// Resolve returns the access behind a bearer token, or ErrNoToken.
func (s *Store) Resolve(ctx context.Context, token string) (Access, error) {
	if token == "" {
		return Access{}, ErrNoToken
	}
	r, err := s.q.GetOAuthAccess(ctx, hashToken(token))
	if errors.Is(err, pgx.ErrNoRows) {
		return Access{}, ErrNoToken
	}
	if err != nil {
		return Access{}, err
	}
	if r.LastUsedAt == nil || s.now().Sub(*r.LastUsedAt) > touchEvery {
		// Best effort: a failed write must not fail the request.
		_ = s.q.TouchOAuthGrant(ctx, r.GrantID)
	}
	return Access{
		Actor:   ledger.Actor{UserID: r.UserID, HouseholdID: r.HouseholdID, Role: r.Role},
		GrantID: r.GrantID, ClientID: r.ClientID, Scope: r.Scope,
	}, nil
}

// RevokeToken revokes the grant behind an access or refresh token, if the client owns it. Unknown
// tokens are ignored, as RFC 7009 asks.
func (s *Store) RevokeToken(ctx context.Context, clientID uuid.UUID, token string) error {
	g, err := s.q.GetOAuthTokenGrant(ctx, hashToken(token))
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && g.ClientID != clientID) {
		return nil
	}
	if err != nil {
		return err
	}
	_, err = s.q.RevokeOAuthGrant(ctx, g.GrantID)
	return err
}

// Connection is a grant the user hasn't revoked.
type Connection struct {
	ID           uuid.UUID
	ClientName   string
	RedirectHost string
	Scope        string
	CreatedAt    time.Time
	LastUsedAt   *time.Time
}

func (s *Store) Connections(ctx context.Context, userID uuid.UUID) ([]Connection, error) {
	rows, err := s.q.ListOAuthGrants(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]Connection, len(rows))
	for i, r := range rows {
		host := ""
		if len(r.RedirectUris) > 0 {
			host = redirectHost(r.RedirectUris[0])
		}
		out[i] = Connection{
			ID: r.ID, ClientName: r.ClientName, RedirectHost: host, Scope: r.Scope,
			CreatedAt: r.CreatedAt, LastUsedAt: r.LastUsedAt,
		}
	}
	return out, nil
}

// RevokeConnection revokes one of the user's grants, or returns ledger.ErrNotFound.
func (s *Store) RevokeConnection(ctx context.Context, userID, grantID uuid.UUID) error {
	n, err := s.q.RevokeOAuthGrantForUser(ctx, dbq.RevokeOAuthGrantForUserParams{ID: grantID, UserID: userID})
	if err != nil {
		return err
	}
	if n == 0 {
		return ledger.ErrNotFound
	}
	return nil
}

// DeleteExpired removes spent requests, expired tokens, dead grants and unused registrations.
func (s *Store) DeleteExpired(ctx context.Context) (int64, error) {
	return s.q.DeleteExpiredOAuth(ctx)
}
