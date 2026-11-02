package apitest

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
)

// Result captures what a scenario observed, so two drivers can be compared.
type Result struct {
	Steps []string
}

func (r *Result) add(format string, args ...any) {
	r.Steps = append(r.Steps, fmt.Sprintf(format, args...))
}

// Scenario exercises auth, the allowlist and a cross-currency transfer through a driver, failing
// on unexpected responses and recording a transcript of stable (id-free) observations.
func Scenario(t *testing.T, do Driver) Result {
	t.Helper()
	var res Result
	rpc := func(service, method string) string { return "/ledger.v1." + service + "/" + method }

	// Unauthenticated RPCs are rejected.
	r := do(t, http.MethodPost, rpc("UserService", "GetMe"), "{}", "")
	if r.Status != http.StatusUnauthorized {
		t.Fatalf("GetMe without session = %d %s", r.Status, r.Body)
	}
	res.add("getme-anon %d", r.Status)

	// Emails outside the allowlist can't sign in.
	r = do(t, http.MethodPost, "/auth/dev/login", map[string]string{"email": "stranger@example.com"}, "")
	if r.Status != http.StatusForbidden {
		t.Fatalf("dev login of stranger = %d", r.Status)
	}
	res.add("login-stranger %d", r.Status)

	// The dev login of an allowlisted email sets an httpOnly session cookie.
	r = do(t, http.MethodPost, "/auth/dev/login", map[string]string{"email": "owner@example.com"}, "")
	session := SessionCookie(r)
	if r.Status != http.StatusNoContent || session == "" {
		t.Fatalf("dev login = %d, cookies %v", r.Status, r.Cookies)
	}
	for _, c := range r.Cookies {
		if c.Name == "ledger_session" && (!c.HttpOnly || c.SameSite != http.SameSiteLaxMode || c.Path != "/") {
			t.Fatalf("session cookie attributes: %+v", c)
		}
	}
	res.add("login-owner %d cookie", r.Status)

	r = do(t, http.MethodPost, rpc("UserService", "GetMe"), "{}", session)
	if r.Status != http.StatusOK || r.JSON(t)["user"].(map[string]any)["email"] != "owner@example.com" {
		t.Fatalf("GetMe = %d %s", r.Status, r.Body)
	}
	res.add("getme %d", r.Status)

	// Two wallets in different currencies and a transfer between them.
	wallet := func(name, currency string) string {
		r := do(t, http.MethodPost, rpc("WalletService", "CreateWallet"),
			map[string]any{"name": name, "currency": currency, "initialBalanceMinor": "50000"}, session)
		if r.Status != http.StatusOK {
			t.Fatalf("CreateWallet = %d %s", r.Status, r.Body)
		}
		return r.JSON(t)["wallet"].(map[string]any)["id"].(string)
	}
	usd := wallet("Parity USD", "USD")
	uyu := wallet("Parity UYU", "UYU")
	res.add("wallets created")

	r = do(t, http.MethodPost, rpc("TransactionService", "CreateTransfer"), map[string]any{
		"fromWalletId": usd, "toWalletId": uyu, "fromAmountMinor": "12000", "toAmountMinor": "470000",
		"rate": "39.1666666667", "occurredOn": "2026-10-27",
	}, session)
	if r.Status != http.StatusOK {
		t.Fatalf("CreateTransfer = %d %s", r.Status, r.Body)
	}
	tr := r.JSON(t)["transfer"].(map[string]any)
	res.add("transfer %d rate=%s", r.Status, tr["rate"])

	r = do(t, http.MethodPost, rpc("WalletService", "GetWallet"), map[string]any{"id": uyu}, session)
	bal := r.JSON(t)["wallet"].(map[string]any)["balanceMinor"]
	if bal != "520000" {
		t.Fatalf("UYU balance = %v", bal)
	}
	res.add("uyu balance %v", bal)

	// A monthly expense, three times from Oct 27, seen on Oct 28: the first one counts, the other
	// two are upcoming.
	r = do(t, http.MethodPost, rpc("CategoryService", "ListCategories"), "{}", session)
	var category string
	for _, c := range r.JSON(t)["categories"].([]any) {
		if m := c.(map[string]any); m["kind"] == "CATEGORY_KIND_EXPENSE" {
			category = m["id"].(string)
			break
		}
	}
	r = do(t, http.MethodPost, rpc("TransactionService", "CreateTransaction"), map[string]any{
		"walletId": usd, "categoryId": category, "amountMinor": "1000", "occurredOn": "2026-10-27", "today": "2026-10-28",
		"repeat": map[string]any{"frequency": "RECURRING_FREQUENCY_MONTHLY", "endCount": 3},
	}, session)
	if r.Status != http.StatusOK {
		t.Fatalf("CreateTransaction with repeat = %d %s", r.Status, r.Body)
	}
	tx := r.JSON(t)["transaction"].(map[string]any)
	rec, _ := tx["recurrence"].(map[string]any)
	if tx["recurringRuleId"] == nil || rec["endDate"] != "2026-12-27" {
		t.Fatalf("recurring transaction = %v", tx)
	}
	res.add("recurring %d end=%s", r.Status, rec["endDate"])

	r = do(t, http.MethodPost, rpc("TransactionService", "ListUpcomingTransactions"),
		map[string]any{"walletId": usd, "today": "2026-10-28"}, session)
	upcoming, _ := r.JSON(t)["transactions"].([]any)
	if r.Status != http.StatusOK || len(upcoming) != 2 {
		t.Fatalf("ListUpcomingTransactions = %d %s", r.Status, r.Body)
	}
	res.add("upcoming %d", len(upcoming))

	r = do(t, http.MethodPost, rpc("WalletService", "GetWallet"), map[string]any{"id": usd, "today": "2026-10-28"}, session)
	bal = r.JSON(t)["wallet"].(map[string]any)["balanceMinor"]
	if bal != "37000" {
		t.Fatalf("USD balance on Oct 28 = %v", bal)
	}
	res.add("usd balance %v", bal)

	// Invalid input maps to invalid_argument (HTTP 400).
	r = do(t, http.MethodPost, rpc("TransactionService", "CreateTransfer"), map[string]any{
		"fromWalletId": usd, "toWalletId": uyu, "fromAmountMinor": "100", "toAmountMinor": "999999",
		"rate": "39.05", "occurredOn": "2026-10-27",
	}, session)
	if r.Status != http.StatusBadRequest || !strings.Contains(string(r.Body), "invalid_argument") {
		t.Fatalf("bad transfer = %d %s", r.Status, r.Body)
	}
	res.add("bad-transfer %d", r.Status)

	// Google sign-in starts with a redirect carrying PKCE and the flow cookie.
	r = do(t, http.MethodGet, "/auth/google/start?next=/wallets", nil, "")
	loc := r.Header.Get("Location")
	if r.Status != http.StatusFound || !strings.HasPrefix(loc, "https://accounts.google.com/") ||
		!strings.Contains(loc, "code_challenge_method=S256") || !strings.Contains(loc, "redirect_uri=http%3A%2F%2Flocalhost%3A3000%2Fauth%2Fgoogle%2Fcallback") {
		t.Fatalf("google start = %d %s", r.Status, loc)
	}
	hasFlow := false
	for _, c := range r.Cookies {
		hasFlow = hasFlow || c.Name == "ledger_oauth"
	}
	if !hasFlow {
		t.Fatal("google start didn't set the flow cookie")
	}
	res.add("google-start %d", r.Status)

	// Logout revokes the session.
	r = do(t, http.MethodPost, "/auth/logout", nil, session)
	if r.Status != http.StatusNoContent {
		t.Fatalf("logout = %d", r.Status)
	}
	r = do(t, http.MethodPost, rpc("UserService", "GetMe"), "{}", session)
	if r.Status != http.StatusUnauthorized {
		t.Fatalf("GetMe after logout = %d", r.Status)
	}
	res.add("after-logout %d", r.Status)
	return res
}
