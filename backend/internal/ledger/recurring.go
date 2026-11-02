package ledger

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/franco-roura/ledger-01/backend/internal/db/dbq"
	"github.com/google/uuid"
)

// Recurring rules pre-create their occurrences as ordinary, future-dated transactions (ADR:
// pre-create future rows). A rule is a template plus a schedule whose first occurrence is the
// transaction it was made from. Occurrences exist up to the rule's generated_through, which only
// moves forward: requests extend it as they create or edit series, and the top-up job (cmd/jobs)
// keeps every rule HorizonMonths ahead. Rows dated after the caller's today are upcoming and
// never count in balances, reports or budgets.

// Recurrence is a repeat schedule. The zero value means the transaction doesn't repeat.
type Recurrence struct {
	Frequency Frequency
	// Last date an occurrence may fall on; nil repeats forever.
	End *time.Time
	// Alternative to End: stop after this many occurrences, the first included. It is stored as
	// End, the date of the last one.
	Count int
}

func (r Recurrence) Repeats() bool { return r.Frequency != "" }

// normalize validates r for a series whose first occurrence is on start and turns Count into End.
func (r *Recurrence) normalize(start time.Time) error {
	if !r.Repeats() {
		r.End, r.Count = nil, 0
		return nil
	}
	if !validFrequency(r.Frequency) {
		return invalid("repeat.frequency", "choose how often it repeats")
	}
	if r.Count != 0 {
		if r.Count < 1 || r.Count > 1000 {
			return invalid("repeat.end_count", "must be between 1 and 1000")
		}
		if r.End != nil {
			return invalid("repeat.end_count", "choose an end date or a number of times, not both")
		}
		end := schedule{freq: r.Frequency, start: start}.nth(r.Count)
		r.End, r.Count = &end, 0
	}
	if r.End != nil && r.End.Before(start) {
		return invalid("repeat.end_date", "must not be before the first date")
	}
	return nil
}

// Scope says which occurrences of a recurring rule an edit or delete applies to.
type Scope int

const (
	// ScopeOnlyThis changes only the chosen rows (the only scope for rows that don't repeat).
	ScopeOnlyThis Scope = iota
	// ScopeThisAndFuture also changes the rule and its later occurrences that aren't due yet.
	// Past occurrences other than the chosen one are never changed.
	ScopeThisAndFuture
)

// ruleTemplate is what every occurrence of a rule copies.
type ruleTemplate struct {
	kind       dbq.RecurringKind
	walletID   uuid.UUID // the expense/income wallet, or the transfer's source
	toWalletID *uuid.UUID
	categoryID *uuid.UUID
	amount     int64 // what an expense/income moves, or what a transfer sends
	toAmount   *int64
	rate       *string
	note       string
	labelIDs   []uuid.UUID
}

func transactionTemplate(kind TransactionKind, in TransactionInput) ruleTemplate {
	k := dbq.RecurringKindExpense
	if kind == KindIncome {
		k = dbq.RecurringKindIncome
	}
	category := in.CategoryID
	return ruleTemplate{kind: k, walletID: in.WalletID, categoryID: &category, amount: in.Amount, note: in.Note, labelIDs: in.LabelIDs}
}

func transferTemplate(in TransferInput) ruleTemplate {
	to, toAmount, rate := in.ToWalletID, in.ToAmount, in.Rate
	return ruleTemplate{
		kind: dbq.RecurringKindTransfer, walletID: in.FromWalletID, toWalletID: &to, amount: in.FromAmount,
		toAmount: &toAmount, rate: &rate, note: in.Note, labelIDs: in.LabelIDs,
	}
}

type rule struct {
	id               uuid.UUID
	householdID      uuid.UUID
	tpl              ruleTemplate
	sched            schedule
	generatedThrough time.Time
	createdBy        *uuid.UUID
}

// occurrence points at a rule's row: an expense/income, or a transfer (both legs).
type occurrence struct {
	txID       uuid.UUID
	transferID *uuid.UUID
}

func link(ctx context.Context, q *dbq.Queries, householdID, ruleID uuid.UUID, o occurrence, on time.Time) error {
	if o.transferID != nil {
		return q.LinkTransferLegs(ctx, dbq.LinkTransferLegsParams{RuleID: &ruleID, OccurrenceDate: &on, TransferID: o.transferID, HouseholdID: householdID})
	}
	err := q.LinkTransaction(ctx, dbq.LinkTransactionParams{RuleID: &ruleID, OccurrenceDate: &on, ID: o.txID, HouseholdID: householdID})
	return dbErr(err, "occurrence on that date")
}

