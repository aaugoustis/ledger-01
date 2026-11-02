package ledger

import (
	"testing"
	"time"
)

func day(s string) time.Time {
	t, err := time.Parse(time.DateOnly, s)
	if err != nil {
		panic(err)
	}
	return t
}

func TestAddMonthsClamped(t *testing.T) {
	for _, c := range []struct {
		from string
		n    int
		want string
	}{
		{"2026-01-31", 1, "2026-02-28"},
		{"2028-01-31", 1, "2028-02-29"},
		{"2026-01-31", 2, "2026-03-31"},
		{"2026-11-15", 3, "2027-02-15"},
		{"2026-03-31", -1, "2026-02-28"},
		{"2026-01-10", -1, "2025-12-10"},
		{"2024-02-29", 12, "2025-02-28"},
	} {
		if got := addMonthsClamped(day(c.from), c.n).Format(time.DateOnly); got != c.want {
			t.Errorf("addMonthsClamped(%s, %d) = %s, want %s", c.from, c.n, got, c.want)
		}
	}
}

func TestBudgetPeriods(t *testing.T) {
	end := day("2026-12-24")
	for _, c := range []struct {
		name     string
		b        Budget
		on       string
		from, to string
	}{
		{"monthly mid-month start", Budget{Repeat: RepeatMonthly, Start: day("2026-01-15")}, "2026-10-14", "2026-09-15", "2026-10-14"},
		{"monthly on the boundary", Budget{Repeat: RepeatMonthly, Start: day("2026-01-15")}, "2026-10-15", "2026-10-15", "2026-11-14"},
		{"monthly from the 31st", Budget{Repeat: RepeatMonthly, Start: day("2026-01-31")}, "2026-03-01", "2026-02-28", "2026-03-30"},
		{"monthly before start", Budget{Repeat: RepeatMonthly, Start: day("2026-11-01")}, "2026-10-29", "2026-11-01", "2026-11-30"},
		{"weekly", Budget{Repeat: RepeatWeekly, Start: day("2026-10-05")}, "2026-10-29", "2026-10-26", "2026-11-01"},
		{"biweekly", Budget{Repeat: RepeatBiweekly, Start: day("2026-10-05")}, "2026-10-29", "2026-10-19", "2026-11-01"},
		{"daily", Budget{Repeat: RepeatDaily, Start: day("2026-10-05")}, "2026-10-29", "2026-10-29", "2026-10-29"},
		{"yearly", Budget{Repeat: RepeatYearly, Start: day("2025-03-01")}, "2026-02-28", "2025-03-01", "2026-02-28"},
		{"once", Budget{Repeat: RepeatOnce, Start: day("2026-12-01"), End: &end}, "2027-01-10", "2026-12-01", "2026-12-24"},
	} {
		i := c.b.periodIndex(day(c.on))
		from, to := c.b.period(i)
		if from.Format(time.DateOnly) != c.from || to.Format(time.DateOnly) != c.to {
			t.Errorf("%s: period for %s = %s..%s, want %s..%s", c.name, c.on, from.Format(time.DateOnly), to.Format(time.DateOnly), c.from, c.to)
		}
	}
}

func TestBudgetProgress(t *testing.T) {
	b := Budget{Repeat: RepeatMonthly, Start: day("2026-10-01"), Amount: 300_00}
	p := b.progress(0, 100_00, day("2026-10-22")) // 10 days left, today included
	if p.State != BudgetActive || p.DaysTotal != 31 || p.DaysLeft != 10 || p.Remaining != 200_00 || p.DailyAllowance != 20_00 {
		t.Fatalf("active: %+v", p)
	}
	p = b.progress(0, 350_00, day("2026-10-22"))
	if p.Remaining != -50_00 || p.DailyAllowance != 0 {
		t.Fatalf("over: %+v", p)
	}
	p = b.progress(0, 0, day("2026-09-20"))
	if p.State != BudgetUpcoming || p.DaysLeft != 31 || p.DailyAllowance != 300_00/31 {
		t.Fatalf("upcoming: %+v", p)
	}
	p = b.progress(0, 10_00, day("2026-11-02"))
	if p.State != BudgetEnded || p.DaysLeft != 0 || p.DailyAllowance != 0 {
		t.Fatalf("ended: %+v", p)
	}
}
