package ledger_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/franco-roura/ledger-01/backend/internal/ledger"
	"github.com/franco-roura/ledger-01/backend/internal/testdb"
	"github.com/google/uuid"
)

var uuidRe = regexp.MustCompile(`[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`)

var emailRe = regexp.MustCompile(`[a-z0-9.-]+@example\.com`)

// canon renders a backup with ids replaced by their order of first appearance, so two exports
// of the same data compare equal even though a restore gave every row a new id. Members and
// creator emails differ between households; TestRestoreMatchesCreatorsByEmail covers them.
func canon(t *testing.T, b *ledger.Backup) string {
	t.Helper()
	c := *b
	c.ExportedAt = time.Time{}
	c.Members = nil
	raw, err := json.MarshalIndent(c, "", " ")
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]int{}
	out := emailRe.ReplaceAllString(string(raw), "someone@example.com")
	return uuidRe.ReplaceAllStringFunc(out, func(id string) string {
		if _, ok := seen[id]; !ok {
			seen[id] = len(seen)
		}
		return fmt.Sprintf("id-%d", seen[id])
	})
}

func export(t *testing.T, svc *ledger.Service, a ledger.Actor) *ledger.Backup {
	t.Helper()
	b, err := svc.ExportBackup(context.Background(), a)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func balances(t *testing.T, svc *ledger.Service, a ledger.Actor) map[string]int64 {
	t.Helper()
	ws, err := svc.ListWallets(context.Background(), a, true, true, day)
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]int64{}
	for _, w := range ws {
		out[w.Name] = w.Balance
	}
	return out
}

