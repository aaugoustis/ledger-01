package money

import (
	"math/big"
	"testing"
)

func TestParseRate(t *testing.T) {
	for _, ok := range []string{"1", "39.05", "0.0000000001", "151.234"} {
		if _, err := ParseRate(ok); err != nil {
			t.Errorf("ParseRate(%q) = %v, want ok", ok, err)
		}
	}
	for _, bad := range []string{"", "0", "-1", "1e3", "abc", "1.12345678901", ".5"} {
		if _, err := ParseRate(bad); err == nil {
			t.Errorf("ParseRate(%q) succeeded, want error", bad)
		}
	}
}

func TestCheckTransfer(t *testing.T) {
	r := func(s string) *big.Rat { v, _ := ParseRate(s); return v }
	tests := []struct {
		name               string
		from, to           int64
		rate               string
		fromUnits, toUnits int
		ok                 bool
	}{
		{"exact", 10000, 390500, "39.05", 2, 2, true},
		{"derived rate", 12000, 470000, "39.1666666667", 2, 2, true},
		// from = round(3905.00 / 39.10) = 99.87 -> 99.87 × 39.10 = 3904.917, off by 8.3 cents < 19.55
		{"from rounded after division", 9987, 390500, "39.10", 2, 2, true},
		{"jpy target", 1234, 1866, "151.234", 2, 0, true},
		{"way off", 10000, 400000, "39.05", 2, 2, false},
		{"same currency mismatch", 10000, 10002, "1", 2, 2, false},
		{"same currency exact", 10000, 10000, "1", 2, 2, true},
		{"non-positive", 0, 100, "1", 2, 2, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CheckTransfer(tt.from, tt.to, r(tt.rate), tt.fromUnits, tt.toUnits)
			if (err == nil) != tt.ok {
				t.Fatalf("CheckTransfer() err = %v, want ok=%v", err, tt.ok)
			}
		})
	}
}

func TestParseDate(t *testing.T) {
	if _, err := ParseDate("2026-10-29"); err != nil {
		t.Fatal(err)
	}
	if _, err := ParseDate("29/10/2026"); err == nil {
		t.Fatal("want error")
	}
}
