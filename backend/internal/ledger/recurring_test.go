package ledger_test

import (
	"context"
	"errors"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/franco-roura/ledger-01/backend/internal/ledger"
	"github.com/franco-roura/ledger-01/backend/internal/testdb"
	"github.com/google/uuid"
)

func on(s string) time.Time {
	d, err := time.Parse(time.DateOnly, s)
	if err != nil {
		panic(err)
	}
	return d
}

func ptr[T any](v T) *T { return &v }

// walletRows returns every row of a wallet as "date:amount" (the scheduled and the upcoming ones),
// oldest first, as seen on today.
func walletRows(t *testing.T, svc *ledger.Service, a ledger.Actor, walletID uuid.UUID, today time.Time) []string {
	t.Helper()
	ctx := context.Background()
	past, _, err := svc.ListTransactions(ctx, a, ledger.Filter{WalletID: walletID, From: on("2020-01-01"), To: on("2030-12-31"), Today: today}, 200, "")
	if err != nil {
		t.Fatal(err)
	}
	upcoming, err := svc.ListUpcoming(ctx, a, &walletID, nil, today)
	if err != nil {
		t.Fatal(err)
	}
	slices.Reverse(past)
	var out []string
	for _, tx := range append(past, upcoming...) {
		out = append(out, tx.OccurredOn.Format(time.DateOnly)+":"+itoa(tx.Amount))
	}
	return out
}

func itoa(v int64) string {
	b := []byte{}
	if v == 0 {
		return "0"
	}
	for ; v > 0; v /= 10 {
		b = append([]byte{byte('0' + v%10)}, b...)
	}
	return string(b)
}

func wantRows(t *testing.T, got []string, want string) {
	t.Helper()
	if strings.Join(got, " ") != want {
		t.Fatalf("rows:\n got  %s\n want %s", strings.Join(got, " "), want)
	}
}

func TestRecurringCreateBackfillAndTopUp(t *testing.T) {
	svc := ledger.New(testdb.Pool(t))
	ctx := context.Background()
	a := newActor(t, svc)
	w := mustWallet(t, svc, a, "Checking", "USD", 1000_00)
	rent := categoryNamed(t, svc, a, ledger.CategoryExpense, "Home")
	today := on("2026-10-15")

	// Monthly from Aug 31: the past occurrences are filled in, the future ones run three months
	// past today (Jan 15), and the 31st clamps to shorter months.
	tx, err := svc.CreateTransaction(ctx, a, ledger.TransactionInput{
		WalletID: w.ID, CategoryID: rent.ID, Amount: 500_00, OccurredOn: on("2026-08-31"), Note: "Rent",
		Repeat: ledger.Recurrence{Frequency: ledger.FreqMonthly}, Today: today,
	})
	if err != nil {
		t.Fatal(err)
	}
	if tx.RecurringRuleID == nil || tx.Recurrence == nil || tx.Recurrence.Frequency != ledger.FreqMonthly {
		t.Fatalf("created %+v", tx)
	}
	wantRows(t, walletRows(t, svc, a, w.ID, today),
		"2026-08-31:50000 2026-09-30:50000 2026-10-31:50000 2026-11-30:50000 2026-12-31:50000")

	// Upcoming rows don't move the balance yet.
	got, err := svc.GetWalletOn(ctx, a, w.ID, today)
	if err != nil {
		t.Fatal(err)
	}
	if got.Balance != 0 {
		t.Fatalf("balance on %s = %d, want 0", today.Format(time.DateOnly), got.Balance)
	}

	// The job tops up idempotently, however often it runs.
	later := on("2026-12-01")
	for range 2 {
		if _, err := svc.TopUpRecurring(ctx, later); err != nil {
			t.Fatal(err)
		}
	}
	wantRows(t, walletRows(t, svc, a, w.ID, later),
		"2026-08-31:50000 2026-09-30:50000 2026-10-31:50000 2026-11-30:50000 2026-12-31:50000 2027-01-31:50000 2027-02-28:50000")
}

