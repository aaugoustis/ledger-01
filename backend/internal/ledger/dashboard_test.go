package ledger_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/franco-roura/ledger-01/backend/internal/ledger"
	"github.com/franco-roura/ledger-01/backend/internal/testdb"
	"github.com/google/uuid"
)

func TestDashboardConvertsWithTransferRates(t *testing.T) {
	svc := ledger.New(testdb.Pool(t))
	ctx := context.Background()
	a := newActor(t, svc) // base currency USD
	usd := mustWallet(t, svc, a, "Northbank", "USD", 1000_00)
	uyu := mustWallet(t, svc, a, "Northbank pesos", "UYU", 0)
	eur := mustWallet(t, svc, a, "Travel", "EUR", 50_00)
	food := categoryNamed(t, svc, a, ledger.CategoryExpense, "Food & Drink")
	salary := categoryNamed(t, svc, a, ledger.CategoryIncome, "Salary")

	on := func(s string) time.Time { v, _ := time.Parse(time.DateOnly, s); return v }
	// 100 USD -> 4000 UYU sets the rate at 40.
	if _, err := svc.CreateTransfer(ctx, a, ledger.TransferInput{
		FromWalletID: usd.ID, ToWalletID: uyu.ID, FromAmount: 100_00, ToAmount: 4000_00, Rate: "40", OccurredOn: on("2026-10-02"),
	}); err != nil {
		t.Fatal(err)
	}
	for _, in := range []ledger.TransactionInput{
		{WalletID: uyu.ID, CategoryID: food.ID, Amount: 800_00, OccurredOn: on("2026-10-05")}, // 20 USD
		{WalletID: usd.ID, CategoryID: food.ID, Amount: 30_00, OccurredOn: on("2026-10-06")},
		{WalletID: usd.ID, CategoryID: salary.ID, Amount: 500_00, OccurredOn: on("2026-10-10")},
		{WalletID: eur.ID, CategoryID: food.ID, Amount: 10_00, OccurredOn: on("2026-10-11")}, // no EUR rate
	} {
		if _, err := svc.CreateTransaction(ctx, a, in); err != nil {
			t.Fatal(err)
		}
	}

	f := ledger.ReportFilter{From: on("2026-10-01"), To: on("2026-10-31")}
	got, err := svc.Dashboard(ctx, a, f, ledger.BucketWeek)
	if err != nil {
		t.Fatal(err)
	}
	if got.Currency != "USD" {
		t.Fatalf("currency = %s", got.Currency)
	}
	if got.Expenses != 50_00 || got.Income != 500_00 {
		t.Fatalf("expenses=%d income=%d, want 5000/50000", got.Expenses, got.Income)
	}
	// USD 1000 - 100 - 30 + 500 = 1370; UYU 4000 - 800 = 3200 -> 80 USD. EUR is left out.
	if got.Balance != 1450_00 {
		t.Fatalf("balance = %d, want 145000", got.Balance)
	}
	if got.TransfersOut != 100_00 || got.TransfersIn != 100_00 || got.TransfersInCount != 1 {
		t.Fatalf("transfers in=%d out=%d", got.TransfersIn, got.TransfersOut)
	}
	if len(got.Unconverted) != 1 || got.Unconverted[0] != "EUR" {
		t.Fatalf("unconverted = %v", got.Unconverted)
	}
	if len(got.ExpenseCategories) != 1 || got.ExpenseCategories[0].Amount != 50_00 || got.ExpenseCategories[0].Count != 2 {
		t.Fatalf("expense categories = %+v", got.ExpenseCategories)
	}
	if len(got.Series) != 5 || got.Series[len(got.Series)-1].Balance != got.Balance {
		t.Fatalf("series = %+v", got.Series)
	}

	// Wallet and category filters.
	only, err := svc.Dashboard(ctx, a, ledger.ReportFilter{From: f.From, To: f.To, WalletIDs: []uuid.UUID{usd.ID}, CategoryIDs: []uuid.UUID{food.ID}}, ledger.BucketMonth)
	if err != nil {
		t.Fatal(err)
	}
	if only.Expenses != 30_00 || only.Income != 0 || only.Balance != 1370_00 || len(only.Unconverted) != 0 {
		t.Fatalf("filtered = %+v", only)
	}

	rep, err := svc.CategoryReportFor(ctx, a, food.ID, f, ledger.BucketDay)
	if err != nil {
		t.Fatal(err)
	}
	if rep.Total != 50_00 || rep.Count != 2 || rep.Average != 25_00 || len(rep.Transactions) != 3 {
		t.Fatalf("category report = total %d count %d avg %d txs %d", rep.Total, rep.Count, rep.Average, len(rep.Transactions))
	}
	if rep.BusiestWeekday < 0 {
		t.Fatal("busiest weekday not set")
	}
}

