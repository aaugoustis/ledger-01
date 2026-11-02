package ledger

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/franco-roura/ledger-01/backend/internal/db/dbq"
	"github.com/franco-roura/ledger-01/backend/internal/money"
	"github.com/google/uuid"
)

type TransactionKind = dbq.TransactionKind

const (
	KindExpense     = dbq.TransactionKindExpense
	KindIncome      = dbq.TransactionKindIncome
	KindTransferOut = dbq.TransactionKindTransferOut
	KindTransferIn  = dbq.TransactionKindTransferIn
)

// Counterpart is the other leg of a transfer, seen from this leg.
type Counterpart struct {
	TransferID uuid.UUID
	WalletID   uuid.UUID
	WalletName string
	Amount     int64
	Currency   string
	Rate       string
}

type Transaction struct {
	ID          uuid.UUID
	WalletID    uuid.UUID
	Kind        TransactionKind
	CategoryID  *uuid.UUID
	Amount      int64
	Currency    string
	OccurredOn  time.Time
	Note        string
	LabelIDs    []uuid.UUID
	Counterpart *Counterpart
	CreatedBy   *uuid.UUID
	CreatedAt   time.Time
	UpdatedAt   time.Time
	// Set when the row is an occurrence of a recurring rule, with the rule's schedule.
	RecurringRuleID *uuid.UUID
	Recurrence      *Recurrence
}

// Filter selects transactions of one wallet in a date range. Zero values mean "no filter".
type Filter struct {
	WalletID    uuid.UUID
	From, To    time.Time
	CategoryIDs []uuid.UUID
	// Rows created by any of these users (the by-person filter).
	MemberIDs []uuid.UUID
	NoteQuery string
	MinAmount *int64
	MaxAmount *int64
	// The caller's local date: rows after it are upcoming and left out. Zero means today in UTC.
	Today time.Time
}

func (f Filter) validate() error {
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
	return nil
}

// upToToday cuts the period off at today. ok is false when the whole period is upcoming.
func (f Filter) upToToday() (Filter, bool) {
	f.Today = orToday(f.Today)
	if f.To.After(f.Today) {
		f.To = f.Today
	}
	return f, !f.From.After(f.To)
}

type cursor struct {
	Date      string    `json:"d"`
	CreatedAt time.Time `json:"c"`
	ID        uuid.UUID `json:"i"`
}

func encodeCursor(t dbq.ListTransactionsRow) string {
	b, _ := json.Marshal(cursor{Date: money.FormatDate(t.OccurredOn), CreatedAt: t.CreatedAt, ID: t.ID})
	return base64.RawURLEncoding.EncodeToString(b)
}

func decodeCursor(token string) (*cursor, error) {
	if token == "" {
		return nil, nil
	}
	b, err := base64.RawURLEncoding.DecodeString(token)
	var c cursor
	if err != nil || json.Unmarshal(b, &c) != nil {
		return nil, invalid("page_token", "is malformed")
	}
	return &c, nil
}

// ListTransactions returns one page of a wallet's transactions up to today, newest first.
func (s *Service) ListTransactions(ctx context.Context, a Actor, f Filter, pageSize int, pageToken string) ([]Transaction, string, error) {
	if err := f.validate(); err != nil {
		return nil, "", err
	}
	if _, err := s.GetWallet(ctx, a, f.WalletID); err != nil {
		return nil, "", err
	}
	f, ok := f.upToToday()
	if !ok {
		return []Transaction{}, "", nil
	}
	if pageSize <= 0 || pageSize > 200 {
		pageSize = 50
	}
	cur, err := decodeCursor(pageToken)
	if err != nil {
		return nil, "", err
	}
	p := dbq.ListTransactionsParams{
		HouseholdID: a.HouseholdID,
		WalletID:    f.WalletID,
		FromDate:    f.From,
		ToDate:      f.To,
		CategoryIds: uniqueIDs(f.CategoryIDs),
		MemberIds:   uniqueIDs(f.MemberIDs),
		NoteQuery:   strings.TrimSpace(f.NoteQuery),
		MinAmount:   f.MinAmount,
		MaxAmount:   f.MaxAmount,
		PageLimit:   int32(pageSize + 1),
	}
	if cur != nil {
		d, err := money.ParseDate(cur.Date)
		if err != nil {
			return nil, "", invalid("page_token", "is malformed")
		}
		p.CursorDate, p.CursorCreatedAt, p.CursorID = &d, &cur.CreatedAt, &cur.ID
	}
	rows, err := s.q.ListTransactions(ctx, p)
	if err != nil {
		return nil, "", err
	}
	next := ""
	if len(rows) > pageSize {
		rows = rows[:pageSize]
		next = encodeCursor(rows[len(rows)-1])
	}
	out := make([]Transaction, len(rows))
	for i, r := range rows {
		out[i] = listedTransaction(r)
	}
	return out, next, nil
}

