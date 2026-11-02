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

func TestBudgetSpendingAndHistory(t *testing.T) {
	svc := ledger.New(testdb.Pool(t))
	ctx := context.Background()
	a := newActor(t, svc)
	usd := mustWallet(t, svc, a, "Northbank", "USD", 1000_00)
	uyu := mustWallet(t, svc, a, "Northbank pesos", "UYU", 0)
	other := mustWallet(t, svc, a, "Savings", "USD", 0)
	food := categoryNamed(t, svc, a, ledger.CategoryExpense, "Food & Drink")
	groceries := categoryNamed(t, svc, a, ledger.CategoryExpense, "Groceries")
	shopping := categoryNamed(t, svc, a, ledger.CategoryExpense, "Shopping")
	salary := categoryNamed(t, svc, a, ledger.CategoryIncome, "Salary")
	on := func(s string) time.Time { v, _ := time.Parse(time.DateOnly, s); return v }

	// 100 USD -> 4000 UYU: UYU expenses count at 1/40.
	if _, err := svc.CreateTransfer(ctx, a, ledger.TransferInput{
		FromWalletID: usd.ID, ToWalletID: uyu.ID, FromAmount: 100_00, ToAmount: 4000_00, Rate: "40", OccurredOn: on("2026-09-01"),
	}); err != nil {
		t.Fatal(err)
	}
	for _, in := range []ledger.TransactionInput{
		{WalletID: usd.ID, CategoryID: food.ID, Amount: 30_00, OccurredOn: on("2026-09-20")},       // September
		{WalletID: usd.ID, CategoryID: food.ID, Amount: 40_00, OccurredOn: on("2026-10-03")},       // counted
		{WalletID: uyu.ID, CategoryID: groceries.ID, Amount: 800_00, OccurredOn: on("2026-10-05")}, // 20 USD
		{WalletID: usd.ID, CategoryID: shopping.ID, Amount: 99_00, OccurredOn: on("2026-10-06")},   // other category
		{WalletID: other.ID, CategoryID: food.ID, Amount: 99_00, OccurredOn: on("2026-10-07")},     // other wallet
		{WalletID: usd.ID, CategoryID: salary.ID, Amount: 500_00, OccurredOn: on("2026-10-08")},    // income
	} {
		if _, err := svc.CreateTransaction(ctx, a, in); err != nil {
			t.Fatal(err)
		}
	}

	b, err := svc.CreateBudget(ctx, a, ledger.BudgetInput{
		Name: "  Food  ", Amount: 300_00, Currency: "usd", Repeat: ledger.RepeatMonthly, Start: on("2026-08-01"),
		WalletIDs: []uuid.UUID{usd.ID, uyu.ID}, CategoryIDs: []uuid.UUID{food.ID, groceries.ID, food.ID},
	})
	if err != nil {
		t.Fatal(err)
	}
	if b.Name != "Food" || b.Currency != "USD" || len(b.CategoryIDs) != 2 || len(b.WalletIDs) != 2 {
		t.Fatalf("created %+v", b)
	}

	today := on("2026-10-22")
	list, unconverted, err := svc.ListBudgets(ctx, a, today)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || len(unconverted) != 0 {
		t.Fatalf("list %+v unconverted %v", list, unconverted)
	}
	cur := list[0].Current
	if cur.Spent != 60_00 || cur.Remaining != 240_00 || cur.DaysLeft != 10 || cur.DailyAllowance != 24_00 || cur.State != ledger.BudgetActive {
		t.Fatalf("current %+v", cur)
	}

	d, err := svc.BudgetDetailFor(ctx, a, b.ID, today, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(d.History) != 3 || d.History[0].Spent != 60_00 || d.History[1].Spent != 30_00 || d.History[2].Spent != 0 {
		t.Fatalf("history %+v", d.History)
	}
	if len(d.Transactions) != 2 || len(d.Categories) != 2 || d.Categories[0].CategoryID != food.ID || len(d.Wallets) != 2 {
		t.Fatalf("detail txs=%d cats=%+v wallets=%+v", len(d.Transactions), d.Categories, d.Wallets)
	}

	sept := on("2026-09-10")
	d, err = svc.BudgetDetailFor(ctx, a, b.ID, today, &sept)
	if err != nil {
		t.Fatal(err)
	}
	if d.Progress.Spent != 30_00 || d.Progress.State != ledger.BudgetEnded || len(d.Transactions) != 1 {
		t.Fatalf("september %+v", d.Progress)
	}

	// Merging a counted category keeps the budget counting the target.
	if _, _, err := svc.MergeCategories(ctx, a, []uuid.UUID{groceries.ID}, shopping.ID); err != nil {
		t.Fatal(err)
	}
	list, _, err = svc.ListBudgets(ctx, a, today)
	if err != nil {
		t.Fatal(err)
	}
	// Food 40 + UYU groceries (now shopping) 20 + USD shopping 99.
	if list[0].Current.Spent != 159_00 {
		t.Fatalf("after merge spent = %d", list[0].Current.Spent)
	}

	// Empty wallets and categories count every expense in every wallet.
	b, err = svc.UpdateBudget(ctx, a, b.ID, ledger.BudgetInput{
		Name: "Everything", Amount: 1000_00, Currency: "USD", Repeat: ledger.RepeatMonthly, Start: on("2026-08-01"),
	})
	if err != nil {
		t.Fatal(err)
	}
	list, _, _ = svc.ListBudgets(ctx, a, today)
	if list[0].Current.Spent != 40_00+20_00+99_00+99_00 || len(b.WalletIDs) != 0 {
		t.Fatalf("all spent = %d", list[0].Current.Spent)
	}

	if err := svc.DeleteBudget(ctx, a, b.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.BudgetDetailFor(ctx, a, b.ID, today, nil); !errors.Is(err, ledger.ErrNotFound) {
		t.Fatalf("after delete: %v", err)
	}
}

func TestBudgetValidationAndIsolation(t *testing.T) {
	svc := ledger.New(testdb.Pool(t))
	ctx := context.Background()
	a := newActor(t, svc)
	b := newActor(t, svc)
	on := func(s string) time.Time { v, _ := time.Parse(time.DateOnly, s); return v }
	salary := categoryNamed(t, svc, a, ledger.CategoryIncome, "Salary")
	theirs := mustWallet(t, svc, b, "Theirs", "USD", 0)
	end := on("2026-09-01")

	base := ledger.BudgetInput{Name: "Trip", Amount: 100_00, Currency: "USD", Repeat: ledger.RepeatMonthly, Start: on("2026-10-01")}
	for name, mut := range map[string]func(*ledger.BudgetInput){
		"amount":         func(in *ledger.BudgetInput) { in.Amount = 0 },
		"currency":       func(in *ledger.BudgetInput) { in.Currency = "XXX" },
		"repeat":         func(in *ledger.BudgetInput) { in.Repeat = "" },
		"once needs end": func(in *ledger.BudgetInput) { in.Repeat = ledger.RepeatOnce },
		"end before":     func(in *ledger.BudgetInput) { in.Repeat, in.End = ledger.RepeatOnce, &end },
		"income":         func(in *ledger.BudgetInput) { in.CategoryIDs = []uuid.UUID{salary.ID} },
		"other wallet":   func(in *ledger.BudgetInput) { in.WalletIDs = []uuid.UUID{theirs.ID} },
	} {
		in := base
		mut(&in)
		var inv *ledger.InvalidError
		if _, err := svc.CreateBudget(ctx, a, in); !errors.As(err, &inv) {
			t.Errorf("%s: err = %v, want InvalidError", name, err)
		}
	}

	mine, err := svc.CreateBudget(ctx, a, base)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.BudgetDetailFor(ctx, b, mine.ID, on("2026-10-02"), nil); !errors.Is(err, ledger.ErrNotFound) {
		t.Fatalf("other household read: %v", err)
	}
	if _, err := svc.UpdateBudget(ctx, b, mine.ID, base); !errors.Is(err, ledger.ErrNotFound) {
		t.Fatalf("other household update: %v", err)
	}
	if err := svc.DeleteBudget(ctx, b, mine.ID); !errors.Is(err, ledger.ErrNotFound) {
		t.Fatalf("other household delete: %v", err)
	}
	if list, _, _ := svc.ListBudgets(ctx, b, on("2026-10-02")); len(list) != 0 {
		t.Fatalf("other household sees %d budgets", len(list))
	}
}
