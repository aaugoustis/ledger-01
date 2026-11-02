package auth

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/franco-roura/ledger-01/backend/internal/config"
	"github.com/franco-roura/ledger-01/backend/internal/ledger"
	"golang.org/x/oauth2"
)

const (
	googleIssuer  = "https://accounts.google.com"
	googleJWKSURL = "https://www.googleapis.com/oauth2/v3/certs"
	// Short-lived cookie carrying state, PKCE verifier, nonce and the return path.
	flowCookie = "ledger_oauth"
	flowTTL    = 10 * time.Minute
)

// googleEndpoint is hard-coded so startup needs no network (no OIDC discovery on cold start).
var googleEndpoint = oauth2.Endpoint{
	AuthURL:   "https://accounts.google.com/o/oauth2/v2/auth",
	TokenURL:  "https://oauth2.googleapis.com/token",
	AuthStyle: oauth2.AuthStyleInParams,
}

// Users provisions the user behind a verified identity.
type Users interface {
	EnsureUser(ctx context.Context, p ledger.Profile) (ledger.Actor, error)
}

type Handler struct {
	cfg      config.Config
	sessions *Sessions
	users    Users
	log      *slog.Logger
	oauth    *oauth2.Config
	verifier *oidc.IDTokenVerifier
}

func NewHandler(cfg config.Config, sessions *Sessions, users Users, log *slog.Logger) *Handler {
	// Signing keys are fetched lazily on first verification and cached.
	keys := oidc.NewRemoteKeySet(context.Background(), googleJWKSURL)
	return &Handler{
		cfg:      cfg,
		sessions: sessions,
		users:    users,
		log:      log,
		oauth: &oauth2.Config{
			ClientID:     cfg.GoogleClientID,
			ClientSecret: cfg.GoogleClientSecret,
			Endpoint:     googleEndpoint,
			RedirectURL:  cfg.OAuthRedirectURL(),
			Scopes:       []string{oidc.ScopeOpenID, "email", "profile"},
		},
		verifier: oidc.NewVerifier(googleIssuer, keys, &oidc.Config{ClientID: cfg.GoogleClientID}),
	}
}

// Register mounts the /auth routes.
func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /auth/google/start", h.start)
	mux.HandleFunc("GET /auth/google/callback", h.callback)
	mux.HandleFunc("POST /auth/logout", h.logout)
	if h.cfg.DevLoginEnabled() {
		mux.HandleFunc("POST /auth/dev/login", h.devLogin)
	}
}

type flowState struct {
	State    string `json:"s"`
	Verifier string `json:"v"`
	Nonce    string `json:"n"`
	Next     string `json:"x"`
}

