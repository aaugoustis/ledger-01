package ledger

import (
	"context"
	"math/big"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/franco-roura/ledger-01/backend/internal/db/dbq"
	"github.com/google/uuid"
)

type Bucket int

const (
	BucketDay Bucket = iota + 1
	BucketWeek
	BucketMonth
)

// ReportFilter selects what a report covers.
type ReportFilter struct {
	From, To    time.Time
	WalletIDs   []uuid.UUID // empty: every visible wallet
	CategoryIDs []uuid.UUID // narrows flows only
	// Case-insensitive note substring and bounds on the amount in the report currency (minor
	// units); like CategoryIDs they narrow flows only, transfers included.
	NoteQuery string
	MinAmount *int64
	MaxAmount *int64
	// The caller's local date; rows after it are upcoming and left out. Zero means today in UTC.
	Today time.Time
	// Report currency; empty means the user's base currency.
	Currency string
}

type SeriesPoint struct {
	Date    time.Time
	Balance int64
	Income  int64
	Expense int64
}

type CategoryTotal struct {
	CategoryID uuid.UUID
	Amount     int64
	Count      int64
}

type Dashboard struct {
	Currency          string
	Balance           int64
	Change            int64
	Expenses          int64
	Income            int64
	Series            []SeriesPoint
	ExpenseCategories []CategoryTotal
	IncomeCategories  []CategoryTotal
	TransfersIn       int64
	TransfersOut      int64
	TransfersInCount  int64
	TransfersOutCount int64
	Unconverted       []string
}

func (f ReportFilter) validate() error {
	if f.From.IsZero() || f.To.IsZero() {
		return invalid("period", "from and to are required")
	}
	if f.To.Before(f.From) {
		return invalid("period", "to must not be before from")
	}
	if f.To.Sub(f.From) > 20*366*24*time.Hour {
		return invalid("period", "range is too long")
	}
	if len(f.NoteQuery) > 200 {
		return invalid("note_query", "is too long")
	}
	if f.MinAmount != nil && f.MaxAmount != nil && *f.MinAmount > *f.MaxAmount {
		return invalid("min_amount_minor", "must not be more than the maximum")
	}
	return nil
}

// flowMatch reports whether a row with this note and converted amount passes the note and amount
// filters.
func (f ReportFilter) flowMatch(note string, amount int64) bool {
	if q := strings.TrimSpace(f.NoteQuery); q != "" && !strings.Contains(strings.ToLower(note), strings.ToLower(q)) {
		return false
	}
	return (f.MinAmount == nil || amount >= *f.MinAmount) && (f.MaxAmount == nil || amount <= *f.MaxAmount)
}

// bucketStarts returns the first day of every bucket covering [from, to]. Weeks start on Monday.
func bucketStarts(from, to time.Time, b Bucket) []time.Time {
	start := from
	switch b {
	case BucketWeek:
		start = from.AddDate(0, 0, -((int(from.Weekday()) + 6) % 7))
	case BucketMonth:
		start = time.Date(from.Year(), from.Month(), 1, 0, 0, 0, 0, time.UTC)
	}
	var out []time.Time
	for d := start; !d.After(to); {
		out = append(out, d)
		switch b {
		case BucketWeek:
			d = d.AddDate(0, 0, 7)
		case BucketMonth:
			d = d.AddDate(0, 1, 0)
		default:
			d = d.AddDate(0, 0, 1)
		}
	}
	return out
}

// bucketIndex finds the bucket a date falls in (starts are sorted).
func bucketIndex(starts []time.Time, d time.Time) int {
	return sort.Search(len(starts), func(i int) bool { return starts[i].After(d) }) - 1
}

// report gathers what both reports need: wallets, their transactions and a converter.
type report struct {
	base      string
	wallets   map[uuid.UUID]dbq.ReportWalletsRow
	txs       []dbq.ReportTransactionsRow
	conv      *Converter
	missing   map[string]bool
	starts    []time.Time
	walletIDs []uuid.UUID
}