// Search selects transactions across wallets. Empty WalletIDs means every wallet of the household;
// the other fields are as in Filter (WalletID is ignored).
type Search struct {
	Filter
	WalletIDs []uuid.UUID
	LabelIDs  []uuid.UUID
	Kinds     []TransactionKind
}

// SearchTransactions returns one page of the household's transactions up to today, newest first.
func (s *Service) SearchTransactions(ctx context.Context, a Actor, sr Search, pageSize int, pageToken string) ([]Transaction, string, error) {
	if err := sr.validate(); err != nil {
		return nil, "", err
	}
	f, ok := sr.upToToday()
	if !ok {
		return []Transaction{}, "", nil
	}
	if pageSize <= 0 || pageSize > 200 {
		pageSize = 50
	}
	cur, err := decodeCursor(pageToken)
	if err != nil {
		return nil, "", err
	}
	kinds := make([]string, 0, len(sr.Kinds))
	for _, k := range sr.Kinds {
		if k != KindExpense && k != KindIncome && k != KindTransferOut && k != KindTransferIn {
			return nil, "", invalid("kinds", "unknown kind %q", k)
		}
		kinds = append(kinds, string(k))
	}
	p := dbq.SearchTransactionsParams{
		HouseholdID: a.HouseholdID,
		WalletIds:   uniqueIDs(sr.WalletIDs),
		FromDate:    f.From,
		ToDate:      f.To,
		Kinds:       kinds,
		CategoryIds: uniqueIDs(f.CategoryIDs),
		MemberIds:   uniqueIDs(f.MemberIDs),
		LabelIds:    uniqueIDs(sr.LabelIDs),
		NoteQuery:   strings.TrimSpace(f.NoteQuery),
		MinAmount:   f.MinAmount,
		MaxAmount:   f.MaxAmount,
		PageLimit:   int32(pageSize + 1),
	}
	if cur != nil {
		d, err := money.ParseDate(cur.Date)
		if err != nil {
			return nil, "", invalid("page_token", "is malformed")
		}
		p.CursorDate, p.CursorCreatedAt, p.CursorID = &d, &cur.CreatedAt, &cur.ID
	}
	rows, err := s.q.SearchTransactions(ctx, p)
	if err != nil {
		return nil, "", err
	}
	next := ""
	if len(rows) > pageSize {
		rows = rows[:pageSize]
		next = encodeCursor(dbq.ListTransactionsRow(rows[len(rows)-1]))
	}
	out := make([]Transaction, len(rows))
	for i, r := range rows {
		out[i] = listedTransaction(dbq.ListTransactionsRow(r))
	}
	return out, next, nil
}

func listedTransaction(r dbq.ListTransactionsRow) Transaction {
	t := Transaction{
		ID: r.ID, WalletID: r.WalletID, Kind: r.Kind, CategoryID: r.CategoryID, Amount: r.AmountMinor,
		Currency: strings.TrimSpace(r.Currency), OccurredOn: r.OccurredOn, Note: r.Note, LabelIDs: r.LabelIds,
		CreatedBy: r.CreatedBy, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt, RecurringRuleID: r.RecurringRuleID,
	}
	if r.TransferID != nil && r.CounterpartWalletID != nil {
		t.Counterpart = &Counterpart{
			TransferID: *r.TransferID, WalletID: *r.CounterpartWalletID, WalletName: r.CounterpartWalletName,
			Amount: r.CounterpartAmountMinor, Currency: strings.TrimSpace(r.CounterpartCurrency), Rate: r.TransferRate,
		}
	}
	if r.RecurringRuleID != nil && r.RuleFrequency != nil {
		t.Recurrence = &Recurrence{Frequency: *r.RuleFrequency, End: r.RuleEndDate}
	}
	return t
}

