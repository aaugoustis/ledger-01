package server_test

import (
	"bytes"
	"encoding/base64"
	"net/http"
	"strings"
	"testing"

	"github.com/franco-roura/ledger-01/backend/internal/apitest"
	"github.com/franco-roura/ledger-01/backend/internal/importer"
	"github.com/franco-roura/ledger-01/backend/internal/rpc"
)

func TestScenarioOverHTTP(t *testing.T) {
	cfg := apitest.Config(t, nil)
	apitest.Scenario(t, apitest.HTTPDriver(t, apitest.Handler(t, cfg)))
}

func TestDevLoginDisabledInProduction(t *testing.T) {
	cfg := apitest.Config(t, nil)
	cfg.Env = "production"
	do := apitest.HTTPDriver(t, apitest.Handler(t, cfg))
	r := do(t, http.MethodPost, "/auth/dev/login", map[string]string{"email": "owner@example.com"}, "")
	if r.Status != http.StatusNotFound && r.Status != http.StatusMethodNotAllowed {
		t.Fatalf("dev login in production = %d, want 404/405", r.Status)
	}
}

func TestBasePathIsStripped(t *testing.T) {
	cfg := apitest.Config(t, map[string]string{"API_BASE_PATH": "/api"})
	do := apitest.HTTPDriver(t, apitest.Handler(t, cfg))
	// Prefixed RPC paths reach the Connect handlers...
	if r := do(t, http.MethodPost, "/api/ledger.v1.UserService/GetMe", "{}", ""); r.Status != http.StatusUnauthorized {
		t.Fatalf("prefixed RPC = %d %s", r.Status, r.Body)
	}
	// ...and unprefixed /auth routes still work.
	if r := do(t, http.MethodGet, "/auth/google/start", nil, ""); r.Status != http.StatusFound {
		t.Fatalf("/auth without prefix = %d", r.Status)
	}
	if r := do(t, http.MethodGet, "/api/healthz", nil, ""); r.Status != http.StatusOK {
		t.Fatalf("/api/healthz = %d", r.Status)
	}
}

// Every other RPC refuses bodies over 1 MB, even from callers without a session.
func TestRequestSizeLimit(t *testing.T) {
	do := apitest.HTTPDriver(t, apitest.Handler(t, apitest.Config(t, nil)))
	big := map[string]any{"name": strings.Repeat("x", rpc.ReadMaxBytes)}
	if r := do(t, http.MethodPost, "/ledger.v1.WalletService/CreateWallet", big, ""); r.Status != http.StatusTooManyRequests {
		t.Fatalf("anonymous 1 MB request = %d %.200s, want 429 (resource_exhausted)", r.Status, r.Body)
	}
	r := do(t, http.MethodPost, "/auth/dev/login", map[string]string{"email": "partner@example.com"}, "")
	if r = do(t, http.MethodPost, "/ledger.v1.WalletService/CreateWallet", big, apitest.SessionCookie(r)); r.Status != http.StatusTooManyRequests {
		t.Fatalf("signed-in 1 MB request = %d %.200s, want 429 (resource_exhausted)", r.Status, r.Body)
	}
}

// A 4 MB file fits through the JSON upload (base64 included); bigger bodies are refused before
// they are parsed.
func TestImportUploadLimits(t *testing.T) {
	do := apitest.HTTPDriver(t, apitest.Handler(t, apitest.Config(t, nil)))
	r := do(t, http.MethodPost, "/auth/dev/login", map[string]string{"email": "partner@example.com"}, "")
	session := apitest.SessionCookie(r)
	r = do(t, http.MethodPost, "/ledger.v1.WalletService/CreateWallet", map[string]any{"name": "Upload limits", "currency": "USD"}, session)
	wallet := r.JSON(t)["wallet"].(map[string]any)["id"].(string)

	upload := func(size int) apitest.Response {
		row := []byte("2026-10-01,Corner Grocer,-1.00\n")
		content := bytes.Repeat(row, size/len(row))
		return do(t, http.MethodPost, "/ledger.v1.ImportService/UploadImportFile", map[string]any{
			"walletId": wallet, "fileName": "big.csv", "content": base64.StdEncoding.EncodeToString(content),
		}, session)
	}
	// Just under the cap parses (and is refused only for its row count).
	if r := upload(importer.MaxFileBytes - 64); r.Status != http.StatusBadRequest || !strings.Contains(string(r.Body), "more than 10000 rows") {
		t.Fatalf("4 MB upload = %d %.200s", r.Status, r.Body)
	}
	// Over the cap: refused by the file check, or by the request size limit.
	if r := upload(importer.MaxFileBytes + 64); r.Status != http.StatusBadRequest || !strings.Contains(string(r.Body), "larger than 4 MB") {
		t.Fatalf("over-cap upload = %d %.200s", r.Status, r.Body)
	}
	if r := upload(5 << 20); r.Status == http.StatusOK || r.Status == http.StatusBadRequest {
		t.Fatalf("5 MB upload = %d %.200s, want the request refused for its size", r.Status, r.Body)
	}
}

func TestMCPOverHTTP(t *testing.T) {
	apitest.MCPScenario(t, apitest.HTTPDriver(t, apitest.Handler(t, apitest.Config(t, nil))))
}

// A token only ever sees its user's household.
func TestMCPHouseholdIsolation(t *testing.T) {
	do := apitest.HTTPDriver(t, apitest.Handler(t, apitest.Config(t, nil)))
	login := func(email string) string {
		return apitest.SessionCookie(do(t, http.MethodPost, "/auth/dev/login", map[string]string{"email": email}, ""))
	}
	owner, partner := login("owner@example.com"), login("partner@example.com")
	r := do(t, http.MethodPost, "/ledger.v1.WalletService/CreateWallet", map[string]any{"name": "Owner only", "currency": "USD"}, owner)
	if r.Status != http.StatusOK {
		t.Fatalf("CreateWallet = %d %s", r.Status, r.Body)
	}

	client := apitest.RegisterClient(t, do, "Isolation client")
	access, _ := apitest.TokenFor(t, do, partner, client)
	for _, w := range apitest.CallTool(t, do, access, "list_wallets", map[string]any{})["wallets"].([]any) {
		if w.(map[string]any)["name"] == "Owner only" {
			t.Fatalf("partner's token sees the owner's wallet")
		}
	}
	// The owner's wallet id is not found either.
	id := r.JSON(t)["wallet"].(map[string]any)["id"].(string)
	got := apitest.CallTool(t, do, access, "search_transactions", map[string]any{
		"from": "2020-01-01", "to": "2030-12-31", "wallet_ids": []string{id},
	})["transactions"].([]any)
	if len(got) != 0 {
		t.Fatalf("partner's search returned %d of the owner's rows", len(got))
	}
	// Another user can't approve or read a request with a guessed id... nor revoke others' grants.
	conns := do(t, http.MethodPost, "/ledger.v1.OAuthService/ListConnections", "{}", partner).JSON(t)["connections"].([]any)
	grant := conns[0].(map[string]any)["id"].(string)
	if r := do(t, http.MethodPost, "/ledger.v1.OAuthService/RevokeConnection", map[string]string{"id": grant}, owner); r.Status != http.StatusNotFound {
		t.Fatalf("owner revoking partner's grant = %d", r.Status)
	}
}

func TestMCPRejectsGet(t *testing.T) {
	do := apitest.HTTPDriver(t, apitest.Handler(t, apitest.Config(t, nil)))
	if r := do(t, http.MethodGet, "/mcp", nil, ""); r.Status != http.StatusMethodNotAllowed {
		t.Fatalf("GET /mcp = %d", r.Status)
	}
}
