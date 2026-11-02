package ledger_test

import (
	"context"
	"testing"

	"github.com/franco-roura/ledger-01/backend/internal/importer"
	"github.com/franco-roura/ledger-01/backend/internal/ledger"
	"github.com/franco-roura/ledger-01/backend/internal/testdb"
	"github.com/google/uuid"
)

// A made-up Uruguayan statement: dd/mm/yyyy dates, comma decimals, one signed amount column.
const statementHeader = "Fecha;Concepto;Importe\n"

type importFixture struct {
	t       *testing.T
	svc     *ledger.Service
	a       ledger.Actor
	wallet  uuid.UUID
	expense uuid.UUID
	income  uuid.UUID
}

func newImportFixture(t *testing.T) importFixture {
	svc := ledger.New(testdb.Pool(t))
	a := newActor(t, svc)
	w := mustWallet(t, svc, a, "Northbank pesos", "UYU", 0)
	return importFixture{
		t: t, svc: svc, a: a, wallet: w.ID,
		expense: categoryNamed(t, svc, a, ledger.CategoryExpense, "Groceries").ID,
		income:  categoryNamed(t, svc, a, ledger.CategoryIncome, "Other").ID,
	}
}

// review uploads a statement body and reads every row with the guessed mapping.
func (f importFixture) review(body string) (uuid.UUID, importer.Mapping, []ledger.ImportRow) {
	f.t.Helper()
	ctx := context.Background()
	up, err := f.svc.UploadImport(ctx, f.a, f.wallet, "estado.csv", []byte(statementHeader+body))
	if err != nil {
		f.t.Fatal(err)
	}
	rows, _, err := f.svc.PreviewImport(ctx, f.a, up.BatchID, up.Mapping, 0)
	if err != nil {
		f.t.Fatal(err)
	}
	return up.BatchID, up.Mapping, rows
}

// commit imports every row that can be chosen and isn't a duplicate.
func (f importFixture) commit(batch uuid.UUID, m importer.Mapping, rows []ledger.ImportRow) int64 {
	f.t.Helper()
	var choices []ledger.ImportChoice
	for _, r := range rows {
		if r.Err != "" || r.Duplicate() || r.MergedInto >= 0 {
			continue
		}
		c := ledger.ImportChoice{Index: r.Index}
		switch {
		case r.DiscountTarget != nil:
		case r.Kind == ledger.KindExpense:
			c.CategoryID = f.expense
		default:
			c.CategoryID = f.income
		}
		choices = append(choices, c)
	}
	n, err := f.svc.CommitImport(context.Background(), f.a, batch, m, choices, true)
	if err != nil {
		f.t.Fatal(err)
	}
	return n
}

func (f importFixture) transactions() []ledger.Transaction {
	f.t.Helper()
	txs, _, err := f.svc.ListTransactions(context.Background(), f.a, ledger.Filter{
		WalletID: f.wallet, From: on("2026-01-01"), To: on("2026-12-31"), Today: on("2026-12-31"),
	}, 100, "")
	if err != nil {
		f.t.Fatal(err)
	}
	return txs
}

func TestImportRecognisesRenamedAndRepricedRows(t *testing.T) {
	f := newImportFixture(t)
	ctx := context.Background()
	body := "01/10/2026;COMPRA KIOSCO AZUL;-250,00\n02/10/2026;COMPRA PANADERIA SUR;-180,00\n"
	batch, m, rows := f.review(body)
	if n := f.commit(batch, m, rows); n != 2 {
		t.Fatalf("imported %d", n)
	}

	// Rename one and reprice the other.
	for _, tx := range f.transactions() {
		in := ledger.TransactionInput{WalletID: f.wallet, CategoryID: f.expense, Amount: tx.Amount, OccurredOn: tx.OccurredOn, Note: tx.Note}
		if tx.Note == "COMPRA KIOSCO AZUL" {
			in.Note = "Snacks"
		} else {
			in.Amount = 200_00
		}
		if _, err := f.svc.UpdateTransaction(ctx, f.a, tx.ID, in); err != nil {
			t.Fatal(err)
		}
	}

	_, _, rows = f.review(body)
	for _, r := range rows {
		if r.DupReason != ledger.DupImported {
			t.Fatalf("row %d = %q, want imported (matched %q)", r.Index, r.DupReason, r.MatchNote)
		}
	}
	if rows[0].MatchNote != "Snacks" {
		t.Fatalf("match note = %q", rows[0].MatchNote)
	}
}