// ListUpcoming returns rows dated after today, soonest first (at most 500): one wallet's, or
// with a nil wallet every non-archived wallet's, each transfer once as its outgoing leg. Non-empty
// memberIDs keeps the rows those users created.
func (s *Service) ListUpcoming(ctx context.Context, a Actor, walletID *uuid.UUID, memberIDs []uuid.UUID, today time.Time) ([]Transaction, error) {
	if walletID != nil {
		if _, err := s.GetWallet(ctx, a, *walletID); err != nil {
			return nil, err
		}
	}
	rows, err := s.q.ListUpcomingTransactions(ctx, dbq.ListUpcomingTransactionsParams{
		HouseholdID: a.HouseholdID, Today: orToday(today), WalletID: walletID, MemberIds: uniqueIDs(memberIDs),
	})
	if err != nil {
		return nil, err
	}
	out := make([]Transaction, len(rows))
	for i, r := range rows {
		out[i] = listedTransaction(dbq.ListTransactionsRow(r))
	}
	return out, nil
}

type Summary struct {
	Currency         string
	MinorUnits       int
	Balance          int64
	Change           int64
	Expenses         int64
	Income           int64
	TransactionCount int64
}

// PeriodSummary totals the filtered period up to today; Balance is the wallet's balance today.
func (s *Service) PeriodSummary(ctx context.Context, a Actor, f Filter) (Summary, error) {
	if err := f.validate(); err != nil {
		return Summary{}, err
	}
	w, err := s.GetWalletOn(ctx, a, f.WalletID, orToday(f.Today))
	if err != nil {
		return Summary{}, err
	}
	f, ok := f.upToToday()
	if !ok {
		return Summary{Currency: w.Currency, MinorUnits: w.MinorUnits, Balance: w.Balance}, nil
	}
	r, err := s.q.PeriodSummary(ctx, dbq.PeriodSummaryParams{
		HouseholdID: a.HouseholdID, WalletID: f.WalletID, FromDate: f.From, ToDate: f.To,
		CategoryIds: uniqueIDs(f.CategoryIDs), MemberIds: uniqueIDs(f.MemberIDs), NoteQuery: strings.TrimSpace(f.NoteQuery),
		MinAmount: f.MinAmount, MaxAmount: f.MaxAmount,
	})
	if err != nil {
		return Summary{}, err
	}
	return Summary{
		Currency: w.Currency, MinorUnits: w.MinorUnits, Balance: w.Balance, Change: r.ChangeMinor,
		Expenses: r.ExpensesMinor, Income: r.IncomeMinor, TransactionCount: r.TransactionCount,
	}, nil
}

// TransactionInput is an expense or income. The kind follows the category's kind.
type TransactionInput struct {
	WalletID   uuid.UUID
	CategoryID uuid.UUID
	Amount     int64
	OccurredOn time.Time
	Note       string
	LabelIDs   []uuid.UUID
	// Optional: reject the input if the category's kind differs.
	ExpectKind TransactionKind

	// Optional: makes a new row (or an edited row that doesn't repeat yet) the first occurrence of
	// a recurring rule. On a THIS_AND_FUTURE edit it is the series' new schedule.
	Repeat Recurrence
	// For occurrences of a recurring rule: which occurrences an edit applies to.
	Scope Scope
	// The caller's local date; zero means today in UTC.
	Today time.Time
}

func checkNote(note string) (string, error) {
	note = strings.TrimSpace(note)
	if utf8.RuneCountInString(note) > 500 {
		return "", invalid("note", "must be at most 500 characters")
	}
	return note, nil
}

// prepare validates an expense/income and returns its kind and cleaned fields.
func (s *Service) prepare(ctx context.Context, q *dbq.Queries, a Actor, in *TransactionInput) (TransactionKind, Wallet, error) {
	if err := checkAmount("amount_minor", in.Amount, false); err != nil {
		return "", Wallet{}, err
	}
	if in.OccurredOn.IsZero() {
		return "", Wallet{}, invalid("occurred_on", "is required")
	}
	note, err := checkNote(in.Note)
	if err != nil {
		return "", Wallet{}, err
	}
	in.Note = note
	w, err := s.GetWallet(ctx, a, in.WalletID)
	if err != nil {
		return "", Wallet{}, err
	}
	if w.Archived {
		return "", Wallet{}, invalid("wallet_id", "wallet is archived")
	}
	c, err := q.GetCategory(ctx, dbq.GetCategoryParams{ID: in.CategoryID, HouseholdID: a.HouseholdID})
	if err != nil {
		return "", Wallet{}, invalid("category_id", "category doesn't exist")
	}
	if in.LabelIDs, err = checkLabels(ctx, q, a, in.LabelIDs); err != nil {
		return "", Wallet{}, err
	}
	kind := KindExpense
	if c.Kind == CategoryIncome {
		kind = KindIncome
	}
	if in.ExpectKind != "" && in.ExpectKind != kind {
		return "", Wallet{}, invalid("category_id", "is an %s category, not %s", c.Kind, in.ExpectKind)
	}
	return kind, w, nil
}