// lockRule loads a rule and locks it until the transaction ends, so concurrent requests and the
// top-up job never create the same occurrences twice.
func lockRule(ctx context.Context, q *dbq.Queries, householdID, id uuid.UUID) (rule, error) {
	r, err := q.LockRecurringRule(ctx, dbq.LockRecurringRuleParams{ID: id, HouseholdID: householdID})
	if err != nil {
		return rule{}, dbErr(err, "recurring rule")
	}
	labels, err := q.ListRecurringRuleLabelIDs(ctx, id)
	if err != nil {
		return rule{}, err
	}
	out := rule{
		id: r.ID, householdID: r.HouseholdID, generatedThrough: r.GeneratedThrough, createdBy: r.CreatedBy,
		sched: schedule{freq: r.Frequency, start: r.StartDate, end: r.EndDate},
		tpl: ruleTemplate{
			kind: r.Kind, walletID: r.WalletID, toWalletID: r.ToWalletID, categoryID: r.CategoryID, amount: r.AmountMinor,
			toAmount: r.ToAmountMinor, note: r.Note, labelIDs: labels,
		},
	}
	if r.Rate != "" {
		out.tpl.rate = &r.Rate
	}
	return out, nil
}

// createRule stores a new rule from its first occurrence (on start), which already exists, links
// that row and creates the following occurrences up to the horizon. A start in the past fills in
// the past occurrences too.
func (s *Service) createRule(ctx context.Context, q *dbq.Queries, a Actor, tpl ruleTemplate, rep Recurrence,
	start, today time.Time, first occurrence,
) error {
	r := rule{
		id: newID(), householdID: a.HouseholdID, tpl: tpl, createdBy: &a.UserID, generatedThrough: start,
		sched: schedule{freq: rep.Frequency, start: start, end: rep.End},
	}
	if err := q.CreateRecurringRule(ctx, dbq.CreateRecurringRuleParams{
		ID: r.id, HouseholdID: a.HouseholdID, Kind: tpl.kind, WalletID: tpl.walletID, ToWalletID: tpl.toWalletID,
		CategoryID: tpl.categoryID, AmountMinor: tpl.amount, ToAmountMinor: tpl.toAmount, Rate: tpl.rate, Note: tpl.note,
		Frequency: rep.Frequency, StartDate: start, EndDate: rep.End, GeneratedThrough: start, CreatedBy: &a.UserID,
	}); err != nil {
		return dbErr(err, "recurring rule")
	}
	if len(tpl.labelIDs) > 0 {
		if err := q.AddRecurringRuleLabels(ctx, dbq.AddRecurringRuleLabelsParams{RuleID: r.id, LabelIds: tpl.labelIDs}); err != nil {
			return err
		}
	}
	if err := link(ctx, q, a.HouseholdID, r.id, first, start); err != nil {
		return err
	}
	_, err := generate(ctx, q, r, today, maxOccurrencesPerRequest)
	return err
}

// generate creates r's occurrences after its generated_through up to the horizon, then moves
// generated_through there. A rule on an archived wallet moves on without creating anything. With
// a limit, it refuses to create more than that many occurrences. Returns how many it created.
func generate(ctx context.Context, q *dbq.Queries, r rule, today time.Time, limit int) (int, error) {
	through := r.sched.horizon(today)
	if !through.After(r.generatedThrough) {
		return 0, nil
	}
	days, more := r.sched.between(r.generatedThrough, through, limit)
	if more {
		return 0, invalid("repeat", "would create more than %d occurrences; choose a later start date or repeat less often", limit)
	}
	wallets := []uuid.UUID{r.tpl.walletID}
	if r.tpl.toWalletID != nil {
		wallets = append(wallets, *r.tpl.toWalletID)
	}
	ws, err := q.GetWalletsByIDs(ctx, dbq.GetWalletsByIDsParams{HouseholdID: r.householdID, Ids: wallets})
	if err != nil {
		return 0, err
	}
	for _, w := range ws {
		if w.ArchivedAt != nil {
			days = nil
		}
	}
	for _, d := range days {
		if err := createOccurrence(ctx, q, r, d); err != nil {
			return 0, err
		}
	}
	if err := q.SetRuleGeneratedThrough(ctx, dbq.SetRuleGeneratedThroughParams{ID: r.id, GeneratedThrough: through}); err != nil {
		return 0, err
	}
	return len(days), nil
}

