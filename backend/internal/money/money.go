// Package money validates amounts, ISO dates and exchange rates. Amounts are int64 minor units;
// rates are decimal strings handled with math/big so nothing goes through floats.
package money

import (
	"errors"
	"fmt"
	"math/big"
	"regexp"
	"time"
)

// RateScale is the number of decimals stored for rates (numeric(20,10)).
const RateScale = 10

var rateRe = regexp.MustCompile(`^\d{1,10}(\.\d{1,` + fmt.Sprint(RateScale) + `})?$`)

// ParseRate parses a positive decimal rate such as "39.05".
func ParseRate(s string) (*big.Rat, error) {
	if !rateRe.MatchString(s) {
		return nil, fmt.Errorf("rate must be a positive decimal with at most %d decimals", RateScale)
	}
	r, ok := new(big.Rat).SetString(s)
	if !ok || r.Sign() <= 0 {
		return nil, errors.New("rate must be greater than zero")
	}
	return r, nil
}

// FormatRate renders a rate with RateScale decimals.
func FormatRate(r *big.Rat) string { return r.FloatString(RateScale) }

// CheckTransfer verifies that to ≈ from × rate, allowing for each amount having been rounded to
// its currency's minor units (e.g. from computed as to ÷ rate, then rounded to cents).
//
//	|to − from×r'| ≤ r'/2 + 1/2 (+ a hair), where r' = rate × 10^(toUnits − fromUnits)
func CheckTransfer(fromMinor, toMinor int64, rate *big.Rat, fromUnits, toUnits int) error {
	if fromMinor <= 0 || toMinor <= 0 {
		return errors.New("transfer amounts must be positive")
	}
	scaled := new(big.Rat).Mul(rate, pow10Rat(toUnits-fromUnits))
	expected := new(big.Rat).Mul(new(big.Rat).SetInt64(fromMinor), scaled)
	diff := new(big.Rat).Sub(new(big.Rat).SetInt64(toMinor), expected)
	diff.Abs(diff)

	half := big.NewRat(1, 2)
	tolerance := new(big.Rat).Add(new(big.Rat).Mul(scaled, half), half)
	tolerance.Add(tolerance, big.NewRat(1, 1000))
	if diff.Cmp(tolerance) > 0 {
		return fmt.Errorf("amounts don't match the rate: expected about %s, got %d (minor units)",
			expected.FloatString(0), toMinor)
	}
	return nil
}

func pow10Rat(n int) *big.Rat {
	if n >= 0 {
		return new(big.Rat).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(n)), nil))
	}
	return new(big.Rat).SetFrac(big.NewInt(1), new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(-n)), nil))
}

// ParseDate parses a "YYYY-MM-DD" calendar date (UTC midnight).
func ParseDate(s string) (time.Time, error) {
	t, err := time.Parse(time.DateOnly, s)
	if err != nil {
		return time.Time{}, fmt.Errorf("date must be YYYY-MM-DD, got %q", s)
	}
	return t, nil
}

// FormatDate renders a date as "YYYY-MM-DD".
func FormatDate(t time.Time) string { return t.Format(time.DateOnly) }

// FormatMinor renders minor units as a decimal string ("-12.50") without going through floats.
func FormatMinor(minor int64, units int) string {
	sign := ""
	u := uint64(minor)
	if minor < 0 {
		sign, u = "-", uint64(-minor)
	}
	if units <= 0 {
		return fmt.Sprintf("%s%d", sign, u)
	}
	p := uint64(1)
	for range units {
		p *= 10
	}
	return fmt.Sprintf("%s%d.%0*d", sign, u/p, units, u%p)
}