func (s *Service) loadReport(ctx context.Context, a Actor, f ReportFilter, b Bucket) (*report, error) {
	if err := f.validate(); err != nil {
		return nil, err
	}
	if b < BucketDay || b > BucketMonth {
		b = BucketDay
	}
	hh, err := s.q.GetHousehold(ctx, a.HouseholdID)
	if err != nil {
		return nil, dbErr(err, "household")
	}
	base := strings.TrimSpace(hh.BaseCurrency)
	if f.Currency != "" {
		base = strings.ToUpper(strings.TrimSpace(f.Currency))
		if _, err := s.q.GetCurrency(ctx, base); err != nil {
			return nil, invalid("currency", "unknown currency %q", base)
		}
	}
	today := orToday(f.Today)
	ws, err := s.q.ReportWallets(ctx, dbq.ReportWalletsParams{
		HouseholdID: a.HouseholdID, FromDate: f.From, Today: today, WalletIds: uniqueIDs(f.WalletIDs),
	})
	if err != nil {
		return nil, err
	}
	r := &report{
		base:    base,
		wallets: map[uuid.UUID]dbq.ReportWalletsRow{},
		missing: map[string]bool{},
		starts:  bucketStarts(f.From, f.To, b),
	}
	for _, w := range ws {
		w.Currency = strings.TrimSpace(w.Currency)
		r.wallets[w.ID] = w
		r.walletIDs = append(r.walletIDs, w.ID)
	}
	if r.txs, err = s.q.ReportTransactions(ctx, dbq.ReportTransactionsParams{
		HouseholdID: a.HouseholdID, WalletIds: r.walletIDs, FromDate: f.From, ToDate: f.To, Today: today,
	}); err != nil {
		return nil, err
	}

	if r.conv, err = s.converter(ctx, a, today); err != nil {
		return nil, err
	}
	return r, nil
}

// convert converts to the base currency, recording currencies with no known rate. Amounts
// that can't be converted are left out of totals and counts.
func (r *report) convert(amount int64, currency string, on time.Time) (int64, bool) {
	v, ok := r.conv.Convert(amount, currency, r.base, on)
	if !ok {
		r.missing[currency] = true
	}
	return v, ok
}

func (r *report) unconverted() []string {
	out := make([]string, 0, len(r.missing))
	for c := range r.missing {
		out = append(out, c)
	}
	slices.Sort(out)
	return out
}

func signed(kind TransactionKind, amount int64) int64 {
	if kind == KindIncome || kind == KindTransferIn {
		return amount
	}
	return -amount
}

// Dashboard builds the household overview for a period.
func (s *Service) Dashboard(ctx context.Context, a Actor, f ReportFilter, b Bucket) (Dashboard, error) {
	r, err := s.loadReport(ctx, a, f, b)
	if err != nil {
		return Dashboard{}, err
	}
	out := Dashboard{Currency: r.base, Series: make([]SeriesPoint, len(r.starts))}
	for i, st := range r.starts {
		out.Series[i].Date = st
	}
	catFilter := map[uuid.UUID]bool{}
	for _, id := range f.CategoryIDs {
		catFilter[id] = true
	}

	// Running balance per currency, then per bucket end.
	running := map[string]int64{}
	for _, w := range r.wallets {
		running[w.Currency] += w.OpeningMinor
	}
	expenseCats := map[uuid.UUID]*CategoryTotal{}
	incomeCats := map[uuid.UUID]*CategoryTotal{}
	ti := 0
	for i := range r.starts {
		end := f.To
		if i+1 < len(r.starts) && r.starts[i+1].AddDate(0, 0, -1).Before(end) {
			end = r.starts[i+1].AddDate(0, 0, -1)
		}
		for ; ti < len(r.txs) && !r.txs[ti].OccurredOn.After(end); ti++ {
			t := r.txs[ti]
			cur := r.wallets[t.WalletID].Currency
			running[cur] += signed(t.Kind, t.AmountMinor)
			v, ok := r.convert(t.AmountMinor, cur, t.OccurredOn)
			if !ok || !f.flowMatch(t.Note, v) {
				continue
			}
			switch t.Kind {
			case KindTransferIn:
				out.TransfersIn += v
				out.TransfersInCount++
				out.Change += v
				continue
			case KindTransferOut:
				out.TransfersOut += v
				out.TransfersOutCount++
				out.Change -= v
				continue
			}
			if len(catFilter) > 0 && (t.CategoryID == nil || !catFilter[*t.CategoryID]) {
				continue
			}
			totals := expenseCats
			if t.Kind == KindIncome {
				totals = incomeCats
				out.Income += v
				out.Series[i].Income += v
				out.Change += v
			} else {
				out.Expenses += v
				out.Series[i].Expense += v
				out.Change -= v
			}
			if t.CategoryID != nil {
				ct := totals[*t.CategoryID]
				if ct == nil {
					ct = &CategoryTotal{CategoryID: *t.CategoryID}
					totals[*t.CategoryID] = ct
				}
				ct.Amount += v
				ct.Count++
			}
		}
		var bal int64
		for cur, amount := range running {
			v, _ := r.convert(amount, cur, end)
			bal += v
		}
		out.Series[i].Balance = bal
	}
	if n := len(out.Series); n > 0 {
		out.Balance = out.Series[n-1].Balance
	}
	out.ExpenseCategories = sortedTotals(expenseCats)
	out.IncomeCategories = sortedTotals(incomeCats)
	out.Unconverted = r.unconverted()
	return out, nil
}