// seedEverything fills a household with one of each kind of row a backup carries.
func seedEverything(t *testing.T, svc *ledger.Service, a ledger.Actor) {
	t.Helper()
	ctx := context.Background()
	checking := mustWallet(t, svc, a, "Checking", "USD", 1000_00)
	pesos := mustWallet(t, svc, a, "Pesos", "UYU", 0)
	groceries := categoryNamed(t, svc, a, ledger.CategoryExpense, "Groceries")
	transport := categoryNamed(t, svc, a, ledger.CategoryExpense, "Transport")
	trip, err := svc.CreateLabel(ctx, a, "Trip", "cat-teal")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.CreateTransaction(ctx, a, ledger.TransactionInput{WalletID: checking.ID, CategoryID: groceries.ID,
		Amount: 42_50, OccurredOn: day, Note: "Corner market", LabelIDs: []uuid.UUID{trip.ID}}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.CreateTransfer(ctx, a, ledger.TransferInput{FromWalletID: checking.ID, ToWalletID: pesos.ID,
		FromAmount: 100_00, ToAmount: 4000_00, Rate: "40", OccurredOn: day, Note: "Exchange"}); err != nil {
		t.Fatal(err)
	}
	// Monthly, so the next occurrences are upcoming rows.
	if _, err := svc.CreateTransaction(ctx, a, ledger.TransactionInput{WalletID: pesos.ID, CategoryID: transport.ID,
		Amount: 1500_00, OccurredOn: day, Note: "Bus pass", Today: day,
		Repeat: ledger.Recurrence{Frequency: ledger.FreqMonthly}}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.CreateBudget(ctx, a, ledger.BudgetInput{Name: "Food", Amount: 300_00, Currency: "USD",
		Repeat: ledger.RepeatMonthly, Start: on("2026-10-01"), WalletIDs: []uuid.UUID{checking.ID},
		CategoryIDs: []uuid.UUID{groceries.ID}}); err != nil {
		t.Fatal(err)
	}
	if err := svc.SetCategoryHidden(ctx, a, pesos.ID, groceries.ID, true); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.CreateCategoryRule(ctx, a, "transit", transport.ID); err != nil {
		t.Fatal(err)
	}
	// A statement import leaves a saved mapping and import refs behind.
	up, err := svc.UploadImport(ctx, a, checking.ID, "northbank.csv", bankFile(t, "northbank.csv"))
	if err != nil {
		t.Fatal(err)
	}
	other := categoryNamed(t, svc, a, ledger.CategoryIncome, "Other")
	if _, err := svc.CommitImport(ctx, a, up.BatchID, up.Mapping, []ledger.ImportChoice{
		{Index: 0, CategoryID: other.ID}, {Index: 3, CategoryID: transport.ID},
	}, true); err != nil {
		t.Fatal(err)
	}
}

func TestBackupRoundTrip(t *testing.T) {
	svc := ledger.New(testdb.Pool(t))
	ctx := context.Background()
	a, b := newActor(t, svc), newActor(t, svc)
	seedEverything(t, svc, a)
	mustWallet(t, svc, b, "Replaced", "USD", 5_00)

	before := export(t, svc, a)
	if len(before.Wallets) != 2 || len(before.Transfers) != 1 || len(before.RecurringRules) != 1 ||
		len(before.Budgets) != 1 || len(before.HiddenCategories) != 1 || len(before.ImportRules) != 1 ||
		len(before.ImportProfiles) != 1 || len(before.ImportRefs) == 0 || len(before.Labels) != 1 {
		t.Fatalf("seeded backup is missing rows: %+v", before.Counts())
	}
	content, err := ledger.EncodeBackup(before)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := ledger.DecodeBackup(content)
	if err != nil {
		t.Fatal(err)
	}
	counts, err := svc.RestoreBackup(ctx, b, decoded)
	if err != nil {
		t.Fatal(err)
	}
	if counts != before.Counts() {
		t.Fatalf("counts = %+v, want %+v", counts, before.Counts())
	}

	after := export(t, svc, b)
	if got, want := canon(t, after), canon(t, before); got != want {
		t.Fatalf("restored household differs:\n got %s\nwant %s", got, want)
	}
	if got, want := balances(t, svc, b), balances(t, svc, a); fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("balances = %v, want %v", got, want)
	}
	// No row of b points at a's data, and a is untouched.
	for _, id := range uuidRe.FindAllString(canonRaw(t, after), -1) {
		if strings.Contains(canonRaw(t, before), id) {
			t.Fatalf("restored backup reuses id %s", id)
		}
	}
	if got, want := canon(t, export(t, svc, a)), canon(t, before); got != want {
		t.Fatal("restoring into b changed a")
	}

	// The restored import refs still flag the statement's rows as duplicates.
	var checking ledger.Wallet
	ws, _ := svc.ListWallets(ctx, b, true, true, day)
	for _, w := range ws {
		if w.Name == "Checking" {
			checking = w
		}
	}
	up, err := svc.UploadImport(ctx, b, checking.ID, "northbank.csv", bankFile(t, "northbank.csv"))
	if err != nil {
		t.Fatal(err)
	}
	if !up.SavedMapping {
		t.Fatal("restored wallet lost its saved mapping")
	}
	rows, _, err := svc.PreviewImport(ctx, b, up.BatchID, up.Mapping, 0)
	if err != nil {
		t.Fatal(err)
	}
	if !rows[0].Duplicate() || !rows[3].Duplicate() {
		t.Fatal("restored import refs don't mark imported rows as duplicates")
	}
}

func canonRaw(t *testing.T, b *ledger.Backup) string {
	t.Helper()
	raw, err := json.Marshal(b)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func TestRestoreRejectsBadBackups(t *testing.T) {
	svc := ledger.New(testdb.Pool(t))
	ctx := context.Background()
	a, b := newActor(t, svc), newActor(t, svc)
	seedEverything(t, svc, a)
	mustWallet(t, svc, b, "Keep me", "USD", 5_00)
	kept := canon(t, export(t, svc, b))

	good := export(t, svc, a)
	clone := func() *ledger.Backup {
		raw, _ := json.Marshal(good)
		var c ledger.Backup
		if err := json.Unmarshal(raw, &c); err != nil {
			t.Fatal(err)
		}
		return &c
	}
	cases := map[string]func(*ledger.Backup){
		"newer schema":      func(c *ledger.Backup) { c.Schema = 9999 },
		"unknown currency":  func(c *ledger.Backup) { c.Wallets[0].Currency = "XXX" },
		"dangling wallet":   func(c *ledger.Backup) { c.Transactions[0].WalletID = uuid.New() },
		"dangling label":    func(c *ledger.Backup) { c.Transactions[0].LabelIDs = []uuid.UUID{uuid.New()} },
		"duplicate id":      func(c *ledger.Backup) { c.Labels = append(c.Labels, c.Labels[0]) },
		"bad date":          func(c *ledger.Backup) { c.Transfers[0].OccurredOn = "15/10/2026" },
		"bad rate":          func(c *ledger.Backup) { c.Transfers[0].Rate = "forty" },
		"unknown kind":      func(c *ledger.Backup) { c.Transactions[0].Kind = "gift" },
		"name the DB fails": func(c *ledger.Backup) { c.Labels[0].Name = strings.Repeat("x", 31) },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			c := clone()
			mutate(c)
			var inv *ledger.InvalidError
			if _, err := svc.RestoreBackup(ctx, b, c); !errors.As(err, &inv) {
				t.Fatalf("err = %v, want InvalidError", err)
			}
			if canon(t, export(t, svc, b)) != kept {
				t.Fatal("a rejected restore changed the household")
			}
		})
	}

	var inv *ledger.InvalidError
	for name, content := range map[string][]byte{
		"not json":      []byte("hello"),
		"other format":  []byte(`{"format":"something-else","version":1}`),
		"newer version": []byte(`{"format":"ledger-01-backup","version":3}`),
		"broken gzip":   {0x1f, 0x8b, 0x00},
	} {
		if _, err := ledger.DecodeBackup(content); !errors.As(err, &inv) {
			t.Errorf("%s: err = %v, want InvalidError", name, err)
		}
	}
}