func TestRecurringTopUpRunsConcurrently(t *testing.T) {
	svc := ledger.New(testdb.Pool(t))
	ctx := context.Background()
	a := newActor(t, svc)
	w := mustWallet(t, svc, a, "Cash", "USD", 0)
	food := categoryNamed(t, svc, a, ledger.CategoryExpense, "Groceries")
	if _, err := svc.CreateTransaction(ctx, a, ledger.TransactionInput{
		WalletID: w.ID, CategoryID: food.ID, Amount: 10_00, OccurredOn: on("2026-10-01"),
		Repeat: ledger.Recurrence{Frequency: ledger.FreqWeekly}, Today: on("2026-10-01"),
	}); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	errs := make(chan error, 4)
	for range 4 {
		wg.Go(func() {
			if _, err := svc.TopUpRecurring(ctx, on("2027-03-01")); err != nil {
				errs <- err
			}
		})
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatal(err)
	}
	// Oct 1 2026 to Jun 1 2027, weekly: 35 Thursdays, each once.
	rows := walletRows(t, svc, a, w.ID, on("2027-03-01"))
	if len(rows) != 35 || rows[len(rows)-1] != "2027-05-27:1000" {
		t.Fatalf("got %d rows, last %s", len(rows), rows[len(rows)-1])
	}
}

func TestRecurringEndsAndValidation(t *testing.T) {
	svc := ledger.New(testdb.Pool(t))
	ctx := context.Background()
	a := newActor(t, svc)
	w := mustWallet(t, svc, a, "Checking", "USD", 0)
	gym := categoryNamed(t, svc, a, ledger.CategoryExpense, "Sport & Hobbies")
	today := on("2026-10-15")

	// "After 3 times" is stored as the date of the third.
	tx, err := svc.CreateTransaction(ctx, a, ledger.TransactionInput{
		WalletID: w.ID, CategoryID: gym.ID, Amount: 30_00, OccurredOn: on("2026-10-20"),
		Repeat: ledger.Recurrence{Frequency: ledger.FreqEvery2Weeks, Count: 3}, Today: today,
	})
	if err != nil {
		t.Fatal(err)
	}
	if tx.Recurrence.End == nil || tx.Recurrence.End.Format(time.DateOnly) != "2026-11-17" {
		t.Fatalf("recurrence %+v", tx.Recurrence)
	}
	wantRows(t, walletRows(t, svc, a, w.ID, today), "2026-10-20:3000 2026-11-03:3000 2026-11-17:3000")
	if _, err := svc.TopUpRecurring(ctx, on("2027-06-01")); err != nil {
		t.Fatal(err)
	}
	wantRows(t, walletRows(t, svc, a, w.ID, today), "2026-10-20:3000 2026-11-03:3000 2026-11-17:3000")

	var inv *ledger.InvalidError
	for name, rep := range map[string]ledger.Recurrence{
		"unknown frequency": {Frequency: "hourly"},
		"end before start":  {Frequency: ledger.FreqWeekly, End: ptr(on("2026-01-01"))},
		"count and date":    {Frequency: ledger.FreqWeekly, Count: 2, End: ptr(on("2027-01-01"))},
		"count too large":   {Frequency: ledger.FreqWeekly, Count: 5000},
	} {
		_, err := svc.CreateTransaction(ctx, a, ledger.TransactionInput{
			WalletID: w.ID, CategoryID: gym.ID, Amount: 1_00, OccurredOn: on("2026-10-01"), Repeat: rep, Today: today,
		})
		if !errors.As(err, &inv) {
			t.Errorf("%s: err = %v, want InvalidError", name, err)
		}
	}
	// A daily rule starting years back would create too many rows in one request.
	if _, err := svc.CreateTransaction(ctx, a, ledger.TransactionInput{
		WalletID: w.ID, CategoryID: gym.ID, Amount: 1_00, OccurredOn: on("2022-01-01"),
		Repeat: ledger.Recurrence{Frequency: ledger.FreqDaily}, Today: today,
	}); !errors.As(err, &inv) || inv.Field != "repeat" {
		t.Fatalf("too many occurrences: err = %v", err)
	}
	// Rejected requests leave nothing behind.
	wantRows(t, walletRows(t, svc, a, w.ID, today), "2026-10-20:3000 2026-11-03:3000 2026-11-17:3000")
}

// Monthly on the 5th with today Oct 15: Sep 5 and Oct 5 are past, Nov 5 to Jan 5 upcoming.
func monthlySeries(t *testing.T, svc *ledger.Service, a ledger.Actor, walletID, categoryID uuid.UUID, today time.Time) {
	t.Helper()
	if _, err := svc.CreateTransaction(context.Background(), a, ledger.TransactionInput{
		WalletID: walletID, CategoryID: categoryID, Amount: 40_00, OccurredOn: on("2026-09-05"), Note: "Phone",
		Repeat: ledger.Recurrence{Frequency: ledger.FreqMonthly}, Today: today,
	}); err != nil {
		t.Fatal(err)
	}
}