func TestImportHeuristicsForTransactionsWithoutFingerprints(t *testing.T) {
	f := newImportFixture(t)
	ctx := context.Background()
	// Entered by hand: the same note on the same day, and a renamed one a day later.
	for _, in := range []ledger.TransactionInput{
		{WalletID: f.wallet, CategoryID: f.expense, Amount: 250_00, OccurredOn: on("2026-10-01"), Note: "compra kiosco  azul"},
		{WalletID: f.wallet, CategoryID: f.expense, Amount: 180_00, OccurredOn: on("2026-10-03"), Note: "Bread"},
	} {
		if _, err := f.svc.CreateTransaction(ctx, f.a, in); err != nil {
			t.Fatal(err)
		}
	}
	_, _, rows := f.review("01/10/2026;COMPRA KIOSCO AZUL;-250,00\n02/10/2026;COMPRA PANADERIA SUR;-180,00\n" +
		"02/10/2026;COMPRA PANADERIA SUR;-180,00\n05/10/2026;COMPRA FERRETERIA;-990,00\n")
	want := []string{ledger.DupSame, ledger.DupSimilar, ledger.DupFile, ""}
	for i, r := range rows {
		if r.DupReason != want[i] {
			t.Fatalf("row %d = %q, want %q", i, r.DupReason, want[i])
		}
	}
	if rows[1].MatchNote != "Bread" || !rows[1].MatchDate.Equal(on("2026-10-03")) {
		t.Fatalf("similar match = %q %s", rows[1].MatchNote, rows[1].MatchDate)
	}
}

func TestImportMergesRedivaIntoItsPurchase(t *testing.T) {
	f := newImportFixture(t)
	body := "01/10/2026;COMPRA MERCADO SOL;-500,00\n01/10/2026;COMPRA FARMACIA NORTE;-120,00\n" +
		"01/10/2026;REDIVA 19210MERCADO SOL;16,39\n"
	batch, m, rows := f.review(body)
	purchase, discount := rows[0], rows[2]
	if purchase.Amount != -483_61 || purchase.Gross != -500_00 || purchase.Discount != 16_39 || len(purchase.DiscountNotes) != 1 {
		t.Fatalf("purchase = %+v", purchase)
	}
	if discount.MergedInto != 0 || rows[1].Discount != 0 {
		t.Fatalf("discount = %+v", discount)
	}
	if n := f.commit(batch, m, rows); n != 2 {
		t.Fatalf("imported %d", n)
	}
	txs := f.transactions()
	if len(txs) != 2 {
		t.Fatalf("transactions = %d", len(txs))
	}
	for _, tx := range txs {
		if tx.Note == "COMPRA MERCADO SOL" && tx.Amount != 483_61 {
			t.Fatalf("purchase amount = %d", tx.Amount)
		}
	}

	// The same statement again: all three rows were imported, the discount included.
	_, _, rows = f.review(body)
	for _, r := range rows {
		if r.DupReason != ledger.DupImported || r.MergedInto >= 0 {
			t.Fatalf("row %d = %q merged %d", r.Index, r.DupReason, r.MergedInto)
		}
	}
}

func TestImportRedivaForAnEarlierPurchase(t *testing.T) {
	f := newImportFixture(t)
	// October's statement ends with the purchase; November's starts with its discount.
	batch, m, rows := f.review("31/10/2026;COMPRA MERCADO SOL;-500,00\n")
	f.commit(batch, m, rows)
	// Renamed in between: the bank description kept in the import refs still finds it.
	tx := f.transactions()[0]
	if _, err := f.svc.UpdateTransaction(context.Background(), f.a, tx.ID, ledger.TransactionInput{
		WalletID: f.wallet, CategoryID: f.expense, Amount: tx.Amount, OccurredOn: tx.OccurredOn, Note: "Weekly shop",
	}); err != nil {
		t.Fatal(err)
	}

	body := "02/11/2026;REDIVA 19210MERCADO SOL;16,39\n03/11/2026;COMPRA KIOSCO AZUL;-90,00\n"
	batch, m, rows = f.review(body)
	if rows[0].DiscountTarget == nil || *rows[0].DiscountTarget != tx.ID || rows[0].TargetNote != "Weekly shop" {
		t.Fatalf("discount row = %+v", rows[0])
	}
	f.commit(batch, m, rows)
	for _, got := range f.transactions() {
		if got.ID == tx.ID && got.Amount != 483_61 {
			t.Fatalf("purchase after its discount = %d", got.Amount)
		}
	}
	if len(f.transactions()) != 2 {
		t.Fatalf("transactions = %d", len(f.transactions()))
	}

	// Again: the discount was applied already.
	_, _, rows = f.review(body)
	if rows[0].DupReason != ledger.DupImported || rows[0].DiscountTarget != nil {
		t.Fatalf("again = %+v", rows[0])
	}
}