func createOccurrence(ctx context.Context, q *dbq.Queries, r rule, on time.Time) error {
	ruleID, occ := r.id, on
	rl := recurringLink{ruleID: &ruleID, on: &occ}
	if r.tpl.kind == dbq.RecurringKindTransfer {
		if r.tpl.toWalletID == nil || r.tpl.toAmount == nil || r.tpl.rate == nil {
			return fmt.Errorf("recurring rule %s: transfer template is incomplete", r.id)
		}
		_, err := insertTransfer(ctx, q, r.householdID, r.createdBy, TransferInput{
			FromWalletID: r.tpl.walletID, ToWalletID: *r.tpl.toWalletID, FromAmount: r.tpl.amount, ToAmount: *r.tpl.toAmount,
			Rate: *r.tpl.rate, OccurredOn: on, Note: r.tpl.note, LabelIDs: r.tpl.labelIDs,
		}, rl)
		return err
	}
	kind := KindExpense
	if r.tpl.kind == dbq.RecurringKindIncome {
		kind = KindIncome
	}
	_, err := insertTransaction(ctx, q, r.householdID, r.createdBy, kind, TransactionInput{
		WalletID: r.tpl.walletID, CategoryID: *r.tpl.categoryID, Amount: r.tpl.amount, OccurredOn: on, Note: r.tpl.note,
		LabelIDs: r.tpl.labelIDs,
	}, rl)
	return err
}

// deleteRows deletes plain transactions and whole transfers, returning how many transaction rows
// went.
func deleteRows(ctx context.Context, q *dbq.Queries, householdID uuid.UUID, plain, transfers []uuid.UUID) (int64, error) {
	var deleted int64
	if len(plain) > 0 {
		n, err := q.DeletePlainTransactions(ctx, dbq.DeletePlainTransactionsParams{HouseholdID: householdID, Ids: plain})
		if err != nil {
			return 0, err
		}
		deleted += n
	}
	if transfers = uniqueIDs(transfers); len(transfers) > 0 {
		legs, err := q.CountTransferLegs(ctx, transfers)
		if err != nil {
			return 0, err
		}
		if _, err := q.DeleteTransfers(ctx, dbq.DeleteTransfersParams{HouseholdID: householdID, Ids: transfers}); err != nil {
			return 0, err
		}
		deleted += legs
	}
	return deleted, nil
}

// deletePending deletes a rule's occurrences scheduled on or after from that aren't due yet.
func deletePending(ctx context.Context, q *dbq.Queries, r rule, from, today time.Time) (int64, error) {
	rows, err := q.PendingOccurrences(ctx, dbq.PendingOccurrencesParams{HouseholdID: r.householdID, RuleID: &r.id, FromDate: &from, Today: today})
	if err != nil {
		return 0, err
	}
	var plain, transfers []uuid.UUID
	for _, row := range rows {
		if row.TransferID != nil {
			transfers = append(transfers, *row.TransferID)
		} else {
			plain = append(plain, row.ID)
		}
	}
	return deleteRows(ctx, q, r.householdID, plain, transfers)
}

// endBefore makes on the first date the rule no longer covers (an earlier end stays). A rule
// that would end before its first date is deleted instead; its remaining rows stay as they are.
func endBefore(ctx context.Context, q *dbq.Queries, r rule, on time.Time) error {
	end := on.AddDate(0, 0, -1)
	if end.Before(r.sched.start) {
		return q.DeleteRecurringRule(ctx, dbq.DeleteRecurringRuleParams{ID: r.id, HouseholdID: r.householdID})
	}
	if r.sched.end != nil && !r.sched.end.After(end) {
		return nil
	}
	return q.SetRuleEnd(ctx, dbq.SetRuleEndParams{ID: r.id, HouseholdID: r.householdID, EndDate: &end})
}