func occurrenceOn(t *testing.T, svc *ledger.Service, a ledger.Actor, walletID uuid.UUID, date string, today time.Time) ledger.Transaction {
	t.Helper()
	ctx := context.Background()
	past, _, err := svc.ListTransactions(ctx, a, ledger.Filter{WalletID: walletID, From: on("2020-01-01"), To: on("2030-12-31"), Today: today}, 200, "")
	if err != nil {
		t.Fatal(err)
	}
	upcoming, err := svc.ListUpcoming(ctx, a, &walletID, nil, today)
	if err != nil {
		t.Fatal(err)
	}
	for _, tx := range append(past, upcoming...) {
		if tx.OccurredOn.Format(time.DateOnly) == date {
			return tx
		}
	}
	t.Fatalf("no occurrence on %s", date)
	return ledger.Transaction{}
}

func TestRecurringSeriesEdits(t *testing.T) {
	svc := ledger.New(testdb.Pool(t))
	ctx := context.Background()
	a := newActor(t, svc)
	w := mustWallet(t, svc, a, "Checking", "USD", 0)
	phone := categoryNamed(t, svc, a, ledger.CategoryExpense, "Bills & Fees")
	today := on("2026-10-15")
	monthlySeries(t, svc, a, w.ID, phone.ID, today)
	wantRows(t, walletRows(t, svc, a, w.ID, today),
		"2026-09-05:4000 2026-10-05:4000 2026-11-05:4000 2026-12-05:4000 2027-01-05:4000")

	edit := func(date string, amount int64, scope ledger.Scope, newDate string, rep ledger.Recurrence) ledger.Transaction {
		t.Helper()
		cur := occurrenceOn(t, svc, a, w.ID, date, today)
		if newDate == "" {
			newDate = date
		}
		got, err := svc.UpdateTransaction(ctx, a, cur.ID, ledger.TransactionInput{
			CategoryID: phone.ID, Amount: amount, OccurredOn: on(newDate), Note: cur.Note, Scope: scope, Repeat: rep, Today: today,
		})
		if err != nil {
			t.Fatal(err)
		}
		return got
	}
	monthly := ledger.Recurrence{Frequency: ledger.FreqMonthly}

	// Only this one: the rest of the series stays, and the row keeps its rule.
	got := edit("2026-12-05", 99_00, ledger.ScopeOnlyThis, "", ledger.Recurrence{})
	if got.RecurringRuleID == nil {
		t.Fatal("an edited occurrence keeps its rule")
	}
	wantRows(t, walletRows(t, svc, a, w.ID, today),
		"2026-09-05:4000 2026-10-05:4000 2026-11-05:4000 2026-12-05:9900 2027-01-05:4000")

	// This and future from Nov 5: Nov 5 and every later upcoming one (the one-off edit too).
	edit("2026-11-05", 45_00, ledger.ScopeThisAndFuture, "", monthly)
	wantRows(t, walletRows(t, svc, a, w.ID, today),
		"2026-09-05:4000 2026-10-05:4000 2026-11-05:4500 2026-12-05:4500 2027-01-05:4500")

	// This and future from a past occurrence: it changes, the past ones after it don't, the
	// upcoming ones do.
	edit("2026-09-05", 42_00, ledger.ScopeThisAndFuture, "", monthly)
	wantRows(t, walletRows(t, svc, a, w.ID, today),
		"2026-09-05:4200 2026-10-05:4000 2026-11-05:4200 2026-12-05:4200 2027-01-05:4200")

	// Moving the date with a new schedule re-anchors the series from there.
	edit("2026-12-05", 42_00, ledger.ScopeThisAndFuture, "2026-12-10", ledger.Recurrence{Frequency: ledger.FreqEvery2Weeks})
	wantRows(t, walletRows(t, svc, a, w.ID, today),
		"2026-09-05:4200 2026-10-05:4000 2026-11-05:4200 2026-12-10:4200 2026-12-24:4200 2027-01-07:4200")

	// The job continues the new schedule (through Mar 1, three months after Dec 1).
	if _, err := svc.TopUpRecurring(ctx, on("2026-12-01")); err != nil {
		t.Fatal(err)
	}
	wantRows(t, walletRows(t, svc, a, w.ID, today),
		"2026-09-05:4200 2026-10-05:4000 2026-11-05:4200 2026-12-10:4200 2026-12-24:4200 2027-01-07:4200 2027-01-21:4200 2027-02-04:4200 2027-02-18:4200")

	// "Never" on a series edit ends it at that occurrence.
	edit("2027-01-07", 42_00, ledger.ScopeThisAndFuture, "", ledger.Recurrence{})
	wantRows(t, walletRows(t, svc, a, w.ID, today),
		"2026-09-05:4200 2026-10-05:4000 2026-11-05:4200 2026-12-10:4200 2026-12-24:4200 2027-01-07:4200")
	if _, err := svc.TopUpRecurring(ctx, on("2027-03-01")); err != nil {
		t.Fatal(err)
	}
	wantRows(t, walletRows(t, svc, a, w.ID, today),
		"2026-09-05:4200 2026-10-05:4000 2026-11-05:4200 2026-12-10:4200 2026-12-24:4200 2027-01-07:4200")
}

