package ledger_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/franco-roura/ledger-01/backend/internal/ledger"
	"github.com/franco-roura/ledger-01/backend/internal/testdb"
	"github.com/google/uuid"
)

var day = time.Date(2026, 10, 15, 0, 0, 0, 0, time.UTC)

func october() ledger.Filter {
	return ledger.Filter{From: time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC), To: time.Date(2026, 10, 31, 0, 0, 0, 0, time.UTC)}
}

func newActor(t *testing.T, svc *ledger.Service) ledger.Actor {
	t.Helper()
	email := fmt.Sprintf("%s@example.com", uuid.NewString())
	a, err := svc.EnsureUser(context.Background(), ledger.Profile{Subject: "test:" + email, Email: email, Name: "Test User"})
	if err != nil {
		t.Fatal(err)
	}
	return a
}

func categoryNamed(t *testing.T, svc *ledger.Service, a ledger.Actor, kind ledger.CategoryKind, name string) ledger.Category {
	t.Helper()
	cs, err := svc.ListCategories(context.Background(), a, false)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range cs {
		if c.Kind == kind && c.Name == name {
			return c
		}
	}
	t.Fatalf("no %s category %q", kind, name)
	return ledger.Category{}
}

func mustWallet(t *testing.T, svc *ledger.Service, a ledger.Actor, name, currency string, initial int64) ledger.Wallet {
	t.Helper()
	w, err := svc.CreateWallet(context.Background(), a, name, currency, initial)
	if err != nil {
		t.Fatal(err)
	}
	return w
}