// dropIfEmpty deletes a rule that no longer has any occurrence and won't create more.
func dropIfEmpty(ctx context.Context, q *dbq.Queries, householdID, ruleID uuid.UUID) error {
	n, err := q.CountRuleOccurrences(ctx, &ruleID)
	if err != nil || n > 0 {
		return err
	}
	r, err := lockRule(ctx, q, householdID, ruleID)
	if errors.Is(err, ErrNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	if next, ok := r.sched.nextAfter(r.generatedThrough); ok && (r.sched.end == nil || !next.After(*r.sched.end)) {
		return nil
	}
	return q.DeleteRecurringRule(ctx, dbq.DeleteRecurringRuleParams{ID: r.id, HouseholdID: householdID})
}

// editSeries applies a "this and future ones" edit. The chosen occurrence (scheduled on occDate)
// has already been rewritten with tpl and moved to newDate. The rule takes tpl and rep, anchored
// at newDate, and its later occurrences that aren't due yet are created again from it. Past
// occurrences other than the chosen one keep their values. A rep that doesn't repeat ends the
// series at the chosen occurrence.
func (s *Service) editSeries(ctx context.Context, q *dbq.Queries, a Actor, ruleID uuid.UUID, occDate, newDate time.Time,
	tpl ruleTemplate, rep Recurrence, today time.Time, chosen occurrence,
) error {
	r, err := lockRule(ctx, q, a.HouseholdID, ruleID)
	if err != nil {
		return err
	}
	if _, err := deletePending(ctx, q, r, occDate.AddDate(0, 0, 1), today); err != nil {
		return err
	}
	if !rep.Repeats() {
		return endBefore(ctx, q, r, occDate.AddDate(0, 0, 1))
	}
	generatedThrough := maxDate(newDate, today)
	if err := q.UpdateRecurringRule(ctx, dbq.UpdateRecurringRuleParams{
		ID: r.id, HouseholdID: a.HouseholdID, WalletID: tpl.walletID, ToWalletID: tpl.toWalletID, CategoryID: tpl.categoryID,
		AmountMinor: tpl.amount, ToAmountMinor: tpl.toAmount, Rate: tpl.rate, Note: tpl.note, Frequency: rep.Frequency,
		StartDate: newDate, EndDate: rep.End, GeneratedThrough: generatedThrough,
	}); err != nil {
		return dbErr(err, "recurring rule")
	}
	if err := q.ClearRecurringRuleLabels(ctx, r.id); err != nil {
		return err
	}
	if len(tpl.labelIDs) > 0 {
		if err := q.AddRecurringRuleLabels(ctx, dbq.AddRecurringRuleLabelsParams{RuleID: r.id, LabelIds: tpl.labelIDs}); err != nil {
			return err
		}
	}
	if err := link(ctx, q, a.HouseholdID, r.id, chosen, newDate); err != nil {
		return err
	}
	r.tpl, r.generatedThrough = tpl, generatedThrough
	r.sched = schedule{freq: rep.Frequency, start: newDate, end: rep.End}
	_, err = generate(ctx, q, r, today, maxOccurrencesPerRequest)
	return err
}

// endSeries handles "delete this and future ones" for a rule: it deletes the occurrences from
// occDate on that aren't due yet and stops the rule before occDate. The chosen rows themselves
// are deleted by the caller.
func endSeries(ctx context.Context, q *dbq.Queries, householdID, ruleID uuid.UUID, occDate, today time.Time) (int64, error) {
	r, err := lockRule(ctx, q, householdID, ruleID)
	if err != nil {
		return 0, err
	}
	n, err := deletePending(ctx, q, r, occDate, today)
	if err != nil {
		return 0, err
	}
	return n, endBefore(ctx, q, r, occDate)
}

// recurrenceOf returns the schedule of a row's rule, for display.
func recurrenceOf(ctx context.Context, q *dbq.Queries, householdID uuid.UUID, ruleID *uuid.UUID) (*Recurrence, error) {
	if ruleID == nil {
		return nil, nil
	}
	r, err := q.GetRuleSchedule(ctx, dbq.GetRuleScheduleParams{ID: *ruleID, HouseholdID: householdID})
	if err != nil {
		return nil, dbErr(err, "recurring rule")
	}
	return &Recurrence{Frequency: r.Frequency, End: r.EndDate}, nil
}

// TopUpResult reports one run of the recurring top-up.
type TopUpResult struct {
	Rules       int
	Occurrences int
}

// TopUpRecurring creates the missing occurrences of every household's rules up to the horizon
// from today. It is the scheduled job behind cmd/jobs: idempotent (each rule is locked and only
// moves forward) and safe to run as often as wanted. A failing rule doesn't stop the others.
func (s *Service) TopUpRecurring(ctx context.Context, today time.Time) (TopUpResult, error) {
	rows, err := s.q.RulesToTopUp(ctx, addMonthsClamped(today, HorizonMonths))
	if err != nil {
		return TopUpResult{}, err
	}
	var res TopUpResult
	var errs []error
	for _, row := range rows {
		var created int
		err := s.inTx(ctx, func(q *dbq.Queries) error {
			r, err := lockRule(ctx, q, row.HouseholdID, row.ID)
			if err != nil {
				return err
			}
			created, err = generate(ctx, q, r, today, 0)
			return err
		})
		if err != nil {
			errs = append(errs, fmt.Errorf("rule %s: %w", row.ID, err))
			continue
		}
		res.Rules++
		res.Occurrences += created
	}
	return res, errors.Join(errs...)
}
