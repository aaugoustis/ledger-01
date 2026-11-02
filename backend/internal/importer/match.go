package importer

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

// NormNote is a description reduced for comparison: lower case, single spaces.
func NormNote(s string) string { return strings.ToLower(strings.Join(strings.Fields(s), " ")) }

// Fingerprint identifies a bank row by what the bank wrote: its date, signed amount and
// description, plus its ordinal among identical rows of the file (two equal coffees on one day
// are two rows). Sixteen hex characters of SHA-256. It stays stable when the transaction made
// from the row is later renamed or repriced, because it's stored apart from the transaction.
func Fingerprint(on time.Time, amount int64, note string, ordinal int) string {
	sum := sha256.Sum256(fmt.Appendf(nil, "%s|%d|%s|%d", on.Format(time.DateOnly), amount, NormNote(note), ordinal))
	return hex.EncodeToString(sum[:8])
}

// Fingerprints returns each row's fingerprint ("" for rows that can't be read).
func Fingerprints(rows []Row) []string {
	type key struct {
		on     time.Time
		amount int64
		note   string
	}
	seen := map[key]int{}
	out := make([]string, len(rows))
	for i, r := range rows {
		if r.Err != "" {
			continue
		}
		k := key{r.Date, r.Amount, NormNote(r.Note)}
		out[i] = Fingerprint(r.Date, r.Amount, r.Note, seen[k])
		seen[k]++
	}
	return out
}

// SameMerchant compares merchants as banks truncate them: equal, or one a prefix of the other
// (at least 4 characters).
func SameMerchant(a, b string) bool {
	if a == "" || b == "" {
		return false
	}
	if len(a) > len(b) {
		a, b = b, a
	}
	return a == b || (len(a) >= 4 && strings.HasPrefix(b, a))
}

// FitKind is how an adjustment fits a purchase.
type FitKind int

const (
	None FitKind = iota
	// Partial: it lowers the purchase.
	Partial
	// Full: a refund of all of it.
	Full
)

// Fit says how an adjustment (RoleDiscount or RoleRefund) listed on `on` for a positive amount
// fits a purchase on purchaseOn for a negative amount. It must come on the purchase's day or
// within window after it; only refunds can be full.
func Fit(role Role, window time.Duration, purchaseOn time.Time, purchase int64, on time.Time, amount int64) FitKind {
	gap := on.Sub(purchaseOn)
	if purchase >= 0 || amount <= 0 || gap < 0 || gap > window {
		return None
	}
	switch {
	case amount < -purchase:
		return Partial
	case amount == -purchase && role == RoleRefund:
		return Full
	}
	return None
}

// Pairing links adjustment rows to purchase rows of one file. Merged: partial adjustments
// (adjustment → purchase), counted into the purchase's amount. Refunded: full refunds
// (refund → purchase), which cancel the purchase out.
type Pairing struct {
	Merged   map[int]int
	Refunded map[int]int
}

// PairAdjustments matches adjustment rows to purchase rows of the same file, as the bank words
// them: the same merchant, on the purchase's day or within the adjustment's window after it, each
// purchase taking one adjustment. A full refund beats a partial fit; then the latest purchase
// wins (in a double charge, the second one); on the same day, the purchase listed nearest to the
// adjustment (banks list them side by side, oldest or newest first). skip leaves rows out
// (already matched elsewhere).
func PairAdjustments(rows []Row, bank Bank, skip func(i int) bool) Pairing {
	out := Pairing{Merged: map[int]int{}, Refunded: map[int]int{}}
	taken := map[int]bool{}
	for a, ar := range rows {
		if ar.Err != "" || skip(a) {
			continue
		}
		role, merchant, window, ok := bank.Adjustment(ar.Note)
		if !ok {
			continue
		}
		best, bestFit := -1, None
		for p, pr := range rows {
			if p == a || pr.Err != "" || taken[p] || skip(p) {
				continue
			}
			pm, ok := bank.Purchase(pr.Note)
			if !ok || !SameMerchant(merchant, pm) {
				continue
			}
			fit := Fit(role, window, pr.Date, pr.Amount, ar.Date, ar.Amount)
			if fit == None || (bestFit == Full && fit != Full) {
				continue
			}
			if best == -1 || closer(fit, pr.Date, p, bestFit, rows[best].Date, best, a) {
				best, bestFit = p, fit
			}
		}
		switch bestFit {
		case Partial:
			out.Merged[a] = best
		case Full:
			out.Refunded[a] = best
		default:
			continue
		}
		taken[best] = true
	}
	return out
}

// closer says whether purchase p (fit, listed on on) suits adjustment row a better than the
// current best one.
func closer(fit FitKind, on time.Time, p int, bestFit FitKind, bestOn time.Time, best, a int) bool {
	switch {
	case fit != bestFit:
		return fit > bestFit
	case !on.Equal(bestOn):
		return on.After(bestOn)
	}
	return abs(int64(p-a)) < abs(int64(best-a))
}
