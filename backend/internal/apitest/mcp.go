package apitest

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

// RedirectURI is where the test MCP client registers to receive codes.
const RedirectURI = "https://client.example.com/callback"

// Verifier is a fixed PKCE code verifier; Challenge is its S256 challenge.
const Verifier = "test-verifier-0123456789-abcdefghijklmnopqrstuvwxyz"

func Challenge(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func rpcPath(service, method string) string { return "/ledger.v1." + service + "/" + method }

// Form is a form-encoded POST body.
func Form(v url.Values) Raw {
	return Raw{Body: v.Encode(), Header: http.Header{"Content-Type": {"application/x-www-form-urlencoded"}}}
}

// RegisterClient registers an MCP client and returns its id.
func RegisterClient(t testing.TB, do Driver, name string) string {
	t.Helper()
	r := do(t, http.MethodPost, "/oauth/register", map[string]any{"client_name": name, "redirect_uris": []string{RedirectURI}}, "")
	if r.Status != http.StatusCreated {
		t.Fatalf("register = %d %s", r.Status, r.Body)
	}
	return r.JSON(t)["client_id"].(string)
}

func authorizeQuery(clientID, state string) string {
	return "/oauth/authorize?" + url.Values{
		"response_type": {"code"}, "client_id": {clientID}, "redirect_uri": {RedirectURI}, "state": {state},
		"code_challenge": {Challenge(Verifier)}, "code_challenge_method": {"S256"},
		"resource": {"http://localhost:3000/mcp"}, "scope": {"ledger:read"},
	}.Encode()
}

// Approve runs /oauth/authorize as the signed-in user, approves on the consent RPC and returns the
// code from the redirect.
func Approve(t testing.TB, do Driver, session, clientID string) string {
	t.Helper()
	r := do(t, http.MethodGet, authorizeQuery(clientID, "st-1"), nil, session)
	loc := r.Header.Get("Location")
	if r.Status != http.StatusFound || !strings.HasPrefix(loc, "/connect?request=") {
		t.Fatalf("authorize = %d %q %s", r.Status, loc, r.Body)
	}
	id := strings.TrimPrefix(loc, "/connect?request=")
	r = do(t, http.MethodPost, rpcPath("OAuthService", "ApproveAuthorization"), map[string]string{"id": id}, session)
	if r.Status != http.StatusOK {
		t.Fatalf("approve = %d %s", r.Status, r.Body)
	}
	to, err := url.Parse(r.JSON(t)["redirectUrl"].(string))
	if err != nil || !strings.HasPrefix(to.String(), RedirectURI+"?") || to.Query().Get("state") != "st-1" ||
		to.Query().Get("iss") != "http://localhost:3000" {
		t.Fatalf("approve redirect = %v", to)
	}
	return to.Query().Get("code")
}

// Exchange trades a code for tokens.
func Exchange(t testing.TB, do Driver, clientID, code, verifier string) Response {
	t.Helper()
	return do(t, http.MethodPost, "/oauth/token", Form(url.Values{
		"grant_type": {"authorization_code"}, "client_id": {clientID}, "code": {code},
		"redirect_uri": {RedirectURI}, "code_verifier": {verifier},
	}), "")
}

// MCP sends one JSON-RPC call to /mcp with a bearer token and returns the response.
func MCP(t testing.TB, do Driver, token, method string, params any) Response {
	t.Helper()
	body, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 1, "method": method, "params": params})
	h := http.Header{"Content-Type": {"application/json"}, "Accept": {"application/json, text/event-stream"}}
	if token != "" {
		h.Set("Authorization", "Bearer "+token)
	}
	return do(t, http.MethodPost, "/mcp", Raw{Body: string(body), Header: h}, "")
}

// MCPResult returns a successful JSON-RPC result.
func MCPResult(t testing.TB, r Response) map[string]any {
	t.Helper()
	if r.Status != http.StatusOK {
		t.Fatalf("mcp = %d %s", r.Status, r.Body)
	}
	m := r.JSON(t)
	res, ok := m["result"].(map[string]any)
	if !ok {
		t.Fatalf("mcp error: %s", r.Body)
	}
	return res
}

// CallTool calls a tool and returns its structured content, failing on tool errors.
func CallTool(t testing.TB, do Driver, token, name string, args any) map[string]any {
	t.Helper()
	res := MCPResult(t, MCP(t, do, token, "tools/call", map[string]any{"name": name, "arguments": args}))
	if res["isError"] == true {
		t.Fatalf("tool %s failed: %v", name, res["content"])
	}
	return res["structuredContent"].(map[string]any)
}

// TokenFor signs in through the whole OAuth flow and returns an access and refresh token.
func TokenFor(t testing.TB, do Driver, session, clientID string) (access, refresh string) {
	t.Helper()
	r := Exchange(t, do, clientID, Approve(t, do, session, clientID), Verifier)
	if r.Status != http.StatusOK {
		t.Fatalf("token = %d %s", r.Status, r.Body)
	}
	m := r.JSON(t)
	return m["access_token"].(string), m["refresh_token"].(string)
}