func sortedTotals(m map[uuid.UUID]*CategoryTotal) []CategoryTotal {
	out := make([]CategoryTotal, 0, len(m))
	for _, ct := range m {
		out = append(out, *ct)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Amount != out[j].Amount {
			return out[i].Amount > out[j].Amount
		}
		return out[i].CategoryID.String() < out[j].CategoryID.String()
	})
	return out
}

type CategoryReport struct {
	Currency       string
	Total          int64
	Count          int64
	Average        int64
	BusiestWeekday int // -1 when there are no transactions
	Series         []SeriesPoint
	Transactions   []Transaction
	Unconverted    []string
}

// CategoryReportFor builds the drill-down for one category over a period.
func (s *Service) CategoryReportFor(ctx context.Context, a Actor, categoryID uuid.UUID, f ReportFilter, b Bucket) (CategoryReport, error) {
	cat, err := s.getCategory(ctx, a, categoryID)
	if err != nil {
		return CategoryReport{}, err
	}
	r, err := s.loadReport(ctx, a, f, b)
	if err != nil {
		return CategoryReport{}, err
	}
	out := CategoryReport{Currency: r.base, BusiestWeekday: -1, Series: make([]SeriesPoint, len(r.starts))}
	for i, st := range r.starts {
		out.Series[i].Date = st
	}
	var weekdays [7]int64
	for _, t := range r.txs {
		if t.CategoryID == nil || *t.CategoryID != categoryID {
			continue
		}
		v, ok := r.convert(t.AmountMinor, r.wallets[t.WalletID].Currency, t.OccurredOn)
		if !ok || !f.flowMatch(t.Note, v) {
			continue
		}
		out.Total += v
		out.Count++
		weekdays[t.OccurredOn.Weekday()]++
		if i := bucketIndex(r.starts, t.OccurredOn); i >= 0 {
			if cat.Kind == CategoryIncome {
				out.Series[i].Income += v
			} else {
				out.Series[i].Expense += v
			}
		}
	}
	if out.Count > 0 {
		out.Average = roundHalfEven(big.NewRat(out.Total, out.Count))
		best := int64(0)
		for d, n := range weekdays {
			if n > best {
				best, out.BusiestWeekday = n, d
			}
		}
	}

	rows, err := s.q.CategoryTransactions(ctx, dbq.CategoryTransactionsParams{
		HouseholdID: a.HouseholdID, CategoryID: &categoryID, WalletIds: r.walletIDs, FromDate: f.From, ToDate: f.To,
		Today: orToday(f.Today),
	})
	if err != nil {
		return CategoryReport{}, err
	}
	for _, row := range rows {
		if f.NoteQuery != "" || f.MinAmount != nil || f.MaxAmount != nil {
			v, ok := r.conv.Convert(row.AmountMinor, strings.TrimSpace(row.Currency), r.base, row.OccurredOn)
			if !ok || !f.flowMatch(row.Note, v) {
				continue
			}
		}
		out.Transactions = append(out.Transactions, Transaction{
			ID: row.ID, WalletID: row.WalletID, Kind: row.Kind, CategoryID: row.CategoryID, Amount: row.AmountMinor,
			Currency: strings.TrimSpace(row.Currency), OccurredOn: row.OccurredOn, Note: row.Note, LabelIDs: row.LabelIds,
			CreatedBy: row.CreatedBy, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
		})
		// The response limit applies to matching rows, including amount filters in the
		// report currency, rather than to the first unfiltered rows in the database.
		if len(out.Transactions) == 200 {
			break
		}
	}
	out.Unconverted = r.unconverted()
	return out, nil
}