func TestEnsureUserSeedsHousehold(t *testing.T) {
	svc := ledger.New(testdb.Pool(t))
	ctx := context.Background()
	a := newActor(t, svc)

	cs, err := svc.ListCategories(ctx, a, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(cs) < 20 {
		t.Fatalf("expected default categories, got %d", len(cs))
	}
	u, h, err := svc.Me(ctx, a)
	if err != nil {
		t.Fatal(err)
	}
	if h.Name != "Test's household" || h.BaseCurrency != "USD" || h.Role != ledger.RoleOwner {
		t.Fatalf("unexpected profile: %+v %+v", u, h)
	}

	// Signing in again returns the same household.
	again, err := svc.EnsureUser(ctx, ledger.Profile{Subject: "test:" + u.Email, Email: u.Email, Name: "Renamed"})
	if err != nil || again != a {
		t.Fatalf("EnsureUser again = %+v, %v; want %+v", again, err, a)
	}
}

// Two first sign-ins racing (two devices, two tabs) must both succeed with the same identity.
func TestEnsureUserConcurrentFirstSignIn(t *testing.T) {
	svc := ledger.New(testdb.Pool(t))
	email := fmt.Sprintf("race-%s@example.com", uuid.NewString())
	p := ledger.Profile{Subject: "test:" + email, Email: email, Name: "Racer"}

	const n = 8
	actors := make(chan ledger.Actor, n)
	errs := make(chan error, n)
	var start sync.WaitGroup
	start.Add(1)
	for range n {
		go func() {
			start.Wait()
			a, err := svc.EnsureUser(context.Background(), p)
			actors <- a
			errs <- err
		}()
	}
	start.Done()
	var first ledger.Actor
	for i := range n {
		if err := <-errs; err != nil {
			t.Fatalf("EnsureUser #%d: %v", i, err)
		}
		a := <-actors
		if i == 0 {
			first = a
		} else if a != first {
			t.Fatalf("got different identities: %+v vs %+v", a, first)
		}
	}
}

func TestBalancesAndSummary(t *testing.T) {
	svc := ledger.New(testdb.Pool(t))
	ctx := context.Background()
	a := newActor(t, svc)
	w := mustWallet(t, svc, a, "Northbank", "USD", 100_00)
	food := categoryNamed(t, svc, a, ledger.CategoryExpense, "Food & Drink")
	salary := categoryNamed(t, svc, a, ledger.CategoryIncome, "Salary")

	for _, in := range []ledger.TransactionInput{
		{WalletID: w.ID, CategoryID: food.ID, Amount: 12_50, OccurredOn: day, Note: "Lunch"},
		{WalletID: w.ID, CategoryID: food.ID, Amount: 7_25, OccurredOn: day, Note: "Coffee beans"},
		{WalletID: w.ID, CategoryID: salary.ID, Amount: 1000_00, OccurredOn: day},
	} {
		if _, err := svc.CreateTransaction(ctx, a, in); err != nil {
			t.Fatal(err)
		}
	}

	got, err := svc.GetWallet(ctx, a, w.ID)
	if err != nil {
		t.Fatal(err)
	}
	if want := int64(100_00 - 12_50 - 7_25 + 1000_00); got.Balance != want {
		t.Fatalf("balance = %d, want %d", got.Balance, want)
	}

	f := october()
	f.WalletID = w.ID
	s, err := svc.PeriodSummary(ctx, a, f)
	if err != nil {
		t.Fatal(err)
	}
	if s.Expenses != 19_75 || s.Income != 1000_00 || s.Change != 980_25 || s.TransactionCount != 3 {
		t.Fatalf("summary = %+v", s)
	}

	f.NoteQuery = "coffee"
	list, _, err := svc.ListTransactions(ctx, a, f, 10, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].Note != "Coffee beans" {
		t.Fatalf("note filter returned %+v", list)
	}
}

func TestPagination(t *testing.T) {
	svc := ledger.New(testdb.Pool(t))
	ctx := context.Background()
	a := newActor(t, svc)
	w := mustWallet(t, svc, a, "Cash", "USD", 0)
	food := categoryNamed(t, svc, a, ledger.CategoryExpense, "Food & Drink")
	for i := range 7 {
		if _, err := svc.CreateTransaction(ctx, a, ledger.TransactionInput{
			WalletID: w.ID, CategoryID: food.ID, Amount: int64(i + 1), OccurredOn: day.AddDate(0, 0, -i),
		}); err != nil {
			t.Fatal(err)
		}
	}
	f := october()
	f.WalletID = w.ID
	seen := map[uuid.UUID]bool{}
	token := ""
	for page := 0; ; page++ {
		ts, next, err := svc.ListTransactions(ctx, a, f, 3, token)
		if err != nil {
			t.Fatal(err)
		}
		for _, tx := range ts {
			if seen[tx.ID] {
				t.Fatalf("duplicate %s on page %d", tx.ID, page)
			}
			seen[tx.ID] = true
		}
		if next == "" {
			break
		}
		token = next
	}
	if len(seen) != 7 {
		t.Fatalf("saw %d transactions, want 7", len(seen))
	}
}

func TestCrossCurrencyTransfer(t *testing.T) {
	svc := ledger.New(testdb.Pool(t))
	ctx := context.Background()
	a := newActor(t, svc)
	usd := mustWallet(t, svc, a, "Northbank", "USD", 500_00)
	uyu := mustWallet(t, svc, a, "Northbank pesos", "UYU", 0)

	tr, err := svc.CreateTransfer(ctx, a, ledger.TransferInput{
		FromWalletID: usd.ID, ToWalletID: uyu.ID, FromAmount: 120_00, ToAmount: 4700_00,
		Rate: "39.1666666667", OccurredOn: day, Note: "pesos",
	})
	if err != nil {
		t.Fatal(err)
	}
	if tr.Rate != "39.1666666667" {
		t.Fatalf("rate = %s", tr.Rate)
	}

	gotUSD, _ := svc.GetWallet(ctx, a, usd.ID)
	gotUYU, _ := svc.GetWallet(ctx, a, uyu.ID)
	if gotUSD.Balance != 500_00-120_00 || gotUYU.Balance != 4700_00 {
		t.Fatalf("balances usd=%d uyu=%d", gotUSD.Balance, gotUYU.Balance)
	}

	// Transfers count in the change but not in expenses/income.
	f := october()
	f.WalletID = usd.ID
	s, _ := svc.PeriodSummary(ctx, a, f)
	if s.Expenses != 0 || s.Change != -120_00 {
		t.Fatalf("summary = %+v", s)
	}

	// The outgoing leg sees its counterpart.
	list, _, _ := svc.ListTransactions(ctx, a, f, 10, "")
	if len(list) != 1 || list[0].Kind != ledger.KindTransferOut || list[0].Counterpart == nil ||
		list[0].Counterpart.Amount != 4700_00 || list[0].Counterpart.Currency != "UYU" || list[0].Counterpart.WalletName != "Northbank pesos" {
		t.Fatalf("leg = %+v counterpart = %+v", list[0], list[0].Counterpart)
	}

	// A rate that doesn't match the amounts is rejected.
	_, err = svc.CreateTransfer(ctx, a, ledger.TransferInput{
		FromWalletID: usd.ID, ToWalletID: uyu.ID, FromAmount: 100_00, ToAmount: 4000_00, Rate: "39.05", OccurredOn: day,
	})
	var inv *ledger.InvalidError
	if !errors.As(err, &inv) || inv.Field != "rate" {
		t.Fatalf("mismatched rate err = %v", err)
	}

	// Deleting one leg deletes both.
	n, err := svc.DeleteTransactions(ctx, a, []uuid.UUID{list[0].ID})
	if err != nil || n != 2 {
		t.Fatalf("delete = %d, %v; want 2", n, err)
	}
	gotUYU, _ = svc.GetWallet(ctx, a, uyu.ID)
	if gotUYU.Balance != 0 {
		t.Fatalf("uyu balance after delete = %d", gotUYU.Balance)
	}
}

func TestSameCurrencyTransferUpdate(t *testing.T) {
	svc := ledger.New(testdb.Pool(t))
	ctx := context.Background()
	a := newActor(t, svc)
	x := mustWallet(t, svc, a, "Harbor Brokerage", "USD", 1000_00)
	y := mustWallet(t, svc, a, "Summit Invest", "USD", 0)
	z := mustWallet(t, svc, a, "Cash", "USD", 0)

	if _, err := svc.CreateTransfer(ctx, a, ledger.TransferInput{
		FromWalletID: x.ID, ToWalletID: y.ID, FromAmount: 10_00, ToAmount: 10_01, OccurredOn: day,
	}); err == nil {
		t.Fatal("same-currency transfer with different amounts must fail")
	}
	tr, err := svc.CreateTransfer(ctx, a, ledger.TransferInput{
		FromWalletID: x.ID, ToWalletID: y.ID, FromAmount: 300_00, ToAmount: 300_00, OccurredOn: day,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.UpdateTransfer(ctx, a, tr.ID, ledger.TransferInput{
		FromWalletID: x.ID, ToWalletID: z.ID, FromAmount: 50_00, ToAmount: 50_00, OccurredOn: day,
	}); err != nil {
		t.Fatal(err)
	}
	gx, _ := svc.GetWallet(ctx, a, x.ID)
	gy, _ := svc.GetWallet(ctx, a, y.ID)
	gz, _ := svc.GetWallet(ctx, a, z.ID)
	if gx.Balance != 950_00 || gy.Balance != 0 || gz.Balance != 50_00 {
		t.Fatalf("balances x=%d y=%d z=%d", gx.Balance, gy.Balance, gz.Balance)
	}
}

func TestHouseholdIsolation(t *testing.T) {
	svc := ledger.New(testdb.Pool(t))
	ctx := context.Background()
	alice := newActor(t, svc)
	bob := newActor(t, svc)
	w := mustWallet(t, svc, alice, "Alice's", "USD", 0)

	if _, err := svc.GetWallet(ctx, bob, w.ID); !errors.Is(err, ledger.ErrNotFound) {
		t.Fatalf("bob read alice's wallet: %v", err)
	}
	food := categoryNamed(t, svc, bob, ledger.CategoryExpense, "Food & Drink")
	if _, err := svc.CreateTransaction(ctx, bob, ledger.TransactionInput{
		WalletID: w.ID, CategoryID: food.ID, Amount: 1, OccurredOn: day,
	}); !errors.Is(err, ledger.ErrNotFound) {
		t.Fatalf("bob wrote to alice's wallet: %v", err)
	}
	ws, _ := svc.ListWallets(ctx, bob, true, true, ledger.Today())
	if len(ws) != 0 {
		t.Fatalf("bob sees %d wallets", len(ws))
	}
}

func TestDeleteAndMergeCategories(t *testing.T) {
	svc := ledger.New(testdb.Pool(t))
	ctx := context.Background()
	a := newActor(t, svc)
	w := mustWallet(t, svc, a, "Cash", "USD", 0)
	food := categoryNamed(t, svc, a, ledger.CategoryExpense, "Food & Drink")
	groceries := categoryNamed(t, svc, a, ledger.CategoryExpense, "Groceries")
	salary := categoryNamed(t, svc, a, ledger.CategoryIncome, "Salary")
	if _, err := svc.CreateTransaction(ctx, a, ledger.TransactionInput{WalletID: w.ID, CategoryID: food.ID, Amount: 5_00, OccurredOn: day}); err != nil {
		t.Fatal(err)
	}

	// A used category needs a replacement of the same kind.
	if _, err := svc.DeleteCategory(ctx, a, food.ID, nil); err == nil {
		t.Fatal("deleting a used category without replacement must fail")
	}
	if _, err := svc.DeleteCategory(ctx, a, food.ID, &salary.ID); err == nil {
		t.Fatal("replacement of another kind must fail")
	}

	// Merge food into groceries.
	target, moved, err := svc.MergeCategories(ctx, a, []uuid.UUID{food.ID}, groceries.ID)
	if err != nil {
		t.Fatal(err)
	}
	if moved != 1 || target.TransactionCount != 1 {
		t.Fatalf("moved=%d target=%+v", moved, target)
	}
	if _, _, err := svc.MergeCategories(ctx, a, []uuid.UUID{salary.ID}, groceries.ID); err == nil {
		t.Fatal("merging across kinds must fail")
	}
}

func TestLabelsAndCategoryKindSwitch(t *testing.T) {
	svc := ledger.New(testdb.Pool(t))
	ctx := context.Background()
	a := newActor(t, svc)
	w := mustWallet(t, svc, a, "Cash", "USD", 0)
	food := categoryNamed(t, svc, a, ledger.CategoryExpense, "Food & Drink")
	refund := categoryNamed(t, svc, a, ledger.CategoryIncome, "Refund")
	l, err := svc.CreateLabel(ctx, a, "Trip", "cat-teal")
	if err != nil {
		t.Fatal(err)
	}
	tx, err := svc.CreateTransaction(ctx, a, ledger.TransactionInput{
		WalletID: w.ID, CategoryID: food.ID, Amount: 20_00, OccurredOn: day, LabelIDs: []uuid.UUID{l.ID, l.ID},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(tx.LabelIDs) != 1 {
		t.Fatalf("labels = %v", tx.LabelIDs)
	}
	// Moving to an income category turns the expense into income.
	up, err := svc.UpdateTransaction(ctx, a, tx.ID, ledger.TransactionInput{CategoryID: refund.ID, Amount: 20_00, OccurredOn: day})
	if err != nil {
		t.Fatal(err)
	}
	if up.Kind != ledger.KindIncome || len(up.LabelIDs) != 0 {
		t.Fatalf("updated = %+v", up)
	}
	got, _ := svc.GetWallet(ctx, a, w.ID)
	if got.Balance != 20_00 {
		t.Fatalf("balance = %d", got.Balance)
	}
}