func TestRecurringDeletes(t *testing.T) {
	svc := ledger.New(testdb.Pool(t))
	ctx := context.Background()
	a := newActor(t, svc)
	w := mustWallet(t, svc, a, "Checking", "USD", 0)
	phone := categoryNamed(t, svc, a, ledger.CategoryExpense, "Bills & Fees")
	today := on("2026-10-15")
	monthlySeries(t, svc, a, w.ID, phone.ID, today)

	// Only this one: it isn't created again by the job.
	nov := occurrenceOn(t, svc, a, w.ID, "2026-11-05", today)
	if n, err := svc.DeleteOccurrences(ctx, a, []uuid.UUID{nov.ID}, ledger.ScopeOnlyThis, today); err != nil || n != 1 {
		t.Fatalf("delete only this = %d, %v", n, err)
	}
	if _, err := svc.TopUpRecurring(ctx, today); err != nil {
		t.Fatal(err)
	}
	wantRows(t, walletRows(t, svc, a, w.ID, today), "2026-09-05:4000 2026-10-05:4000 2026-12-05:4000 2027-01-05:4000")

	// This and future from Dec 5: Dec and Jan go, and nothing comes back.
	dec := occurrenceOn(t, svc, a, w.ID, "2026-12-05", today)
	if n, err := svc.DeleteOccurrences(ctx, a, []uuid.UUID{dec.ID}, ledger.ScopeThisAndFuture, today); err != nil || n != 2 {
		t.Fatalf("delete this and future = %d, %v", n, err)
	}
	if _, err := svc.TopUpRecurring(ctx, on("2027-06-01")); err != nil {
		t.Fatal(err)
	}
	wantRows(t, walletRows(t, svc, a, w.ID, today), "2026-09-05:4000 2026-10-05:4000")

	// This and future from the first occurrence of another series removes the rule; the past rows
	// after it stay.
	w2 := mustWallet(t, svc, a, "Savings", "USD", 0)
	monthlySeries(t, svc, a, w2.ID, phone.ID, today)
	first := occurrenceOn(t, svc, a, w2.ID, "2026-09-05", today)
	if n, err := svc.DeleteOccurrences(ctx, a, []uuid.UUID{first.ID}, ledger.ScopeThisAndFuture, today); err != nil || n != 4 {
		t.Fatalf("delete from first = %d, %v", n, err)
	}
	rest := occurrenceOn(t, svc, a, w2.ID, "2026-10-05", today)
	if rest.RecurringRuleID != nil {
		t.Fatalf("the rule should be gone, row still points at %v", rest.RecurringRuleID)
	}
	wantRows(t, walletRows(t, svc, a, w2.ID, today), "2026-10-05:4000")
}