func randomHex() string {
	b := make([]byte, 6)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// MCPScenario connects an MCP client the way Claude does (discovery, registration, sign-in,
// consent, code exchange) and asks the ledger a question, recording stable observations.
func MCPScenario(t *testing.T, do Driver) Result {
	t.Helper()
	var res Result

	// Without a token, /mcp points at the protected-resource metadata.
	r := MCP(t, do, "", "tools/list", map[string]any{})
	www := r.Header.Get("WWW-Authenticate")
	if r.Status != http.StatusUnauthorized ||
		!strings.Contains(www, `resource_metadata="http://localhost:3000/.well-known/oauth-protected-resource/mcp"`) {
		t.Fatalf("anonymous mcp = %d %q", r.Status, www)
	}
	res.add("mcp-anon %d", r.Status)

	r = do(t, http.MethodGet, "/.well-known/oauth-protected-resource/mcp", nil, "")
	if r.Status != http.StatusOK || r.JSON(t)["resource"] != "http://localhost:3000/mcp" {
		t.Fatalf("resource metadata = %d %s", r.Status, r.Body)
	}
	r = do(t, http.MethodGet, "/.well-known/oauth-authorization-server", nil, "")
	if m := r.JSON(t); r.Status != http.StatusOK || m["issuer"] != "http://localhost:3000" ||
		m["registration_endpoint"] != "http://localhost:3000/oauth/register" {
		t.Fatalf("server metadata = %d %s", r.Status, r.Body)
	}
	res.add("metadata ok")

	// Registration refuses plain-http redirects off loopback.
	r = do(t, http.MethodPost, "/oauth/register", map[string]any{"client_name": "Bad", "redirect_uris": []string{"http://client.example.com/cb"}}, "")
	if r.Status != http.StatusBadRequest {
		t.Fatalf("register http redirect = %d", r.Status)
	}
	res.add("register-http-redirect %d", r.Status)
	clientName := "Test client " + randomHex()
	clientID := RegisterClient(t, do, clientName)

	// Signed-out users go through the login page and come back.
	r = do(t, http.MethodGet, authorizeQuery(clientID, "st-1"), nil, "")
	if loc := r.Header.Get("Location"); r.Status != http.StatusFound || !strings.HasPrefix(loc, "/login?next=%2Foauth%2Fauthorize%3F") {
		t.Fatalf("authorize signed out = %d %q", r.Status, loc)
	}
	// An unregistered redirect URI is an error page, never a redirect.
	bad := strings.Replace(authorizeQuery(clientID, "st-1"), url.QueryEscape(RedirectURI), url.QueryEscape("https://evil.example.com/cb"), 1)
	if r = do(t, http.MethodGet, bad, nil, ""); r.Status != http.StatusBadRequest {
		t.Fatalf("authorize with unregistered redirect = %d", r.Status)
	}
	res.add("authorize-anon redirects to login")

	r = do(t, http.MethodPost, "/auth/dev/login", map[string]string{"email": "owner@example.com"}, "")
	session := SessionCookie(r)

	// Something to ask about: a pizza in a fresh wallet.
	r = do(t, http.MethodPost, rpcPath("WalletService", "CreateWallet"), map[string]any{"name": "MCP wallet", "currency": "USD"}, session)
	wallet := r.JSON(t)["wallet"].(map[string]any)["id"].(string)
	r = do(t, http.MethodPost, rpcPath("CategoryService", "ListCategories"), "{}", session)
	var category string
	for _, c := range r.JSON(t)["categories"].([]any) {
		if c := c.(map[string]any); c["kind"] == "CATEGORY_KIND_EXPENSE" {
			category = c["id"].(string)
			break
		}
	}
	marker := "pizza-" + randomHex()
	for _, on := range []string{"2026-09-12", "2026-10-03", "2026-12-01"} {
		r = do(t, http.MethodPost, rpcPath("TransactionService", "CreateTransaction"), map[string]any{
			"walletId": wallet, "kind": "TRANSACTION_KIND_EXPENSE", "categoryId": category, "amountMinor": "1850",
			"occurredOn": on, "note": "Corner " + marker, "today": "2026-10-30",
		}, session)
		if r.Status != http.StatusOK {
			t.Fatalf("CreateTransaction = %d %s", r.Status, r.Body)
		}
	}

	// A wrong verifier spends the code.
	code := Approve(t, do, session, clientID)
	if r = Exchange(t, do, clientID, code, Verifier+"x"); r.Status != http.StatusBadRequest || r.JSON(t)["error"] != "invalid_grant" {
		t.Fatalf("exchange with wrong verifier = %d %s", r.Status, r.Body)
	}
	if r = Exchange(t, do, clientID, code, Verifier); r.Status != http.StatusBadRequest {
		t.Fatalf("exchange of a spent code = %d %s", r.Status, r.Body)
	}
	res.add("bad-verifier %d", r.Status)

	code = Approve(t, do, session, clientID)
	r = Exchange(t, do, clientID, code, Verifier)
	if r.Status != http.StatusOK || r.Header.Get("Cache-Control") != "no-store" {
		t.Fatalf("exchange = %d %s", r.Status, r.Body)
	}
	tok := r.JSON(t)
	access, refresh := tok["access_token"].(string), tok["refresh_token"].(string)
	if r = Exchange(t, do, clientID, code, Verifier); r.Status != http.StatusBadRequest {
		t.Fatalf("code reuse = %d", r.Status)
	}
	res.add("token %v %v", tok["token_type"], tok["scope"])

	init := MCPResult(t, MCP(t, do, access, "initialize", map[string]any{
		"protocolVersion": "2025-06-18", "capabilities": map[string]any{},
		"clientInfo": map[string]any{"name": "test", "version": "1"},
	}))
	res.add("initialize %v", init["serverInfo"].(map[string]any)["name"])

	tools := MCPResult(t, MCP(t, do, access, "tools/list", map[string]any{}))["tools"].([]any)
	var names []string
	for _, tl := range tools {
		tl := tl.(map[string]any)
		if tl["annotations"].(map[string]any)["readOnlyHint"] != true {
			t.Fatalf("tool %v is not read-only", tl["name"])
		}
		names = append(names, tl["name"].(string))
	}
	res.add("tools %s", strings.Join(names, ","))

	// Upcoming rows (after today) never show.
	found := CallTool(t, do, access, "search_transactions", map[string]any{
		"from": "2026-01-01", "to": "2026-12-31", "today": "2026-10-30", "note_query": strings.ToUpper(marker),
	})["transactions"].([]any)
	if len(found) != 2 {
		t.Fatalf("search found %d rows, want 2: %v", len(found), found)
	}
	first := found[0].(map[string]any)
	amount := first["amount"].(map[string]any)
	if first["date"] != "2026-10-03" || first["wallet"] != "MCP wallet" || amount["display"] != "18.50" || amount["currency"] != "USD" {
		t.Fatalf("search row = %v", first)
	}
	res.add("search %d rows, newest %v %v", len(found), first["date"], amount["display"])

	sum := CallTool(t, do, access, "spending_summary", map[string]any{
		"from": "2026-09-01", "to": "2026-10-31", "today": "2026-10-30", "note_query": marker, "bucket": "month",
		"wallet_ids": []string{wallet},
	})
	if len(sum["series"].([]any)) != 2 || sum["expenses"].(map[string]any)["minor"] != float64(3700) {
		t.Fatalf("spending summary = %v", sum)
	}
	res.add("summary expenses %v", sum["expenses"].(map[string]any)["display"])

	// A tool error is a result the model can read, not a protocol failure.
	bad2 := MCPResult(t, MCP(t, do, access, "tools/call", map[string]any{
		"name": "search_transactions", "arguments": map[string]any{"from": "yesterday", "to": "2026-10-30"},
	}))
	if bad2["isError"] != true {
		t.Fatalf("bad date = %v", bad2)
	}
	res.add("tool-error ok")

	// Refresh tokens rotate; replaying a spent one revokes the grant.
	rr := do(t, http.MethodPost, "/oauth/token", Form(url.Values{"grant_type": {"refresh_token"}, "client_id": {clientID}, "refresh_token": {refresh}}), "")
	if rr.Status != http.StatusOK {
		t.Fatalf("refresh = %d %s", rr.Status, rr.Body)
	}
	access2 := rr.JSON(t)["access_token"].(string)
	MCPResult(t, MCP(t, do, access2, "tools/list", map[string]any{}))
	rr = do(t, http.MethodPost, "/oauth/token", Form(url.Values{"grant_type": {"refresh_token"}, "client_id": {clientID}, "refresh_token": {refresh}}), "")
	if rr.Status != http.StatusBadRequest {
		t.Fatalf("refresh replay = %d", rr.Status)
	}
	if r = MCP(t, do, access2, "tools/list", map[string]any{}); r.Status != http.StatusUnauthorized ||
		!strings.Contains(r.Header.Get("WWW-Authenticate"), `error="invalid_token"`) {
		t.Fatalf("after refresh replay = %d %q", r.Status, r.Header.Get("WWW-Authenticate"))
	}
	res.add("refresh-replay revokes %d", r.Status)

	// Connected apps: a new grant shows up and revoking it cuts the client off.
	access3, _ := TokenFor(t, do, session, clientID)
	r = do(t, http.MethodPost, rpcPath("OAuthService", "ListConnections"), "{}", session)
	var grant string
	for _, c := range r.JSON(t)["connections"].([]any) {
		if c := c.(map[string]any); c["clientName"] == clientName {
			grant = c["id"].(string)
			if c["redirectHost"] != "client.example.com" {
				t.Fatalf("connection = %v", c)
			}
		}
	}
	if grant == "" {
		t.Fatalf("connection not listed: %s", r.Body)
	}
	if r = do(t, http.MethodPost, rpcPath("OAuthService", "RevokeConnection"), map[string]string{"id": grant}, session); r.Status != http.StatusOK {
		t.Fatalf("revoke = %d %s", r.Status, r.Body)
	}
	if r = MCP(t, do, access3, "tools/list", map[string]any{}); r.Status != http.StatusUnauthorized {
		t.Fatalf("after revoke = %d", r.Status)
	}
	res.add("revoked %d", r.Status)
	return res
}
