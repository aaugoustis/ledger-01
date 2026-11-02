package ledger_test

import (
	"context"
	"testing"
	"time"

	"github.com/franco-roura/ledger-01/backend/internal/ledger"
	"github.com/franco-roura/ledger-01/backend/internal/testdb"
	"github.com/google/uuid"
)

func TestSearchTransactionsAcrossWallets(t *testing.T) {
	svc := ledger.New(testdb.Pool(t))
	ctx := context.Background()
	a := newActor(t, svc)
	cash := mustWallet(t, svc, a, "Cash", "USD", 0)
	card := mustWallet(t, svc, a, "Card", "USD", 0)
	food := categoryNamed(t, svc, a, ledger.CategoryExpense, "Food & Drink")
	treat, err := svc.CreateLabel(ctx, a, "Treat", "cat-teal")
	if err != nil {
		t.Fatal(err)
	}
	day := func(s string) time.Time { d, _ := time.Parse(time.DateOnly, s); return d }
	expense := func(w ledger.Wallet, on, note string, amount int64, labels ...uuid.UUID) {
		t.Helper()
		if _, err := svc.CreateTransaction(ctx, a, ledger.TransactionInput{
			WalletID: w.ID, CategoryID: food.ID, Amount: amount, OccurredOn: day(on), Note: note, LabelIDs: labels,
		}); err != nil {
			t.Fatal(err)
		}
	}
	expense(cash, "2026-09-05", "Pizza place", 1500, treat.ID)
	expense(card, "2026-10-02", "PIZZA delivery", 2200)
	expense(card, "2026-10-10", "Groceries", 9000)
	expense(card, "2026-11-20", "Pizza (planned)", 1800) // after today
	if _, err := svc.CreateTransfer(ctx, a, ledger.TransferInput{
		FromWalletID: card.ID, ToWalletID: cash.ID, FromAmount: 5000, ToAmount: 5000, OccurredOn: day("2026-10-15"),
	}); err != nil {
		t.Fatal(err)
	}
	base := ledger.Search{Filter: ledger.Filter{From: day("2026-01-01"), To: day("2026-12-31"), Today: day("2026-10-30")}}

	notes := func(sr ledger.Search, pageSize int, token string) ([]string, string) {
		t.Helper()
		ts, next, err := svc.SearchTransactions(ctx, a, sr, pageSize, token)
		if err != nil {
			t.Fatal(err)
		}
		out := make([]string, len(ts))
		for i, tx := range ts {
			out[i] = tx.Note
		}
		return out, next
	}

	pizza := base
	pizza.NoteQuery = "pizza"
	if got, _ := notes(pizza, 0, ""); len(got) != 2 || got[0] != "PIZZA delivery" || got[1] != "Pizza place" {
		t.Fatalf("pizza across wallets, upcoming left out = %v", got)
	}

	labelled := base
	labelled.LabelIDs = []uuid.UUID{treat.ID}
	if got, _ := notes(labelled, 0, ""); len(got) != 1 || got[0] != "Pizza place" {
		t.Fatalf("by label = %v", got)
	}

	inCash := base
	inCash.WalletIDs = []uuid.UUID{cash.ID}
	inCash.Kinds = []ledger.TransactionKind{ledger.KindTransferIn}
	if got, _ := notes(inCash, 0, ""); len(got) != 1 {
		t.Fatalf("transfers into cash = %v", got)
	}

	big := base
	min := int64(2000)
	big.MinAmount = &min
	big.Kinds = []ledger.TransactionKind{ledger.KindExpense}
	if got, _ := notes(big, 0, ""); len(got) != 2 {
		t.Fatalf("expenses of 20.00 or more = %v", got)
	}

	// Pages follow on from the cursor.
	expenses := base
	expenses.Kinds = []ledger.TransactionKind{ledger.KindExpense}
	page1, next := notes(expenses, 2, "")
	page2, last := notes(expenses, 2, next)
	if len(page1) != 2 || page1[0] != "Groceries" || next == "" || len(page2) != 1 || last != "" || page2[0] != "Pizza place" {
		t.Fatalf("pages = %v %q / %v %q", page1, next, page2, last)
	}

	// Another household's search sees none of it.
	other := newActor(t, svc)
	stranger := base
	stranger.WalletIDs = []uuid.UUID{cash.ID, card.ID}
	if ts, _, err := svc.SearchTransactions(ctx, other, stranger, 0, ""); err != nil || len(ts) != 0 {
		t.Fatalf("other household = %d rows, %v", len(ts), err)
	}

	bad := base
	bad.Kinds = []ledger.TransactionKind{"refund"}
	if _, _, err := svc.SearchTransactions(ctx, a, bad, 0, ""); err == nil {
		t.Fatal("unknown kind accepted")
	}
}