func TestRecurringTransfersAndConversion(t *testing.T) {
	svc := ledger.New(testdb.Pool(t))
	ctx := context.Background()
	a := newActor(t, svc)
	usd := mustWallet(t, svc, a, "Checking", "USD", 1000_00)
	uyu := mustWallet(t, svc, a, "Pesos", "UYU", 0)
	today := on("2026-10-15")

	// Every 4 weeks, 100 USD -> 4,000 UYU, from Oct 1.
	if _, err := svc.CreateTransfer(ctx, a, ledger.TransferInput{
		FromWalletID: usd.ID, ToWalletID: uyu.ID, FromAmount: 100_00, ToAmount: 4000_00, Rate: "40", OccurredOn: on("2026-10-01"),
		Note: "Allowance", Repeat: ledger.Recurrence{Frequency: ledger.FreqEvery4Weeks}, Today: today,
	}); err != nil {
		t.Fatal(err)
	}
	wantRows(t, walletRows(t, svc, a, usd.ID, today), "2026-10-01:10000 2026-10-29:10000 2026-11-26:10000 2026-12-24:10000")
	wantRows(t, walletRows(t, svc, a, uyu.ID, today), "2026-10-01:400000 2026-10-29:400000 2026-11-26:400000 2026-12-24:400000")

	// Household-wide, each upcoming transfer is listed once.
	all, err := svc.ListUpcoming(ctx, a, nil, nil, today)
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 3 || all[0].Kind != ledger.KindTransferOut || all[0].Counterpart == nil || all[0].RecurringRuleID == nil {
		t.Fatalf("household upcoming = %+v", all)
	}

	// Edit this and future from the second occurrence: both legs of every upcoming one change.
	if _, err := svc.UpdateTransfer(ctx, a, all[0].Counterpart.TransferID, ledger.TransferInput{
		FromWalletID: usd.ID, ToWalletID: uyu.ID, FromAmount: 50_00, ToAmount: 2100_00, Rate: "42", OccurredOn: on("2026-10-29"),
		Note: "Allowance", Scope: ledger.ScopeThisAndFuture, Repeat: ledger.Recurrence{Frequency: ledger.FreqEvery4Weeks}, Today: today,
	}); err != nil {
		t.Fatal(err)
	}
	wantRows(t, walletRows(t, svc, a, usd.ID, today), "2026-10-01:10000 2026-10-29:5000 2026-11-26:5000 2026-12-24:5000")
	wantRows(t, walletRows(t, svc, a, uyu.ID, today), "2026-10-01:400000 2026-10-29:210000 2026-11-26:210000 2026-12-24:210000")

	// Upcoming transfers aren't rates yet: an Oct 20 UYU expense converts at the Oct 1 rate (40),
	// not the Oct 29 one (42).
	fun := categoryNamed(t, svc, a, ledger.CategoryExpense, "Entertainment")
	if _, err := svc.CreateTransaction(ctx, a, ledger.TransactionInput{WalletID: uyu.ID, CategoryID: fun.ID, Amount: 400_00, OccurredOn: on("2026-10-01")}); err != nil {
		t.Fatal(err)
	}
	d, err := svc.Dashboard(ctx, a, ledger.ReportFilter{From: on("2026-10-01"), To: on("2026-12-31"), Today: today}, ledger.BucketMonth)
	if err != nil {
		t.Fatal(err)
	}
	// Balance on Oct 15: 900 USD + 3,600 UYU at 40 = 990 USD. Only the Oct 1 rows count.
	if d.Balance != 990_00 || d.Expenses != 10_00 || d.TransfersOutCount != 1 || d.TransfersInCount != 1 {
		t.Fatalf("dashboard = balance %d, expenses %d, transfers %d/%d", d.Balance, d.Expenses, d.TransfersOutCount, d.TransfersInCount)
	}
}

