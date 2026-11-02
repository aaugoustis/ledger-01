package ledger

import (
	"time"

	"github.com/franco-roura/ledger-01/backend/internal/db/dbq"
)

type Frequency = dbq.RecurringFrequency

const (
	FreqDaily        = dbq.RecurringFrequencyDaily
	FreqEvery2Days   = dbq.RecurringFrequencyEvery2Days
	FreqWeekdays     = dbq.RecurringFrequencyWeekdays
	FreqWeekly       = dbq.RecurringFrequencyWeekly
	FreqEvery2Weeks  = dbq.RecurringFrequencyEvery2Weeks
	FreqEvery4Weeks  = dbq.RecurringFrequencyEvery4Weeks
	FreqMonthly      = dbq.RecurringFrequencyMonthly
	FreqEvery2Months = dbq.RecurringFrequencyEvery2Months
	FreqEvery3Months = dbq.RecurringFrequencyEvery3Months
	FreqEvery6Months = dbq.RecurringFrequencyEvery6Months
	FreqYearly       = dbq.RecurringFrequencyYearly
)

// HorizonMonths is how far past today recurring rules pre-create their occurrences (always at
// least the next one).
const HorizonMonths = 3

// maxOccurrencesPerRequest caps how many occurrences one API request may create (a daily rule
// starting years back). The top-up job has no cap.
const maxOccurrencesPerRequest = 1000

// step returns the spacing of a frequency: a number of days, or a number of months.
func step(f Frequency) (days, months int) {
	switch f {
	case FreqDaily:
		return 1, 0
	case FreqEvery2Days:
		return 2, 0
	case FreqWeekly:
		return 7, 0
	case FreqEvery2Weeks:
		return 14, 0
	case FreqEvery4Weeks:
		return 28, 0
	case FreqMonthly:
		return 0, 1
	case FreqEvery2Months:
		return 0, 2
	case FreqEvery3Months:
		return 0, 3
	case FreqEvery6Months:
		return 0, 6
	case FreqYearly:
		return 0, 12
	}
	return 0, 0
}

func validFrequency(f Frequency) bool {
	if f == FreqWeekdays {
		return true
	}
	d, m := step(f)
	return d > 0 || m > 0
}

// schedule is a rule's dates: start is the first occurrence and end (if set) the last date one may
// fall on. Month-based schedules count from start and clamp to shorter months, so a rule on the
// 31st falls on Feb 28 and is back on Mar 31. Weekday schedules continue on the weekdays after
// start (start itself may be a weekend).
type schedule struct {
	freq  Frequency
	start time.Time
	end   *time.Time
}

func isWeekend(d time.Time) bool { return d.Weekday() == time.Saturday || d.Weekday() == time.Sunday }

// nextAfter returns the first occurrence strictly after d, and false when there is none (the
// schedule has ended).
func (s schedule) nextAfter(d time.Time) (time.Time, bool) {
	var next time.Time
	switch days, months := step(s.freq); {
	case d.Before(s.start):
		next = s.start
	case s.freq == FreqWeekdays:
		next = d.AddDate(0, 0, 1)
		for isWeekend(next) {
			next = next.AddDate(0, 0, 1)
		}
	case days > 0:
		next = s.start.AddDate(0, 0, (daysBetween(s.start, d)/days+1)*days)
	case months > 0:
		// Estimate the index from the month difference, then step forward past d.
		i := ((d.Year()-s.start.Year())*12 + int(d.Month()) - int(s.start.Month())) / months
		for i > 0 && addMonthsClamped(s.start, i*months).After(d) {
			i--
		}
		for next = addMonthsClamped(s.start, i*months); !next.After(d); next = addMonthsClamped(s.start, i*months) {
			i++
		}
	default:
		return time.Time{}, false
	}
	if s.end != nil && next.After(*s.end) {
		return time.Time{}, false
	}
	return next, true
}

// between returns the occurrences in (after, through], at most limit of them (0 means no limit),
// and whether more remain in that range.
func (s schedule) between(after, through time.Time, limit int) ([]time.Time, bool) {
	var out []time.Time
	for d, ok := s.nextAfter(after); ok && !d.After(through); d, ok = s.nextAfter(d) {
		if limit > 0 && len(out) == limit {
			return out, true
		}
		out = append(out, d)
	}
	return out, false
}

// nth returns the date of the n-th occurrence (1 is start).
func (s schedule) nth(n int) time.Time {
	d := s.start
	for i := 1; i < n; i++ {
		next, ok := s.nextAfter(d)
		if !ok {
			break
		}
		d = next
	}
	return d
}

// horizon is the last date a top-up creates occurrences for: today plus HorizonMonths, or the
// next occurrence after today when that is further out (a yearly rule always shows its next one).
func (s schedule) horizon(today time.Time) time.Time {
	h := addMonthsClamped(today, HorizonMonths)
	if next, ok := s.nextAfter(today); ok && next.After(h) {
		return next
	}
	return h
}

func maxDate(a, b time.Time) time.Time {
	if a.After(b) {
		return a
	}
	return b
}