// recurringLink marks a new row as a recurring rule's occurrence (zero for plain rows).
type recurringLink struct {
	ruleID *uuid.UUID
	on     *time.Time
}

// insertTransaction writes a validated expense or income and its labels.
func insertTransaction(ctx context.Context, q *dbq.Queries, householdID uuid.UUID, createdBy *uuid.UUID, kind TransactionKind,
	in TransactionInput, rl recurringLink,
) (dbq.Transaction, error) {
	t, err := q.CreateTransaction(ctx, dbq.CreateTransactionParams{
		ID: newID(), HouseholdID: householdID, WalletID: in.WalletID, Kind: kind, CategoryID: &in.CategoryID,
		AmountMinor: in.Amount, OccurredOn: in.OccurredOn, Note: in.Note, CreatedBy: createdBy,
		RecurringRuleID: rl.ruleID, OccurrenceDate: rl.on,
	})
	if err != nil {
		return dbq.Transaction{}, dbErr(err, "transaction")
	}
	if len(in.LabelIDs) > 0 {
		if err := q.AddTransactionLabels(ctx, dbq.AddTransactionLabelsParams{TransactionID: t.ID, LabelIds: in.LabelIDs}); err != nil {
			return dbq.Transaction{}, err
		}
	}
	return t, nil
}

// CreateTransaction adds an expense or income. With in.Repeat it becomes the first occurrence of a
// recurring rule, and the following occurrences are created up to the horizon.
func (s *Service) CreateTransaction(ctx context.Context, a Actor, in TransactionInput) (Transaction, error) {
	var out Transaction
	err := s.inTx(ctx, func(q *dbq.Queries) error {
		kind, w, err := s.prepare(ctx, q, a, &in)
		if err != nil {
			return err
		}
		if err := in.Repeat.normalize(in.OccurredOn); err != nil {
			return err
		}
		t, err := insertTransaction(ctx, q, a.HouseholdID, &a.UserID, kind, in, recurringLink{})
		if err != nil {
			return err
		}
		if in.Repeat.Repeats() {
			if err := s.createRule(ctx, q, a, transactionTemplate(kind, in), in.Repeat, in.OccurredOn,
				orToday(in.Today), occurrence{txID: t.ID}); err != nil {
				return err
			}
		}
		out, err = s.reloadTransaction(ctx, q, a, t.ID, w.Currency, in.LabelIDs)
		return err
	})
	return out, err
}

// reloadTransaction reads a row back after a write, with its recurring rule's schedule.
func (s *Service) reloadTransaction(ctx context.Context, q *dbq.Queries, a Actor, id uuid.UUID, currency string, labels []uuid.UUID) (Transaction, error) {
	t, err := q.GetTransaction(ctx, dbq.GetTransactionParams{ID: id, HouseholdID: a.HouseholdID})
	if err != nil {
		return Transaction{}, dbErr(err, "transaction")
	}
	out := transactionFrom(t, currency, labels)
	out.Recurrence, err = recurrenceOf(ctx, q, a.HouseholdID, t.RecurringRuleID)
	return out, err
}

