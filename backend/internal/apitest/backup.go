package apitest

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"testing"
)

// backupFixture is a minimal backup: one wallet and nothing else.
const backupFixture = `{"format":"ledger-01-backup","version":1,"household":{"name":"Parity household","base_currency":"USD"},
"wallets":[{"id":"0192f5a0-0000-7000-8000-000000000001","name":"Parity backup","currency":"USD","initial_balance_minor":1234,"hidden":false,"position":0}]}`

// BackupScenario restores a fixed backup, then exports it again and as CSV. It starts from
// a restore, so the household's state from an earlier run doesn't change what it observes.
func BackupScenario(t *testing.T, do Driver) Result {
	t.Helper()
	var res Result

	r := do(t, http.MethodPost, "/auth/dev/login", map[string]string{"email": "partner@example.com"}, "")
	session := SessionCookie(r)
	if session == "" {
		t.Fatalf("dev login = %d", r.Status)
	}

	var gz bytes.Buffer
	zw := gzip.NewWriter(&gz)
	_, _ = zw.Write([]byte(backupFixture))
	_ = zw.Close()
	content := base64.StdEncoding.EncodeToString(gz.Bytes())

	// Restoring needs the explicit replace flag.
	r = do(t, http.MethodPost, rpcPath("BackupService", "RestoreBackup"), map[string]any{"content": content}, session)
	if r.Status != http.StatusBadRequest {
		t.Fatalf("restore without replace = %d %s", r.Status, r.Body)
	}
	res.add("restore-unconfirmed %d", r.Status)

	r = do(t, http.MethodPost, rpcPath("BackupService", "RestoreBackup"), map[string]any{"content": content, "replace": true}, session)
	if r.Status != http.StatusOK {
		t.Fatalf("restore = %d %s", r.Status, r.Body)
	}
	res.add("restore %d counts=%v", r.Status, r.JSON(t)["counts"])

	r = do(t, http.MethodPost, rpcPath("WalletService", "ListWallets"), "{}", session)
	wallets := r.JSON(t)["wallets"].([]any)
	if len(wallets) != 1 || wallets[0].(map[string]any)["name"] != "Parity backup" {
		t.Fatalf("wallets after restore = %s", r.Body)
	}
	res.add("wallets %d balance=%v", len(wallets), wallets[0].(map[string]any)["balanceMinor"])

	r = do(t, http.MethodPost, rpcPath("BackupService", "ExportBackup"), "{}", session)
	if r.Status != http.StatusOK {
		t.Fatalf("export = %d %s", r.Status, r.Body)
	}
	raw, err := base64.StdEncoding.DecodeString(r.JSON(t)["content"].(string))
	if err != nil {
		t.Fatal(err)
	}
	zr, err := gzip.NewReader(bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	plain, _ := io.ReadAll(zr)
	var b struct {
		Format    string `json:"format"`
		Household struct {
			Name string `json:"name"`
		} `json:"household"`
		Wallets []struct {
			Name string `json:"name"`
		} `json:"wallets"`
	}
	if err := json.Unmarshal(plain, &b); err != nil {
		t.Fatal(err)
	}
	res.add("export %s household=%q wallets=%d", b.Format, b.Household.Name, len(b.Wallets))

	r = do(t, http.MethodPost, rpcPath("BackupService", "ExportTransactionsCsv"), map[string]any{"today": "2026-10-31"}, session)
	if r.Status != http.StatusOK {
		t.Fatalf("csv = %d %s", r.Status, r.Body)
	}
	m := r.JSON(t)
	res.add("csv %d rows=%v name=%v", r.Status, m["rowCount"], m["fileName"])
	return res
}