func TestExportTransactionsCsv(t *testing.T) {
	svc := ledger.New(testdb.Pool(t))
	ctx := context.Background()
	a := newActor(t, svc)
	seedEverything(t, svc, a)

	out, n, err := svc.ExportTransactionsCsv(ctx, a, ledger.CsvFilter{Today: day})
	if err != nil {
		t.Fatal(err)
	}
	text := string(out)
	if !strings.HasPrefix(text, "\xEF\xBB\xBFDate,Wallet,Type,Category,Amount,Currency,Labels,Note,Transfer wallet\n") {
		t.Fatalf("header = %q", strings.SplitN(text, "\n", 2)[0])
	}
	for _, want := range []string{
		"2026-10-15,Checking,expense,Groceries,-42.50,USD,Trip,Corner market,",
		"2026-10-15,Checking,transfer out,,-100.00,USD,,Exchange,Pesos",
		"2026-10-15,Pesos,transfer in,,4000.00,UYU,,Exchange,Checking",
		"2026-10-15,Pesos,expense,Transport,-1500.00,UYU,,Bus pass,",
	} {
		if !strings.Contains(text, want+"\n") {
			t.Errorf("missing row %q in\n%s", want, text)
		}
	}
	// The upcoming bus passes are left out.
	if strings.Count(text, "Bus pass") != 1 {
		t.Errorf("upcoming rows exported:\n%s", text)
	}
	if lines := strings.Count(text, "\n"); lines != n+1 {
		t.Errorf("%d lines for %d rows", lines, n)
	}

	from := on("2026-10-16")
	if _, n, err := svc.ExportTransactionsCsv(ctx, a, ledger.CsvFilter{Today: day, From: &from}); err != nil || n != 0 {
		t.Fatalf("from tomorrow = %d rows, %v", n, err)
	}
	// Another household's caller sees none of it.
	if _, n, err := svc.ExportTransactionsCsv(ctx, newActor(t, svc), ledger.CsvFilter{Today: day}); err != nil || n != 0 {
		t.Fatalf("stranger = %d rows, %v", n, err)
	}
}
