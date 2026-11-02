package ledger

import (
	"strings"
	"testing"
	"time"
)

func dates(ds []time.Time) string {
	out := make([]string, len(ds))
	for i, d := range ds {
		out[i] = d.Format(time.DateOnly)
	}
	return strings.Join(out, " ")
}

func TestScheduleOccurrences(t *testing.T) {
	for _, c := range []struct {
		name    string
		freq    Frequency
		start   string
		end     string
		through string
		want    string
	}{
		{"daily", FreqDaily, "2026-10-30", "", "2026-11-02", "2026-10-30 2026-10-31 2026-11-01 2026-11-02"},
		{"every 2 days", FreqEvery2Days, "2026-10-30", "", "2026-11-05", "2026-10-30 2026-11-01 2026-11-03 2026-11-05"},
		// Friday start: Monday comes next.
		{"weekdays", FreqWeekdays, "2026-10-30", "", "2026-11-04", "2026-10-30 2026-11-02 2026-11-03 2026-11-04"},
		// A Saturday start still counts once, then weekdays.
		{"weekdays from a weekend", FreqWeekdays, "2026-10-31", "", "2026-11-03", "2026-10-31 2026-11-02 2026-11-03"},
		{"weekly", FreqWeekly, "2026-10-30", "", "2026-11-20", "2026-10-30 2026-11-06 2026-11-13 2026-11-20"},
		{"every 2 weeks", FreqEvery2Weeks, "2026-10-30", "", "2026-12-01", "2026-10-30 2026-11-13 2026-11-27"},
		{"every 4 weeks", FreqEvery4Weeks, "2026-10-30", "", "2026-12-31", "2026-10-30 2026-11-27 2026-12-25"},
		// The 31st clamps to shorter months and comes back.
		{"monthly from the 31st", FreqMonthly, "2026-01-31", "", "2026-05-31", "2026-01-31 2026-02-28 2026-03-31 2026-04-30 2026-05-31"},
		{"monthly in a leap year", FreqMonthly, "2028-01-30", "", "2028-03-30", "2028-01-30 2028-02-29 2028-03-30"},
		{"every 2 months", FreqEvery2Months, "2026-08-31", "", "2027-03-01", "2026-08-31 2026-10-31 2026-12-31 2027-02-28"},
		{"every 3 months", FreqEvery3Months, "2026-11-30", "", "2027-09-01", "2026-11-30 2027-02-28 2027-05-30 2027-08-30"},
		{"every 6 months", FreqEvery6Months, "2026-08-31", "", "2028-03-01", "2026-08-31 2027-02-28 2027-08-31 2028-02-29"},
		{"yearly on Feb 29", FreqYearly, "2028-02-29", "", "2033-01-01", "2028-02-29 2029-02-28 2030-02-28 2031-02-28 2032-02-29"},
		{"ends on a date", FreqWeekly, "2026-10-30", "2026-11-19", "2026-12-31", "2026-10-30 2026-11-06 2026-11-13"},
		{"end on an occurrence includes it", FreqWeekly, "2026-10-30", "2026-11-13", "2026-12-31", "2026-10-30 2026-11-06 2026-11-13"},
	} {
		t.Run(c.name, func(t *testing.T) {
			s := schedule{freq: c.freq, start: day(c.start)}
			if c.end != "" {
				e := day(c.end)
				s.end = &e
			}
			got, more := s.between(s.start.AddDate(0, 0, -1), day(c.through), 0)
			if dates(got) != c.want || more {
				t.Fatalf("got %s (more=%v), want %s", dates(got), more, c.want)
			}
		})
	}
}

func TestScheduleNextAfterMidSeries(t *testing.T) {
	for _, c := range []struct {
		freq        Frequency
		start, from string
		want        string
	}{
		{FreqDaily, "2026-01-01", "2026-03-15", "2026-03-16"},
		{FreqEvery2Days, "2026-01-01", "2026-01-04", "2026-01-05"},
		{FreqEvery2Days, "2026-01-01", "2026-01-05", "2026-01-07"},
		{FreqWeekly, "2026-01-01", "2026-01-08", "2026-01-15"},
		{FreqWeekdays, "2026-01-01", "2026-01-09", "2026-01-12"},
		{FreqMonthly, "2026-01-31", "2026-02-28", "2026-03-31"},
		{FreqMonthly, "2026-01-31", "2026-03-01", "2026-03-31"},
		{FreqEvery3Months, "2026-01-15", "2026-04-14", "2026-04-15"},
		{FreqYearly, "2026-06-01", "2030-05-31", "2030-06-01"},
		// Before the start, the start comes next.
		{FreqMonthly, "2026-06-01", "2026-01-01", "2026-06-01"},
	} {
		s := schedule{freq: c.freq, start: day(c.start)}
		got, ok := s.nextAfter(day(c.from))
		if !ok || got.Format(time.DateOnly) != c.want {
			t.Errorf("%s from %s: nextAfter(%s) = %s, %v; want %s", c.freq, c.start, c.from, got.Format(time.DateOnly), ok, c.want)
		}
	}
}

func TestScheduleNthAndLimit(t *testing.T) {
	s := schedule{freq: FreqMonthly, start: day("2026-01-31")}
	if got := s.nth(1).Format(time.DateOnly); got != "2026-01-31" {
		t.Fatalf("nth(1) = %s", got)
	}
	if got := s.nth(3).Format(time.DateOnly); got != "2026-03-31" {
		t.Fatalf("nth(3) = %s", got)
	}
	got, more := schedule{freq: FreqDaily, start: day("2026-01-01")}.between(day("2025-12-31"), day("2026-12-31"), 10)
	if len(got) != 10 || !more {
		t.Fatalf("limit: got %d, more=%v", len(got), more)
	}
}

func TestScheduleHorizon(t *testing.T) {
	today := day("2026-10-30")
	// Today plus three months (clamped like any month step).
	if got := (schedule{freq: FreqWeekly, start: day("2026-10-01")}).horizon(today); got.Format(time.DateOnly) != "2027-01-30" {
		t.Fatalf("weekly horizon = %s", got.Format(time.DateOnly))
	}
	// A yearly rule always reaches its next occurrence.
	if got := (schedule{freq: FreqYearly, start: day("2026-06-15")}).horizon(today); got.Format(time.DateOnly) != "2027-06-15" {
		t.Fatalf("yearly horizon = %s", got.Format(time.DateOnly))
	}
	// An ended rule falls back to the plain horizon.
	end := day("2026-06-15")
	if got := (schedule{freq: FreqYearly, start: day("2026-06-15"), end: &end}).horizon(today); got.Format(time.DateOnly) != "2027-01-30" {
		t.Fatalf("ended horizon = %s", got.Format(time.DateOnly))
	}
}