func TestDashboardNoteAndAmountFilters(t *testing.T) {
	svc := ledger.New(testdb.Pool(t))
	ctx := context.Background()
	a := newActor(t, svc) // base currency USD
	usd := mustWallet(t, svc, a, "Checking", "USD", 1000_00)
	uyu := mustWallet(t, svc, a, "Pesos", "UYU", 0)
	food := categoryNamed(t, svc, a, ledger.CategoryExpense, "Food & Drink")
	on := func(s string) time.Time { v, _ := time.Parse(time.DateOnly, s); return v }

	// 100 USD -> 4000 UYU (rate 40), noted "Allowance".
	if _, err := svc.CreateTransfer(ctx, a, ledger.TransferInput{
		FromWalletID: usd.ID, ToWalletID: uyu.ID, FromAmount: 100_00, ToAmount: 4000_00, Rate: "40", OccurredOn: on("2026-10-02"), Note: "Allowance",
	}); err != nil {
		t.Fatal(err)
	}
	for _, in := range []ledger.TransactionInput{
		{WalletID: usd.ID, CategoryID: food.ID, Amount: 12_00, OccurredOn: on("2026-10-05"), Note: "Corner Grocer"},
		{WalletID: usd.ID, CategoryID: food.ID, Amount: 80_00, OccurredOn: on("2026-10-06"), Note: "Corner grocer weekly shop"},
		{WalletID: uyu.ID, CategoryID: food.ID, Amount: 2000_00, OccurredOn: on("2026-10-07"), Note: "Feria"}, // 50 USD
		{WalletID: usd.ID, CategoryID: food.ID, Amount: 5_00, OccurredOn: on("2026-10-08"), Note: "Coffee"},
	} {
		if _, err := svc.CreateTransaction(ctx, a, in); err != nil {
			t.Fatal(err)
		}
	}
	ptr := func(v int64) *int64 { return &v }
	f := ledger.ReportFilter{From: on("2026-10-01"), To: on("2026-10-31"), Today: on("2026-10-31")}
	all, err := svc.Dashboard(ctx, a, f, ledger.BucketWeek)
	if err != nil {
		t.Fatal(err)
	}

	// The note matches case-insensitively; balances don't move with filters.
	f.NoteQuery = "GROCER"
	got, err := svc.Dashboard(ctx, a, f, ledger.BucketWeek)
	if err != nil {
		t.Fatal(err)
	}
	if got.Expenses != 92_00 || got.TransfersOutCount != 0 || got.Balance != all.Balance || got.Change != -92_00 {
		t.Fatalf("note filter: expenses %d, transfers %d, balance %d (all %d), change %d", got.Expenses, got.TransfersOutCount, got.Balance, all.Balance, got.Change)
	}

	// Amounts compare in the report currency: the 2,000 UYU feria is 50 USD.
	f.NoteQuery, f.MinAmount, f.MaxAmount = "", ptr(10_00), ptr(60_00)
	if got, err = svc.Dashboard(ctx, a, f, ledger.BucketWeek); err != nil {
		t.Fatal(err)
	}
	if got.Expenses != 62_00 || got.TransfersOutCount != 0 {
		t.Fatalf("amount filter: expenses %d, transfers out %d", got.Expenses, got.TransfersOutCount)
	}
	cr, err := svc.CategoryReportFor(ctx, a, food.ID, f, ledger.BucketWeek)
	if err != nil || cr.Total != 62_00 || cr.Count != 2 || len(cr.Transactions) != 2 {
		t.Fatalf("category report: total %d count %d rows %d, %v", cr.Total, cr.Count, len(cr.Transactions), err)
	}

	f.MinAmount, f.MaxAmount = ptr(70_00), ptr(60_00)
	var inv *ledger.InvalidError
	if _, err := svc.Dashboard(ctx, a, f, ledger.BucketWeek); !errors.As(err, &inv) {
		t.Fatalf("min above max: %v", err)
	}
}

func TestWalletOverviewInItsOwnCurrency(t *testing.T) {
	svc := ledger.New(testdb.Pool(t))
	ctx := context.Background()
	a := newActor(t, svc) // base currency USD
	uyu := mustWallet(t, svc, a, "Pesos", "UYU", 1000_00)
	food := categoryNamed(t, svc, a, ledger.CategoryExpense, "Food & Drink")
	on := func(s string) time.Time { v, _ := time.Parse(time.DateOnly, s); return v }
	if _, err := svc.CreateTransaction(ctx, a, ledger.TransactionInput{WalletID: uyu.ID, CategoryID: food.ID, Amount: 250_00, OccurredOn: on("2026-10-03")}); err != nil {
		t.Fatal(err)
	}
	// No UYU<->USD transfer exists, so only a UYU report can count this wallet.
	f := ledger.ReportFilter{From: on("2026-10-01"), To: on("2026-10-31"), Today: on("2026-10-31"), WalletIDs: []uuid.UUID{uyu.ID}, Currency: "uyu"}
	got, err := svc.Dashboard(ctx, a, f, ledger.BucketDay)
	if err != nil {
		t.Fatal(err)
	}
	if got.Currency != "UYU" || got.Balance != 750_00 || got.Expenses != 250_00 || len(got.Unconverted) != 0 {
		t.Fatalf("overview = %s balance %d expenses %d unconverted %v", got.Currency, got.Balance, got.Expenses, got.Unconverted)
	}
	f.Currency = "XXX"
	var inv *ledger.InvalidError
	if _, err := svc.Dashboard(ctx, a, f, ledger.BucketDay); !errors.As(err, &inv) || inv.Field != "currency" {
		t.Fatalf("unknown currency: %v", err)
	}
}
