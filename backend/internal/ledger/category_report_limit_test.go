package ledger_test

import (
	"context"
	"testing"

	"github.com/franco-roura/ledger-01/backend/internal/ledger"
	"github.com/franco-roura/ledger-01/backend/internal/testdb"
	"github.com/google/uuid"
)

func TestCategoryReportFiltersBeforeTransactionLimit(t *testing.T) {
	svc := ledger.New(testdb.Pool(t))
	ctx := context.Background()
	a := newActor(t, svc)
	usd := mustWallet(t, svc, a, "Daily", "USD", 0)
	uyu := mustWallet(t, svc, a, "Pesos", "UYU", 0)
	cat := categoryNamed(t, svc, a, ledger.CategoryExpense, "Groceries")
	old := day.AddDate(0, 0, -1)
	if _, err := svc.CreateTransfer(ctx, a, ledger.TransferInput{
		FromWalletID: usd.ID, ToWalletID: uyu.ID, FromAmount: 100_00,
		ToAmount: 4000_00, Rate: "40", OccurredOn: old,
	}); err != nil {
		t.Fatal(err)
	}
	label, err := svc.CreateLabel(ctx, a, "Market", "cat-teal")
	if err != nil {
		t.Fatal(err)
	}
	match, err := svc.CreateTransaction(ctx, a, ledger.TransactionInput{
		WalletID: uyu.ID, CategoryID: cat.ID, Amount: 2000_00, OccurredOn: old,
		Note: "Needle", LabelIDs: []uuid.UUID{label.ID},
	})
	if err != nil {
		t.Fatal(err)
	}
	for range 200 {
		if _, err := svc.CreateTransaction(ctx, a, ledger.TransactionInput{
			WalletID: usd.ID, CategoryID: cat.ID, Amount: 100_00, OccurredOn: day,
			Note: "Ordinary expense",
		}); err != nil {
			t.Fatal(err)
		}
	}
	max := int64(50_00) // 2,000 UYU must compare as 50 USD, not its native amount.
	for _, tc := range []struct {
		name string
		note string
		max  *int64
	}{
		{name: "note", note: "needle"},
		{name: "converted amount", max: &max},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r, err := svc.CategoryReportFor(ctx, a, cat.ID, ledger.ReportFilter{
				From: old, To: day, Today: day, NoteQuery: tc.note, MaxAmount: tc.max,
			}, ledger.BucketDay)
			if err != nil {
				t.Fatal(err)
			}
			if r.Total != 50_00 || r.Count != 1 || len(r.Transactions) != 1 {
				t.Fatalf("filtered report total=%d count=%d listed=%d", r.Total, r.Count, len(r.Transactions))
			}
			got := r.Transactions[0]
			if got.ID != match.ID || len(got.LabelIDs) != 1 || got.LabelIDs[0] != label.ID || got.Currency != "UYU" {
				t.Fatalf("matching row details changed: %+v", got)
			}
		})
	}
	r, err := svc.CategoryReportFor(ctx, a, cat.ID, ledger.ReportFilter{
		From: old, To: day, Today: day,
	}, ledger.BucketDay)
	if err != nil {
		t.Fatal(err)
	}
	if r.Count != 201 || len(r.Transactions) != 200 {
		t.Fatalf("unfiltered report count=%d listed=%d", r.Count, len(r.Transactions))
	}
	for _, tx := range r.Transactions {
		if tx.ID == match.ID {
			t.Fatal("transaction limit did not retain the newest rows")
		}
	}
}