func randomString() string {
	b := make([]byte, 24)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

// safeNext only allows same-site absolute paths, never "//host" or schemes. Browsers drop tabs and
// newlines from URLs and read a backslash as "/", so a tab or backslash after the first slash would
// leave the site: any control character or backslash is refused.
func safeNext(next string) string {
	const fallback = "/dashboard"
	if !strings.HasPrefix(next, "/") || strings.HasPrefix(next, "//") || strings.ContainsAny(next, "\\") {
		return fallback
	}
	for _, r := range next {
		if r < 0x20 || r == 0x7f {
			return fallback
		}
	}
	if u, err := url.Parse(next); err != nil || u.Scheme != "" || u.Host != "" {
		return fallback
	}
	return next
}

func (h *Handler) start(w http.ResponseWriter, r *http.Request) {
	if h.cfg.GoogleClientID == "" {
		http.Error(w, "Google sign-in is not configured (GOOGLE_CLIENT_ID)", http.StatusServiceUnavailable)
		return
	}
	fs := flowState{State: randomString(), Verifier: oauth2.GenerateVerifier(), Nonce: randomString(), Next: safeNext(r.URL.Query().Get("next"))}
	raw, _ := json.Marshal(fs)
	http.SetCookie(w, &http.Cookie{
		Name: flowCookie, Value: base64.RawURLEncoding.EncodeToString(raw), Path: "/auth",
		MaxAge: int(flowTTL.Seconds()), HttpOnly: true, Secure: h.cfg.SecureCookies(), SameSite: http.SameSiteLaxMode,
	})
	url := h.oauth.AuthCodeURL(fs.State,
		oauth2.S256ChallengeOption(fs.Verifier),
		oidc.Nonce(fs.Nonce),
		oauth2.SetAuthURLParam("prompt", "select_account"),
	)
	http.Redirect(w, r, url, http.StatusFound)
}

func (h *Handler) loginError(w http.ResponseWriter, r *http.Request, code string) {
	http.Redirect(w, r, "/login?error="+url.QueryEscape(code), http.StatusFound)
}

func (h *Handler) callback(w http.ResponseWriter, r *http.Request) {
	c, err := r.Cookie(flowCookie)
	http.SetCookie(w, &http.Cookie{Name: flowCookie, Path: "/auth", MaxAge: -1, HttpOnly: true, Secure: h.cfg.SecureCookies()})
	if err != nil {
		http.Error(w, "sign-in expired, try again", http.StatusBadRequest)
		return
	}
	var fs flowState
	raw, err := base64.RawURLEncoding.DecodeString(c.Value)
	if err != nil || json.Unmarshal(raw, &fs) != nil || fs.State == "" {
		http.Error(w, "invalid sign-in state", http.StatusBadRequest)
		return
	}
	q := r.URL.Query()
	if subtle.ConstantTimeCompare([]byte(q.Get("state")), []byte(fs.State)) != 1 {
		http.Error(w, "invalid sign-in state", http.StatusBadRequest)
		return
	}
	if e := q.Get("error"); e != "" {
		h.loginError(w, r, "cancelled")
		return
	}

	ctx := r.Context()
	tok, err := h.oauth.Exchange(ctx, q.Get("code"), oauth2.VerifierOption(fs.Verifier))
	if err != nil {
		h.log.WarnContext(ctx, "oauth exchange failed", "err", err)
		h.loginError(w, r, "exchange_failed")
		return
	}
	rawID, _ := tok.Extra("id_token").(string)
	idTok, err := h.verifier.Verify(ctx, rawID)
	if err != nil || idTok.Nonce != fs.Nonce {
		h.log.WarnContext(ctx, "id token rejected", "err", err)
		h.loginError(w, r, "invalid_token")
		return
	}
	var claims struct {
		Email         string `json:"email"`
		EmailVerified bool   `json:"email_verified"`
		Name          string `json:"name"`
		Picture       string `json:"picture"`
	}
	if err := idTok.Claims(&claims); err != nil || claims.Email == "" || !claims.EmailVerified {
		h.loginError(w, r, "email_unverified")
		return
	}
	h.finish(w, r, ledger.Profile{Subject: idTok.Subject, Email: claims.Email, Name: claims.Name, AvatarURL: claims.Picture}, fs.Next, true)
}

// finish checks the allowlist, provisions the user and starts a session.
func (h *Handler) finish(w http.ResponseWriter, r *http.Request, p ledger.Profile, next string, redirect bool) {
	ctx := r.Context()
	if !h.cfg.EmailAllowed(p.Email) {
		h.log.InfoContext(ctx, "sign-in rejected: email not allowed", "email", p.Email)
		if redirect {
			h.loginError(w, r, "not_allowed")
		} else {
			http.Error(w, "email not allowed", http.StatusForbidden)
		}
		return
	}
	actor, err := h.users.EnsureUser(ctx, p)
	if err != nil {
		h.log.ErrorContext(ctx, "provision user", "err", err)
		http.Error(w, "could not sign in", http.StatusInternalServerError)
		return
	}
	token, expires, err := h.sessions.Create(ctx, actor.UserID, r.UserAgent())
	if err != nil {
		h.log.ErrorContext(ctx, "create session", "err", err)
		http.Error(w, "could not sign in", http.StatusInternalServerError)
		return
	}
	setSessionCookie(w, token, expires, h.cfg.SecureCookies())
	if redirect {
		http.Redirect(w, r, safeNext(next), http.StatusFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) logout(w http.ResponseWriter, r *http.Request) {
	if err := h.sessions.Revoke(r.Context(), TokenFromHeader(r.Header)); err != nil {
		h.log.ErrorContext(r.Context(), "revoke session", "err", err)
	}
	clearSessionCookie(w, h.cfg.SecureCookies())
	w.WriteHeader(http.StatusNoContent)
}

// devLogin signs in any allowlisted email without Google. Only mounted in development with
// AUTH_DEV_LOGIN=1; used by Playwright.
func (h *Handler) devLogin(w http.ResponseWriter, r *http.Request) {
	if !h.cfg.DevLoginEnabled() {
		http.NotFound(w, r)
		return
	}
	var body struct {
		Email string `json:"email"`
		Name  string `json:"name"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&body); err != nil || body.Email == "" {
		http.Error(w, "expected JSON {\"email\": \"...\"}", http.StatusBadRequest)
		return
	}
	email := strings.ToLower(strings.TrimSpace(body.Email))
	name := body.Name
	if name == "" {
		name, _, _ = strings.Cut(email, "@")
	}
	h.finish(w, r, ledger.Profile{Subject: "dev:" + email, Email: email, Name: name}, "", false)
}