// Charged twice and refunded once: the first charge matches what was entered by hand, the
// second and its refund cancel out and start unticked.
func TestImportDoubleChargeAndRefund(t *testing.T) {
	f := newImportFixture(t)
	if _, err := f.svc.CreateTransaction(context.Background(), f.a, ledger.TransactionInput{
		WalletID: f.wallet, CategoryID: f.expense, Amount: 7_40, OccurredOn: on("2026-10-02"), Note: "Ride home",
	}); err != nil {
		t.Fatal(err)
	}
	_, _, rows := f.review("02/10/2026;COMPRA PAY*TRANS*BUS;-7,40\n03/10/2026;COMPRA KIOSCO AZUL;-90,00\n" +
		"02/10/2026;COMPRA PAY*TRANS*BUS;-7,40\n02/10/2026;DEVOLUCION PAY*TRANS*BUS;7,40\n")
	if rows[0].DupReason != ledger.DupSimilar || rows[0].MatchNote != "Ride home" {
		t.Fatalf("first charge = %q %q", rows[0].DupReason, rows[0].MatchNote)
	}
	if rows[2].RefundedBy != 3 || rows[3].RefundOf != 2 || !rows[2].LeftOut() || !rows[3].LeftOut() {
		t.Fatalf("refund pair = %+v / %+v", rows[2], rows[3])
	}
	if rows[2].Duplicate() || rows[3].Duplicate() || rows[1].LeftOut() || rows[1].Duplicate() {
		t.Fatalf("flags = %+v", rows)
	}
}

func TestImportRefundOfAnEarlierPurchase(t *testing.T) {
	f := newImportFixture(t)
	batch, m, rows := f.review("30/10/2026;COMPRA TIENDA LUNA;-800,00\n")
	f.commit(batch, m, rows)
	_, _, rows = f.review("02/11/2026;DEVOLUCION TIENDA LUNA;800,00\n04/11/2026;DEVOLUCION TIENDA LUNA;50,00\n")
	if rows[0].RefundsNote != "COMPRA TIENDA LUNA" || !rows[0].LeftOut() || rows[0].DiscountTarget != nil {
		t.Fatalf("full refund = %+v", rows[0])
	}
	// The purchase is taken by the full refund, so the partial one is a plain income row.
	if rows[1].DiscountTarget != nil || rows[1].LeftOut() {
		t.Fatalf("second refund = %+v", rows[1])
	}
}

// The bank decides the wording: detected as Itaú, the REDIVA row merges; with "Other bank" it's a
// plain income row; an unknown bank is refused.
func TestImportBankChoice(t *testing.T) {
	f := newImportFixture(t)
	ctx := context.Background()
	up, err := f.svc.UploadImport(ctx, f.a, f.wallet, "estado.csv",
		[]byte(statementHeader+"01/10/2026;COMPRA MERCADO SOL;-500,00\n01/10/2026;REDIVA 19210MERCADO SOL;16,39\n"))
	if err != nil {
		t.Fatal(err)
	}
	if up.Mapping.Bank != "uy-itau" {
		t.Fatalf("detected bank = %q", up.Mapping.Bank)
	}
	m := up.Mapping
	m.Bank = "generic"
	rows, _, err := f.svc.PreviewImport(ctx, f.a, up.BatchID, m, 0)
	if err != nil {
		t.Fatal(err)
	}
	if rows[1].MergedInto >= 0 || rows[0].Discount != 0 {
		t.Fatalf("generic merged: %+v", rows[1])
	}
	m.Bank = "xx-nowhere"
	if _, _, err := f.svc.PreviewImport(ctx, f.a, up.BatchID, m, 0); err == nil {
		t.Fatal("unknown bank accepted")
	}
}
