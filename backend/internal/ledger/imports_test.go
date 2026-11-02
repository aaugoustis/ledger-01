package ledger_test

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/franco-roura/ledger-01/backend/internal/importer"
	"github.com/franco-roura/ledger-01/backend/internal/ledger"
	"github.com/franco-roura/ledger-01/backend/internal/testdb"
	"github.com/google/uuid"
)

func bankFile(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile("../importer/testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestImportReviewAndCommit(t *testing.T) {
	svc := ledger.New(testdb.Pool(t))
	ctx := context.Background()
	a := newActor(t, svc)
	w := mustWallet(t, svc, a, "Checking", "USD", 0)
	groceries := categoryNamed(t, svc, a, ledger.CategoryExpense, "Groceries")
	transport := categoryNamed(t, svc, a, ledger.CategoryExpense, "Transport")
	home := categoryNamed(t, svc, a, ledger.CategoryExpense, "Home")
	salary := categoryNamed(t, svc, a, ledger.CategoryIncome, "Salary")
	other := categoryNamed(t, svc, a, ledger.CategoryIncome, "Other")

	// Already in the wallet: one of the two grocery rows.
	if _, err := svc.CreateTransaction(ctx, a, ledger.TransactionInput{
		WalletID: w.ID, CategoryID: groceries.ID, Amount: 45_20, OccurredOn: on("2026-09-03"), Note: "corner  grocer",
	}); err != nil {
		t.Fatal(err)
	}
	// Rules settle the transit row (the longest matching pattern wins); a rule pointing at an
	// income category doesn't touch expenses.
	if _, err := svc.CreateCategoryRule(ctx, a, " Transit ", transport.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.CreateCategoryRule(ctx, a, "transit card", home.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.CreateCategoryRule(ctx, a, "Riverside", salary.ID); err != nil {
		t.Fatal(err)
	}

	up, err := svc.UploadImport(ctx, a, w.ID, `C:\Downloads\northbank.csv`, bankFile(t, "northbank.csv"))
	if err != nil {
		t.Fatal(err)
	}
	if up.SavedMapping || up.Rows != 7 || up.Columns != 4 || len(up.Sample) != 7 || up.Mapping.Amount != 2 {
		t.Fatalf("upload = %+v", up)
	}

	rows, total, err := svc.PreviewImport(ctx, a, up.BatchID, up.Mapping, 0)
	if err != nil {
		t.Fatal(err)
	}
	if total != 6 || len(rows) != 6 {
		t.Fatalf("preview %d rows of %d", len(rows), total)
	}
	type view struct {
		kind ledger.TransactionKind
		cat  *uuid.UUID
		src  string
		dup  bool
	}
	got := func(r ledger.ImportRow) view { return view{r.Kind, r.CategoryID, r.CategorySource, r.Duplicate()} }
	for i, want := range []view{
		{ledger.KindIncome, nil, "", false},
		{ledger.KindExpense, nil, "", true}, // in the wallet already
		{ledger.KindExpense, nil, "", true}, // repeats the row above
		{ledger.KindExpense, &home.ID, "rule", false},
		{ledger.KindIncome, nil, "", false},
		{ledger.KindExpense, nil, "", false}, // "Riverside" points at an income category
	} {
		g := got(rows[i])
		if g.kind != want.kind || g.src != want.src || g.dup != want.dup || (want.cat != nil) != (g.cat != nil) || (g.cat != nil && *g.cat != *want.cat) {
			t.Errorf("row %d = %+v, want %+v", i, g, want)
		}
	}
	// A live preview of the first rows only.
	if few, total, _ := svc.PreviewImport(ctx, a, up.BatchID, up.Mapping, 2); len(few) != 2 || total != 6 {
		t.Fatalf("limited preview = %d of %d", len(few), total)
	}

	// Wrong kind and missing categories are refused, and nothing is written.
	var inv *ledger.InvalidError
	for _, bad := range [][]ledger.ImportChoice{
		{{Index: 0, CategoryID: groceries.ID}},
		{{Index: 3, CategoryID: uuid.New()}},
		{{Index: 3, CategoryID: transport.ID}, {Index: 3, CategoryID: transport.ID}},
		{{Index: 9, CategoryID: transport.ID}},
		{},
	} {
		if _, err := svc.CommitImport(ctx, a, up.BatchID, up.Mapping, bad, false); !errors.As(err, &inv) {
			t.Errorf("commit %+v: err = %v", bad, err)
		}
	}

	n, err := svc.CommitImport(ctx, a, up.BatchID, up.Mapping, []ledger.ImportChoice{
		{Index: 0, CategoryID: other.ID}, {Index: 3, CategoryID: transport.ID},
		{Index: 4, CategoryID: salary.ID}, {Index: 5, CategoryID: home.ID},
	}, true)
	if err != nil || n != 4 {
		t.Fatalf("commit = %d, %v", n, err)
	}
	got2, err := svc.GetWalletOn(ctx, a, w.ID, on("2026-10-01"))
	if err != nil {
		t.Fatal(err)
	}
	if want := int64(1500_00 - 45_20 - 20_00 + 2400_00 - 1200_00); got2.Balance != want {
		t.Fatalf("balance = %d, want %d", got2.Balance, want)
	}
	// The upload is gone once imported.
	if _, _, err := svc.PreviewImport(ctx, a, up.BatchID, up.Mapping, 0); !errors.As(err, &inv) || inv.Field != "batch_id" {
		t.Fatalf("preview after commit: %v", err)
	}

	// The next upload of the same export suggests the saved mapping, and now flags the imported
	// rows as duplicates.
	again, err := svc.UploadImport(ctx, a, w.ID, "northbank.csv", bankFile(t, "northbank.csv"))
	if err != nil {
		t.Fatal(err)
	}
	if !again.SavedMapping || again.Mapping != up.Mapping {
		t.Fatalf("saved mapping = %v %+v", again.SavedMapping, again.Mapping)
	}
	rows, _, err = svc.PreviewImport(ctx, a, again.BatchID, again.Mapping, 0)
	if err != nil {
		t.Fatal(err)
	}
	for i, r := range rows {
		if !r.Duplicate() {
			t.Errorf("row %d should be a duplicate now", i)
		}
	}
}

func TestImportDebitCreditAndCategoryColumn(t *testing.T) {
	svc := ledger.New(testdb.Pool(t))
	ctx := context.Background()
	a := newActor(t, svc)

	// Spanish-style export into a UYU wallet: debit and credit columns, decimal commas.
	pesos := mustWallet(t, svc, a, "Pesos", "UYU", 0)
	up, err := svc.UploadImport(ctx, a, pesos.ID, "movimientos.csv", bankFile(t, "bancosur.csv"))
	if err != nil {
		t.Fatal(err)
	}
	rows, _, err := svc.PreviewImport(ctx, a, up.BatchID, up.Mapping, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 4 || rows[1].Amount != -1250_50 || rows[1].Note != "Almacén Central" || rows[3].Kind != ledger.KindIncome {
		t.Fatalf("rows = %+v", rows)
	}

	// A card statement whose category column names the household's categories.
	card := mustWallet(t, svc, a, "Card", "USD", 0)
	up, err = svc.UploadImport(ctx, a, card.ID, "card.tsv", bankFile(t, "card.tsv"))
	if err != nil {
		t.Fatal(err)
	}
	m := up.Mapping
	m.Sign = importer.PositiveIsExpense
	rows, _, err = svc.PreviewImport(ctx, a, up.BatchID, m, 0)
	if err != nil {
		t.Fatal(err)
	}
	car := categoryNamed(t, svc, a, ledger.CategoryExpense, "Car")
	if rows[1].CategoryID == nil || *rows[1].CategoryID != car.ID || rows[1].CategorySource != "column" {
		t.Fatalf("category column: %+v", rows[1])
	}
	if rows[3].Kind != ledger.KindIncome || rows[3].CategoryID != nil {
		t.Fatalf("payment row: %+v", rows[3])
	}

	// A bad mapping names the field; another household can't see the upload.
	bad := m
	bad.Date = 9
	var inv *ledger.InvalidError
	if _, _, err := svc.PreviewImport(ctx, a, up.BatchID, bad, 0); !errors.As(err, &inv) || inv.Field != "mapping.date_column" {
		t.Fatalf("bad mapping: %v", err)
	}
	stranger := newActor(t, svc)
	if _, _, err := svc.PreviewImport(ctx, stranger, up.BatchID, m, 0); !errors.As(err, &inv) || inv.Field != "batch_id" {
		t.Fatalf("other household: %v", err)
	}
	if _, err := svc.UploadImport(ctx, stranger, card.ID, "card.tsv", bankFile(t, "card.tsv")); !errors.Is(err, ledger.ErrNotFound) {
		t.Fatalf("upload into another household's wallet: %v", err)
	}

	// Unsupported files are refused at upload.
	if _, err := svc.UploadImport(ctx, a, card.ID, "old.xls", []byte{0xD0, 0xCF, 0x11, 0xE0}); !errors.As(err, &inv) || inv.Field != "content" {
		t.Fatalf("xls upload: %v", err)
	}

	// The job purges expired uploads.
	if n, err := svc.PurgeExpiredImports(ctx, time.Now().Add(25*time.Hour)); err != nil || n < 2 {
		t.Fatalf("purge = %d, %v", n, err)
	}
	if _, _, err := svc.PreviewImport(ctx, a, up.BatchID, m, 0); !errors.As(err, &inv) || inv.Field != "batch_id" {
		t.Fatalf("preview after purge: %v", err)
	}
}

func TestCategoryRules(t *testing.T) {
	svc := ledger.New(testdb.Pool(t))
	ctx := context.Background()
	a := newActor(t, svc)
	food := categoryNamed(t, svc, a, ledger.CategoryExpense, "Food & Drink")
	groceries := categoryNamed(t, svc, a, ledger.CategoryExpense, "Groceries")

	r, err := svc.CreateCategoryRule(ctx, a, "  Sunrise   Bakery ", food.ID)
	if err != nil || r.Pattern != "Sunrise Bakery" {
		t.Fatalf("create = %+v, %v", r, err)
	}
	var conf *ledger.ConflictError
	if _, err := svc.CreateCategoryRule(ctx, a, "sunrise bakery", groceries.ID); !errors.As(err, &conf) {
		t.Fatalf("duplicate pattern: %v", err)
	}
	var inv *ledger.InvalidError
	if _, err := svc.CreateCategoryRule(ctx, a, " ", food.ID); !errors.As(err, &inv) {
		t.Fatalf("empty pattern: %v", err)
	}
	if _, err := svc.CreateCategoryRule(ctx, a, "x", uuid.New()); !errors.As(err, &inv) {
		t.Fatalf("unknown category: %v", err)
	}

	// Merging categories carries the rule along.
	if _, _, err := svc.MergeCategories(ctx, a, []uuid.UUID{food.ID}, groceries.ID); err != nil {
		t.Fatal(err)
	}
	rules, err := svc.ListCategoryRules(ctx, a)
	if err != nil || len(rules) != 1 || rules[0].CategoryID != groceries.ID {
		t.Fatalf("rules after merge = %+v, %v", rules, err)
	}
	if err := svc.DeleteCategoryRule(ctx, newActor(t, svc), r.ID); !errors.Is(err, ledger.ErrNotFound) {
		t.Fatalf("delete from another household: %v", err)
	}
	if err := svc.DeleteCategoryRule(ctx, a, r.ID); err != nil {
		t.Fatal(err)
	}
}
