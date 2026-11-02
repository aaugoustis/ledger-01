package ledger

import (
	"math/big"
	"testing"
	"time"
)

func d(s string) time.Time {
	t, _ := time.Parse(time.DateOnly, s)
	return t
}

func rat(s string) *big.Rat { r, _ := new(big.Rat).SetString(s); return r }

func TestConverter(t *testing.T) {
	c := NewConverter([]RatePoint{
		{On: d("2026-09-10"), From: "USD", To: "UYU", Rate: rat("40")},
		{On: d("2026-10-27"), From: "USD", To: "UYU", Rate: rat("39.05")},
		{On: d("2026-10-05"), From: "UYU", To: "USD", Rate: rat("0.025")}, // reverse direction = 40
		{On: d("2026-10-01"), From: "EUR", To: "JPY", Rate: rat("160")},
	}, map[string]int{"USD": 2, "UYU": 2, "EUR": 2, "JPY": 0})

	tests := []struct {
		name     string
		amount   int64
		from, to string
		on       string
		want     int64
		ok       bool
	}{
		{"same currency", 1234, "USD", "USD", "2026-10-01", 1234, true},
		{"latest on or before", 390500, "UYU", "USD", "2026-10-28", 10000, true}, // 3905 UYU / 39.05
		{"reverse-direction point", 400000, "UYU", "USD", "2026-10-06", 10000, true},
		{"before any point uses the earliest", 10000, "USD", "UYU", "2026-01-01", 400000, true},
		{"different decimals", 1000, "EUR", "JPY", "2026-10-02", 1600, true},
		{"inverse with decimals", 1600, "JPY", "EUR", "2026-10-02", 1000, true},
		{"no rate", 100, "USD", "EUR", "2026-10-02", 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := c.Convert(tt.amount, tt.from, tt.to, d(tt.on))
			if ok != tt.ok || got != tt.want {
				t.Fatalf("Convert = %d, %v; want %d, %v", got, ok, tt.want, tt.ok)
			}
		})
	}
}

func TestBuckets(t *testing.T) {
	weeks := bucketStarts(d("2026-10-01"), d("2026-10-31"), BucketWeek)
	if weeks[0] != d("2026-09-28") || len(weeks) != 5 {
		t.Fatalf("weeks = %v", weeks)
	}
	months := bucketStarts(d("2026-08-15"), d("2026-10-31"), BucketMonth)
	if len(months) != 3 || months[0] != d("2026-08-01") {
		t.Fatalf("months = %v", months)
	}
	if i := bucketIndex(weeks, d("2026-10-06")); i != 1 {
		t.Fatalf("bucketIndex = %d", i)
	}
}