// UpdateTransaction replaces an expense/income's fields. Transfer legs go through UpdateTransfer.
// On an occurrence of a recurring rule, ScopeThisAndFuture also rewrites the series from it on
// (see editSeries). On a row that doesn't repeat, in.Repeat makes it the first occurrence of a
// new rule.
func (s *Service) UpdateTransaction(ctx context.Context, a Actor, id uuid.UUID, in TransactionInput) (Transaction, error) {
	var out Transaction
	err := s.inTx(ctx, func(q *dbq.Queries) error {
		cur, err := q.GetTransaction(ctx, dbq.GetTransactionParams{ID: id, HouseholdID: a.HouseholdID})
		if err != nil {
			return dbErr(err, "transaction")
		}
		if cur.TransferID != nil {
			return invalid("id", "is a transfer; use UpdateTransfer")
		}
		in.WalletID = cur.WalletID
		kind, w, err := s.prepare(ctx, q, a, &in)
		if err != nil {
			return err
		}
		recurring := cur.RecurringRuleID != nil && cur.OccurrenceDate != nil
		series := recurring && in.Scope == ScopeThisAndFuture
		if !series && recurring {
			in.Repeat = Recurrence{}
		}
		if err := in.Repeat.normalize(in.OccurredOn); err != nil {
			return err
		}
		t, err := q.UpdateTransaction(ctx, dbq.UpdateTransactionParams{
			ID: id, HouseholdID: a.HouseholdID, Kind: kind, CategoryID: &in.CategoryID, AmountMinor: in.Amount,
			OccurredOn: in.OccurredOn, Note: in.Note,
		})
		if err != nil {
			return dbErr(err, "transaction")
		}
		if err := q.ClearTransactionLabels(ctx, []uuid.UUID{id}); err != nil {
			return err
		}
		if len(in.LabelIDs) > 0 {
			if err := q.AddTransactionLabels(ctx, dbq.AddTransactionLabelsParams{TransactionID: id, LabelIds: in.LabelIDs}); err != nil {
				return err
			}
		}
		today, tpl, chosen := orToday(in.Today), transactionTemplate(kind, in), occurrence{txID: id}
		switch {
		case series:
			err = s.editSeries(ctx, q, a, *cur.RecurringRuleID, *cur.OccurrenceDate, in.OccurredOn, tpl, in.Repeat, today, chosen)
		case !recurring && in.Repeat.Repeats():
			err = s.createRule(ctx, q, a, tpl, in.Repeat, in.OccurredOn, today, chosen)
		}
		if err != nil {
			return err
		}
		out, err = s.reloadTransaction(ctx, q, a, t.ID, w.Currency, in.LabelIDs)
		return err
	})
	return out, err
}

// DeleteTransactions deletes the given transactions. Deleting either leg of a transfer deletes
// the whole transfer. Returns how many transaction rows were removed.
func (s *Service) DeleteTransactions(ctx context.Context, a Actor, ids []uuid.UUID) (int64, error) {
	return s.DeleteOccurrences(ctx, a, ids, ScopeOnlyThis, time.Time{})
}

// DeleteOccurrences deletes the given transactions like DeleteTransactions. With
// ScopeThisAndFuture, each chosen occurrence of a recurring rule also ends its rule there: the
// rule's later occurrences that aren't due yet (dated after today) go too, and it creates no more.
func (s *Service) DeleteOccurrences(ctx context.Context, a Actor, ids []uuid.UUID, scope Scope, today time.Time) (int64, error) {
	ids = uniqueIDs(ids)
	if len(ids) == 0 {
		return 0, nil
	}
	if len(ids) > 500 {
		return 0, invalid("ids", "at most 500 per request")
	}
	var deleted int64
	err := s.inTx(ctx, func(q *dbq.Queries) error {
		refs, err := q.GetTransactionRefs(ctx, dbq.GetTransactionRefsParams{HouseholdID: a.HouseholdID, Ids: ids})
		if err != nil {
			return err
		}
		var plain, transfers []uuid.UUID
		from := map[uuid.UUID]time.Time{} // rule -> earliest chosen occurrence
		var rules []uuid.UUID
		for _, r := range refs {
			if r.TransferID != nil {
				transfers = append(transfers, *r.TransferID)
			} else {
				plain = append(plain, r.ID)
			}
			if scope == ScopeThisAndFuture && r.RecurringRuleID != nil && r.OccurrenceDate != nil {
				d, seen := from[*r.RecurringRuleID]
				if !seen {
					rules = append(rules, *r.RecurringRuleID)
				}
				if !seen || r.OccurrenceDate.Before(d) {
					from[*r.RecurringRuleID] = *r.OccurrenceDate
				}
			}
		}
		for _, id := range rules {
			n, err := endSeries(ctx, q, a.HouseholdID, id, from[id], orToday(today))
			if err != nil {
				return err
			}
			deleted += n
		}
		n, err := deleteRows(ctx, q, a.HouseholdID, plain, transfers)
		if err != nil {
			return err
		}
		deleted += n
		for _, id := range rules {
			if err := dropIfEmpty(ctx, q, a.HouseholdID, id); err != nil {
				return err
			}
		}
		return nil
	})
	return deleted, err
}

func transactionFrom(t dbq.Transaction, currency string, labels []uuid.UUID) Transaction {
	if labels == nil {
		labels = []uuid.UUID{}
	}
	return Transaction{
		ID: t.ID, WalletID: t.WalletID, Kind: t.Kind, CategoryID: t.CategoryID, Amount: t.AmountMinor,
		Currency: strings.TrimSpace(currency), OccurredOn: t.OccurredOn, Note: t.Note, LabelIDs: labels,
		CreatedBy: t.CreatedBy, CreatedAt: t.CreatedAt, UpdatedAt: t.UpdatedAt, RecurringRuleID: t.RecurringRuleID,
	}
}
