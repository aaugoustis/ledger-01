package oauth

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/franco-roura/ledger-01/backend/internal/auth"
	"github.com/franco-roura/ledger-01/backend/internal/config"
	"github.com/franco-roura/ledger-01/backend/internal/ledger"
	"github.com/google/uuid"
)

// MCPPath is the protected resource: the MCP endpoint.
const MCPPath = "/mcp"

const (
	consentPath = "/connect"
	loginPath   = "/login"
	maxBody     = 16 << 10
)

// Handler serves the OAuth endpoints and protects the MCP endpoint with bearer tokens.
type Handler struct {
	store    *Store
	sessions *auth.Sessions
	log      *slog.Logger
	issuer   string
}

func NewHandler(cfg config.Config, store *Store, sessions *auth.Sessions, log *slog.Logger) *Handler {
	return &Handler{store: store, sessions: sessions, log: log, issuer: Issuer(cfg)}
}

// Issuer is the authorization server's identifier: the public URL without a trailing slash.
func Issuer(cfg config.Config) string { return strings.TrimRight(cfg.PublicURL.String(), "/") }

func (h *Handler) resource() string { return h.issuer + MCPPath }

func (h *Handler) resourceMetadataURL() string {
	return h.issuer + "/.well-known/oauth-protected-resource" + MCPPath
}

// Register mounts the discovery documents and the /oauth routes.
func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /.well-known/oauth-protected-resource", h.protectedResource)
	mux.HandleFunc("GET /.well-known/oauth-protected-resource"+MCPPath, h.protectedResource)
	mux.HandleFunc("GET /.well-known/oauth-authorization-server", h.authorizationServer)
	mux.HandleFunc("POST /oauth/register", h.register)
	mux.HandleFunc("GET /oauth/authorize", h.authorize)
	mux.HandleFunc("POST /oauth/token", h.token)
	mux.HandleFunc("POST /oauth/revoke", h.revoke)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// oauthError writes an RFC 6749 error response.
func oauthError(w http.ResponseWriter, status int, code, desc string) {
	writeJSON(w, status, map[string]string{"error": code, "error_description": desc})
}

func (h *Handler) protectedResource(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"resource":                 h.resource(),
		"authorization_servers":    []string{h.issuer},
		"scopes_supported":         []string{ScopeRead},
		"bearer_methods_supported": []string{"header"},
		"resource_name":            "Ledger/01",
	})
}

func (h *Handler) authorizationServer(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"issuer":                                         h.issuer,
		"authorization_endpoint":                         h.issuer + "/oauth/authorize",
		"token_endpoint":                                 h.issuer + "/oauth/token",
		"registration_endpoint":                          h.issuer + "/oauth/register",
		"revocation_endpoint":                            h.issuer + "/oauth/revoke",
		"response_types_supported":                       []string{"code"},
		"grant_types_supported":                          []string{"authorization_code", "refresh_token"},
		"code_challenge_methods_supported":               []string{"S256"},
		"token_endpoint_auth_methods_supported":          []string{"none"},
		"revocation_endpoint_auth_methods_supported":     []string{"none"},
		"scopes_supported":                               []string{ScopeRead},
		"authorization_response_iss_parameter_supported": true,
	})
}

// validRedirectURI allows https, and plain http only on loopback hosts (native apps, RFC 8252).
func validRedirectURI(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil || u.Fragment != "" || u.Host == "" || len(raw) > 2000 {
		return false
	}
	switch u.Scheme {
	case "https":
		return true
	case "http":
		return isLoopback(u.Hostname())
	}
	return false
}