func TestUpcomingRowsDontCount(t *testing.T) {
	svc := ledger.New(testdb.Pool(t))
	ctx := context.Background()
	a := newActor(t, svc)
	w := mustWallet(t, svc, a, "Checking", "USD", 1000_00)
	food := categoryNamed(t, svc, a, ledger.CategoryExpense, "Groceries")
	today := on("2026-10-15")

	// Weekly 20.00 from Oct 1: Oct 1 and Oct 8 are due, Oct 22 and later aren't.
	if _, err := svc.CreateTransaction(ctx, a, ledger.TransactionInput{
		WalletID: w.ID, CategoryID: food.ID, Amount: 20_00, OccurredOn: on("2026-10-01"),
		Repeat: ledger.Recurrence{Frequency: ledger.FreqWeekly}, Today: today,
	}); err != nil {
		t.Fatal(err)
	}
	// A one-off future-dated expense is upcoming too.
	if _, err := svc.CreateTransaction(ctx, a, ledger.TransactionInput{WalletID: w.ID, CategoryID: food.ID, Amount: 5_00, OccurredOn: on("2026-10-20")}); err != nil {
		t.Fatal(err)
	}

	f := ledger.Filter{WalletID: w.ID, From: on("2026-10-01"), To: on("2026-10-31"), Today: today}
	sum, err := svc.PeriodSummary(ctx, a, f)
	if err != nil {
		t.Fatal(err)
	}
	if sum.Balance != 940_00 || sum.Expenses != 60_00 || sum.TransactionCount != 3 {
		t.Fatalf("summary = %+v", sum)
	}
	list, _, err := svc.ListTransactions(ctx, a, f, 50, "")
	if err != nil || len(list) != 3 {
		t.Fatalf("list = %d rows, %v", len(list), err)
	}
	// A period entirely after today is empty.
	f.From, f.To = on("2026-11-01"), on("2026-11-30")
	if list, _, _ := svc.ListTransactions(ctx, a, f, 50, ""); len(list) != 0 {
		t.Fatalf("future period listed %d rows", len(list))
	}

	ws, err := svc.ListWallets(ctx, a, true, false, today)
	if err != nil || len(ws) != 1 || ws[0].Balance != 940_00 {
		t.Fatalf("wallets = %+v, %v", ws, err)
	}

	d, err := svc.Dashboard(ctx, a, ledger.ReportFilter{From: on("2026-10-01"), To: on("2026-10-31"), Today: today}, ledger.BucketWeek)
	if err != nil {
		t.Fatal(err)
	}
	if d.Balance != 940_00 || d.Expenses != 60_00 {
		t.Fatalf("dashboard balance %d expenses %d", d.Balance, d.Expenses)
	}
	cr, err := svc.CategoryReportFor(ctx, a, food.ID, ledger.ReportFilter{From: on("2026-10-01"), To: on("2026-10-31"), Today: today}, ledger.BucketWeek)
	if err != nil || cr.Total != 60_00 || len(cr.Transactions) != 3 {
		t.Fatalf("category report total %d, %d rows, %v", cr.Total, len(cr.Transactions), err)
	}

	b, err := svc.CreateBudget(ctx, a, ledger.BudgetInput{
		Name: "Food", Amount: 200_00, Currency: "USD", Repeat: ledger.RepeatMonthly, Start: on("2026-10-01"),
	})
	if err != nil {
		t.Fatal(err)
	}
	bd, err := svc.BudgetDetailFor(ctx, a, b.ID, today, nil)
	if err != nil {
		t.Fatal(err)
	}
	if bd.Progress.Spent != 60_00 || len(bd.Transactions) != 3 {
		t.Fatalf("budget spent %d, %d rows", bd.Progress.Spent, len(bd.Transactions))
	}
	// Once the date comes, the rows count.
	bd, err = svc.BudgetDetailFor(ctx, a, b.ID, on("2026-10-31"), nil)
	if err != nil || bd.Progress.Spent != 105_00 {
		t.Fatalf("budget spent on Oct 31 = %d, %v", bd.Progress.Spent, err)
	}
}

func TestMakeRecurringAndCategoryCleanup(t *testing.T) {
	svc := ledger.New(testdb.Pool(t))
	ctx := context.Background()
	a := newActor(t, svc)
	w := mustWallet(t, svc, a, "Checking", "USD", 0)
	salary := categoryNamed(t, svc, a, ledger.CategoryIncome, "Salary")
	today := on("2026-10-15")

	// An existing row becomes the first occurrence of a new series.
	tx, err := svc.CreateTransaction(ctx, a, ledger.TransactionInput{WalletID: w.ID, CategoryID: salary.ID, Amount: 2000_00, OccurredOn: on("2026-10-01")})
	if err != nil {
		t.Fatal(err)
	}
	got, err := svc.UpdateTransaction(ctx, a, tx.ID, ledger.TransactionInput{
		CategoryID: salary.ID, Amount: 2000_00, OccurredOn: on("2026-10-01"),
		Repeat: ledger.Recurrence{Frequency: ledger.FreqMonthly, End: ptr(on("2026-12-31"))}, Today: today,
	})
	if err != nil || got.RecurringRuleID == nil {
		t.Fatalf("make recurring = %+v, %v", got, err)
	}
	wantRows(t, walletRows(t, svc, a, w.ID, today), "2026-10-01:200000 2026-11-01:200000 2026-12-01:200000")

	// A category a rule uses can't vanish without a replacement; the rule follows the replacement.
	bonus, err := svc.CreateCategory(ctx, a, ledger.CategoryIncome, "Bonus", "gift", "cat-teal")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.DeleteCategory(ctx, a, salary.ID, nil); err == nil {
		t.Fatal("deleting a category in use needs a replacement")
	}
	if _, err := svc.DeleteCategory(ctx, a, salary.ID, &bonus.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.TopUpRecurring(ctx, on("2026-11-15")); err != nil {
		t.Fatal(err)
	}
	for _, row := range []string{"2026-10-01", "2026-12-01"} {
		if tx := occurrenceOn(t, svc, a, w.ID, row, today); tx.CategoryID == nil || *tx.CategoryID != bonus.ID {
			t.Fatalf("%s category = %v", row, tx.CategoryID)
		}
	}
}