func isLoopback(host string) bool {
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// redirectMatches compares a requested redirect URI with a registered one. Loopback redirects may
// use any port, since native apps pick a free one at run time (RFC 8252 §7.3).
func redirectMatches(registered, requested string) bool {
	if registered == requested {
		return true
	}
	a, err1 := url.Parse(registered)
	b, err2 := url.Parse(requested)
	if err1 != nil || err2 != nil || a.Scheme != "http" || b.Scheme != "http" || !isLoopback(a.Hostname()) {
		return false
	}
	return a.Hostname() == b.Hostname() && a.Path == b.Path && a.RawQuery == b.RawQuery
}

// register implements RFC 7591 dynamic client registration for public clients.
func (h *Handler) register(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ClientName   string   `json:"client_name"`
		RedirectURIs []string `json:"redirect_uris"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBody)).Decode(&body); err != nil {
		oauthError(w, http.StatusBadRequest, "invalid_client_metadata", "expected a JSON body")
		return
	}
	if len(body.RedirectURIs) == 0 || len(body.RedirectURIs) > 10 {
		oauthError(w, http.StatusBadRequest, "invalid_redirect_uri", "give 1 to 10 redirect_uris")
		return
	}
	for _, u := range body.RedirectURIs {
		if !validRedirectURI(u) {
			oauthError(w, http.StatusBadRequest, "invalid_redirect_uri", "redirect URIs must be https, or http on a loopback host")
			return
		}
	}
	name := strings.TrimSpace(body.ClientName)
	if name == "" {
		name = "MCP client"
	}
	if utf8.RuneCountInString(name) > 100 {
		name = string([]rune(name)[:100])
	}
	c, err := h.store.RegisterClient(r.Context(), name, body.RedirectURIs)
	if err != nil {
		h.log.ErrorContext(r.Context(), "register oauth client", "err", err)
		oauthError(w, http.StatusInternalServerError, "server_error", "could not register the client")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"client_id":                  c.ID.String(),
		"client_id_issued_at":        c.CreatedAt.Unix(),
		"client_name":                c.Name,
		"redirect_uris":              c.RedirectURIs,
		"grant_types":                []string{"authorization_code", "refresh_token"},
		"response_types":             []string{"code"},
		"token_endpoint_auth_method": "none",
		"scope":                      ScopeRead,
	})
}

// authorize validates the request, sends signed-out users through Google sign-in first, then
// stores the request and hands over to the consent page.
func (h *Handler) authorize(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	q := r.URL.Query()
	clientID, err := uuid.Parse(q.Get("client_id"))
	if err != nil {
		http.Error(w, "unknown client_id", http.StatusBadRequest)
		return
	}
	client, err := h.store.Client(ctx, clientID)
	if errors.Is(err, ledger.ErrNotFound) {
		http.Error(w, "unknown client_id", http.StatusBadRequest)
		return
	}
	if err != nil {
		h.log.ErrorContext(ctx, "load oauth client", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	redirectURI := q.Get("redirect_uri")
	registered := false
	for _, u := range client.RedirectURIs {
		if redirectMatches(u, redirectURI) {
			registered = true
			break
		}
	}
	if !registered {
		// Never redirect to an unregistered URI.
		http.Error(w, "redirect_uri is not registered for this client", http.StatusBadRequest)
		return
	}

	// From here on, errors go back to the client.
	state := q.Get("state")
	fail := func(code, desc string) {
		p := url.Values{"error": {code}, "error_description": {desc}, "iss": {h.issuer}}
		if state != "" {
			p.Set("state", state)
		}
		http.Redirect(w, r, redirectWith(redirectURI, p), http.StatusFound)
	}
	if q.Get("response_type") != "code" {
		fail("unsupported_response_type", "only response_type=code is supported")
		return
	}
	challenge := q.Get("code_challenge")
	if q.Get("code_challenge_method") != "S256" || len(challenge) != 43 {
		fail("invalid_request", "PKCE with code_challenge_method=S256 is required")
		return
	}
	if res := q.Get("resource"); res != "" && strings.TrimRight(res, "/") != h.resource() {
		fail("invalid_target", "the only resource is "+h.resource())
		return
	}
	if len(state) > 500 {
		fail("invalid_request", "state is too long")
		return
	}

	if _, err := h.sessions.Resolve(ctx, auth.TokenFromHeader(r.Header)); err != nil {
		if !errors.Is(err, auth.ErrNoSession) {
			h.log.ErrorContext(ctx, "resolve session", "err", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		// Sign in, then come back here with the same query.
		http.Redirect(w, r, loginPath+"?next="+url.QueryEscape("/oauth/authorize?"+r.URL.RawQuery), http.StatusFound)
		return
	}

	id, err := h.store.CreateRequest(ctx, Request{
		ClientID: clientID, RedirectURI: redirectURI, CodeChallenge: challenge,
		// Whatever was asked for, read-only access is all there is to grant.
		Scope: ScopeRead, Resource: h.resource(), State: state,
	})
	if err != nil {
		h.log.ErrorContext(ctx, "create oauth request", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, consentPath+"?request="+id.String(), http.StatusFound)
}

func (h *Handler) token(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxBody)
	if err := r.ParseForm(); err != nil {
		oauthError(w, http.StatusBadRequest, "invalid_request", "expected a form body")
		return
	}
	clientID, err := uuid.Parse(r.PostForm.Get("client_id"))
	if err != nil {
		oauthError(w, http.StatusUnauthorized, "invalid_client", "unknown client_id")
		return
	}
	ctx := r.Context()
	var t Tokens
	switch r.PostForm.Get("grant_type") {
	case "authorization_code":
		t, err = h.store.ExchangeCode(ctx, clientID, r.PostForm.Get("code"), r.PostForm.Get("redirect_uri"), r.PostForm.Get("code_verifier"))
	case "refresh_token":
		t, err = h.store.Refresh(ctx, clientID, r.PostForm.Get("refresh_token"))
	default:
		oauthError(w, http.StatusBadRequest, "unsupported_grant_type", "use authorization_code or refresh_token")
		return
	}
	if errors.Is(err, ErrInvalidGrant) {
		oauthError(w, http.StatusBadRequest, "invalid_grant", "the code or refresh token is invalid, expired or already used")
		return
	}
	if err != nil {
		h.log.ErrorContext(ctx, "oauth token", "err", err)
		oauthError(w, http.StatusInternalServerError, "server_error", "could not issue tokens")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"access_token":  t.Access,
		"token_type":    "Bearer",
		"expires_in":    int(t.ExpiresIn / time.Second),
		"refresh_token": t.Refresh,
		"scope":         t.Scope,
	})
}

func (h *Handler) revoke(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxBody)
	if err := r.ParseForm(); err != nil {
		oauthError(w, http.StatusBadRequest, "invalid_request", "expected a form body")
		return
	}
	clientID, err := uuid.Parse(r.PostForm.Get("client_id"))
	if err != nil {
		oauthError(w, http.StatusUnauthorized, "invalid_client", "unknown client_id")
		return
	}
	if err := h.store.RevokeToken(r.Context(), clientID, r.PostForm.Get("token")); err != nil {
		h.log.ErrorContext(r.Context(), "oauth revoke", "err", err)
		oauthError(w, http.StatusInternalServerError, "server_error", "could not revoke")
		return
	}
	w.WriteHeader(http.StatusOK)
}

// Protect requires a valid bearer token and puts its actor in the request context. Failures get
// a 401 pointing at the protected-resource metadata, which is how MCP clients discover sign-in.
func (h *Handler) Protect(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, hasToken := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
		a, err := h.store.Resolve(r.Context(), strings.TrimSpace(token))
		if errors.Is(err, ErrNoToken) {
			challenge := `Bearer resource_metadata="` + h.resourceMetadataURL() + `", scope="` + ScopeRead + `"`
			if hasToken {
				challenge += `, error="invalid_token"`
			}
			w.Header().Set("WWW-Authenticate", challenge)
			http.Error(w, "sign in required", http.StatusUnauthorized)
			return
		}
		if err != nil {
			h.log.ErrorContext(r.Context(), "resolve access token", "err", err)
			http.Error(w, "internal error", http.StatusServiceUnavailable)
			return
		}
		next.ServeHTTP(w, r.WithContext(auth.WithActor(r.Context(), a.Actor)))
	})
}
