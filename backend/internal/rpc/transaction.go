package rpc

import (
	"context"
	"time"

	"connectrpc.com/connect"
	v1 "github.com/franco-roura/ledger-01/backend/gen/ledger/v1"
	"github.com/franco-roura/ledger-01/backend/internal/ledger"
	"github.com/franco-roura/ledger-01/backend/internal/money"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func kindToProto(k ledger.TransactionKind) v1.TransactionKind {
	switch k {
	case ledger.KindExpense:
		return v1.TransactionKind_TRANSACTION_KIND_EXPENSE
	case ledger.KindIncome:
		return v1.TransactionKind_TRANSACTION_KIND_INCOME
	case ledger.KindTransferOut:
		return v1.TransactionKind_TRANSACTION_KIND_TRANSFER_OUT
	case ledger.KindTransferIn:
		return v1.TransactionKind_TRANSACTION_KIND_TRANSFER_IN
	}
	return v1.TransactionKind_TRANSACTION_KIND_UNSPECIFIED
}

var frequencyToProto = map[ledger.Frequency]v1.RecurringFrequency{
	ledger.FreqDaily:        v1.RecurringFrequency_RECURRING_FREQUENCY_DAILY,
	ledger.FreqEvery2Days:   v1.RecurringFrequency_RECURRING_FREQUENCY_EVERY_2_DAYS,
	ledger.FreqWeekdays:     v1.RecurringFrequency_RECURRING_FREQUENCY_WEEKDAYS,
	ledger.FreqWeekly:       v1.RecurringFrequency_RECURRING_FREQUENCY_WEEKLY,
	ledger.FreqEvery2Weeks:  v1.RecurringFrequency_RECURRING_FREQUENCY_EVERY_2_WEEKS,
	ledger.FreqEvery4Weeks:  v1.RecurringFrequency_RECURRING_FREQUENCY_EVERY_4_WEEKS,
	ledger.FreqMonthly:      v1.RecurringFrequency_RECURRING_FREQUENCY_MONTHLY,
	ledger.FreqEvery2Months: v1.RecurringFrequency_RECURRING_FREQUENCY_EVERY_2_MONTHS,
	ledger.FreqEvery3Months: v1.RecurringFrequency_RECURRING_FREQUENCY_EVERY_3_MONTHS,
	ledger.FreqEvery6Months: v1.RecurringFrequency_RECURRING_FREQUENCY_EVERY_6_MONTHS,
	ledger.FreqYearly:       v1.RecurringFrequency_RECURRING_FREQUENCY_YEARLY,
}

func recurrenceToProto(r *ledger.Recurrence) *v1.Recurrence {
	if r == nil {
		return nil
	}
	out := &v1.Recurrence{Frequency: frequencyToProto[r.Frequency]}
	if r.End != nil {
		out.EndDate = money.FormatDate(*r.End)
	}
	return out
}

// recurrenceFromProto reads a repeat; nil or UNSPECIFIED means "doesn't repeat".
func recurrenceFromProto(r *v1.Recurrence) (ledger.Recurrence, error) {
	var out ledger.Recurrence
	if r == nil || r.Frequency == v1.RecurringFrequency_RECURRING_FREQUENCY_UNSPECIFIED {
		return out, nil
	}
	for k, v := range frequencyToProto {
		if v == r.Frequency {
			out.Frequency = k
		}
	}
	if out.Frequency == "" {
		return out, &ledger.InvalidError{Field: "repeat.frequency", Msg: "is unknown"}
	}
	if r.EndDate != "" {
		end, err := parseDate("repeat.end_date", r.EndDate)
		if err != nil {
			return out, err
		}
		out.End = &end
	}
	out.Count = int(r.EndCount)
	return out, nil
}

func scopeFromProto(s v1.OccurrenceScope) ledger.Scope {
	if s == v1.OccurrenceScope_OCCURRENCE_SCOPE_THIS_AND_FUTURE {
		return ledger.ScopeThisAndFuture
	}
	return ledger.ScopeOnlyThis
}

func transactionToProto(t ledger.Transaction) *v1.Transaction {
	out := &v1.Transaction{
		Id:          t.ID.String(),
		WalletId:    t.WalletID.String(),
		Kind:        kindToProto(t.Kind),
		AmountMinor: t.Amount,
		Currency:    t.Currency,
		OccurredOn:  money.FormatDate(t.OccurredOn),
		Note:        t.Note,
		LabelIds:    idStrings(t.LabelIDs),
		CreatedAt:   timestamppb.New(t.CreatedAt),
		UpdatedAt:   timestamppb.New(t.UpdatedAt),
	}
	if t.CategoryID != nil {
		out.CategoryId = t.CategoryID.String()
	}
	if t.CreatedBy != nil {
		out.CreatedBy = t.CreatedBy.String()
	}
	if t.RecurringRuleID != nil {
		out.RecurringRuleId = t.RecurringRuleID.String()
		out.Recurrence = recurrenceToProto(t.Recurrence)
	}
	if c := t.Counterpart; c != nil {
		out.Counterpart = &v1.TransferCounterpart{
			TransferId: c.TransferID.String(), WalletId: c.WalletID.String(), WalletName: c.WalletName,
			AmountMinor: c.Amount, Currency: c.Currency, Rate: c.Rate,
		}
	}
	return out
}

func transferToProto(t ledger.Transfer) *v1.Transfer {
	return &v1.Transfer{
		Id: t.ID.String(), FromWalletId: t.FromWalletID.String(), ToWalletId: t.ToWalletID.String(),
		FromAmountMinor: t.FromAmount, ToAmountMinor: t.ToAmount, Rate: t.Rate,
		OccurredOn: money.FormatDate(t.OccurredOn), Note: t.Note, LabelIds: idStrings(t.LabelIDs),
	}
}

func filterFromProto(f *v1.TransactionFilter) (ledger.Filter, error) {
	if f == nil {
		return ledger.Filter{}, &ledger.InvalidError{Field: "filter", Msg: "is required"}
	}
	wallet, err := parseID("filter.wallet_id", f.WalletId)
	if err != nil {
		return ledger.Filter{}, err
	}
	out := ledger.Filter{WalletID: wallet, NoteQuery: f.NoteQuery, MinAmount: f.MinAmountMinor, MaxAmount: f.MaxAmountMinor}
	if f.Period == nil {
		return ledger.Filter{}, &ledger.InvalidError{Field: "filter.period", Msg: "is required"}
	}
	if out.From, err = parseDate("filter.period.from", f.Period.From); err != nil {
		return ledger.Filter{}, err
	}
	if out.To, err = parseDate("filter.period.to", f.Period.To); err != nil {
		return ledger.Filter{}, err
	}
	if out.CategoryIDs, err = parseIDs("filter.category_ids", f.CategoryIds); err != nil {
		return ledger.Filter{}, err
	}
	if out.MemberIDs, err = parseIDs("filter.member_ids", f.MemberIds); err != nil {
		return ledger.Filter{}, err
	}
	if out.Today, err = parseToday(f.Today); err != nil {
		return ledger.Filter{}, err
	}
	return out, nil
}

func (s *Server) ListTransactions(ctx context.Context, req *connect.Request[v1.ListTransactionsRequest]) (*connect.Response[v1.ListTransactionsResponse], error) {
	a, err := s.actor(ctx)
	if err != nil {
		return nil, err
	}
	f, err := filterFromProto(req.Msg.Filter)
	if err != nil {
		return nil, s.fail(ctx, err)
	}
	ts, next, err := s.svc.ListTransactions(ctx, a, f, int(req.Msg.PageSize), req.Msg.PageToken)
	if err != nil {
		return nil, s.fail(ctx, err)
	}
	out := make([]*v1.Transaction, len(ts))
	for i, t := range ts {
		out[i] = transactionToProto(t)
	}
	return connect.NewResponse(&v1.ListTransactionsResponse{Transactions: out, NextPageToken: next}), nil
}

func (s *Server) ListUpcomingTransactions(ctx context.Context, req *connect.Request[v1.ListUpcomingTransactionsRequest]) (*connect.Response[v1.ListUpcomingTransactionsResponse], error) {
	a, err := s.actor(ctx)
	if err != nil {
		return nil, err
	}
	m := req.Msg
	wallet, err := parseOptID("wallet_id", &m.WalletId)
	if err != nil {
		return nil, s.fail(ctx, err)
	}
	members, err := parseIDs("member_ids", m.MemberIds)
	if err != nil {
		return nil, s.fail(ctx, err)
	}
	today, err := parseToday(m.Today)
	if err != nil {
		return nil, s.fail(ctx, err)
	}
	ts, err := s.svc.ListUpcoming(ctx, a, wallet, members, today)
	if err != nil {
		return nil, s.fail(ctx, err)
	}
	out := make([]*v1.Transaction, len(ts))
	for i, t := range ts {
		out[i] = transactionToProto(t)
	}
	return connect.NewResponse(&v1.ListUpcomingTransactionsResponse{Transactions: out}), nil
}

func (s *Server) GetPeriodSummary(ctx context.Context, req *connect.Request[v1.GetPeriodSummaryRequest]) (*connect.Response[v1.GetPeriodSummaryResponse], error) {
	a, err := s.actor(ctx)
	if err != nil {
		return nil, err
	}
	f, err := filterFromProto(req.Msg.Filter)
	if err != nil {
		return nil, s.fail(ctx, err)
	}
	sum, err := s.svc.PeriodSummary(ctx, a, f)
	if err != nil {
		return nil, s.fail(ctx, err)
	}
	return connect.NewResponse(&v1.GetPeriodSummaryResponse{
		Currency: sum.Currency, BalanceMinor: sum.Balance, ChangeMinor: sum.Change,
		ExpensesMinor: sum.Expenses, IncomeMinor: sum.Income, TransactionCount: sum.TransactionCount,
	}), nil
}

// recurringFields reads the repeat, scope and today fields shared by create and update requests.
func recurringFields(repeat *v1.Recurrence, scope v1.OccurrenceScope, today string) (ledger.Recurrence, ledger.Scope, time.Time, error) {
	rep, err := recurrenceFromProto(repeat)
	if err != nil {
		return rep, 0, time.Time{}, err
	}
	t, err := parseToday(today)
	return rep, scopeFromProto(scope), t, err
}

func transactionInput(walletID, categoryID string, amount int64, occurredOn, note string, labels []string) (ledger.TransactionInput, error) {
	var in ledger.TransactionInput
	var err error
	if walletID != "" {
		if in.WalletID, err = parseID("wallet_id", walletID); err != nil {
			return in, err
		}
	}
	if in.CategoryID, err = parseID("category_id", categoryID); err != nil {
		return in, err
	}
	if in.OccurredOn, err = parseDate("occurred_on", occurredOn); err != nil {
		return in, err
	}
	if in.LabelIDs, err = parseIDs("label_ids", labels); err != nil {
		return in, err
	}
	in.Amount, in.Note = amount, note
	return in, nil
}

func (s *Server) CreateTransaction(ctx context.Context, req *connect.Request[v1.CreateTransactionRequest]) (*connect.Response[v1.CreateTransactionResponse], error) {
	a, err := s.actor(ctx)
	if err != nil {
		return nil, err
	}
	m := req.Msg
	in, err := transactionInput(m.WalletId, m.CategoryId, m.AmountMinor, m.OccurredOn, m.Note, m.LabelIds)
	if err != nil {
		return nil, s.fail(ctx, err)
	}
	if in.Repeat, _, in.Today, err = recurringFields(m.Repeat, 0, m.Today); err != nil {
		return nil, s.fail(ctx, err)
	}
	switch m.Kind {
	case v1.TransactionKind_TRANSACTION_KIND_EXPENSE:
		in.ExpectKind = ledger.KindExpense
	case v1.TransactionKind_TRANSACTION_KIND_INCOME:
		in.ExpectKind = ledger.KindIncome
	case v1.TransactionKind_TRANSACTION_KIND_UNSPECIFIED:
	default:
		return nil, s.fail(ctx, &ledger.InvalidError{Field: "kind", Msg: "must be EXPENSE or INCOME; use CreateTransfer for transfers"})
	}
	t, err := s.svc.CreateTransaction(ctx, a, in)
	if err != nil {
		return nil, s.fail(ctx, err)
	}
	return connect.NewResponse(&v1.CreateTransactionResponse{Transaction: transactionToProto(t)}), nil
}

func (s *Server) UpdateTransaction(ctx context.Context, req *connect.Request[v1.UpdateTransactionRequest]) (*connect.Response[v1.UpdateTransactionResponse], error) {
	a, err := s.actor(ctx)
	if err != nil {
		return nil, err
	}
	m := req.Msg
	id, err := parseID("id", m.Id)
	if err != nil {
		return nil, s.fail(ctx, err)
	}
	in, err := transactionInput("", m.CategoryId, m.AmountMinor, m.OccurredOn, m.Note, m.LabelIds)
	if err != nil {
		return nil, s.fail(ctx, err)
	}
	if in.Repeat, in.Scope, in.Today, err = recurringFields(m.Repeat, m.Scope, m.Today); err != nil {
		return nil, s.fail(ctx, err)
	}
	t, err := s.svc.UpdateTransaction(ctx, a, id, in)
	if err != nil {
		return nil, s.fail(ctx, err)
	}
	return connect.NewResponse(&v1.UpdateTransactionResponse{Transaction: transactionToProto(t)}), nil
}

func (s *Server) DeleteTransactions(ctx context.Context, req *connect.Request[v1.DeleteTransactionsRequest]) (*connect.Response[v1.DeleteTransactionsResponse], error) {
	a, err := s.actor(ctx)
	if err != nil {
		return nil, err
	}
	ids, err := parseIDs("ids", req.Msg.Ids)
	if err != nil {
		return nil, s.fail(ctx, err)
	}
	today, err := parseToday(req.Msg.Today)
	if err != nil {
		return nil, s.fail(ctx, err)
	}
	n, err := s.svc.DeleteOccurrences(ctx, a, ids, scopeFromProto(req.Msg.Scope), today)
	if err != nil {
		return nil, s.fail(ctx, err)
	}
	return connect.NewResponse(&v1.DeleteTransactionsResponse{DeletedCount: n}), nil
}

func transferInput(from, to string, fromAmount, toAmount int64, rate, occurredOn, note string, labels []string) (ledger.TransferInput, error) {
	var in ledger.TransferInput
	var err error
	if in.FromWalletID, err = parseID("from_wallet_id", from); err != nil {
		return in, err
	}
	if in.ToWalletID, err = parseID("to_wallet_id", to); err != nil {
		return in, err
	}
	if in.OccurredOn, err = parseDate("occurred_on", occurredOn); err != nil {
		return in, err
	}
	if in.LabelIDs, err = parseIDs("label_ids", labels); err != nil {
		return in, err
	}
	in.FromAmount, in.ToAmount, in.Rate, in.Note = fromAmount, toAmount, rate, note
	return in, nil
}

func (s *Server) CreateTransfer(ctx context.Context, req *connect.Request[v1.CreateTransferRequest]) (*connect.Response[v1.CreateTransferResponse], error) {
	a, err := s.actor(ctx)
	if err != nil {
		return nil, err
	}
	m := req.Msg
	in, err := transferInput(m.FromWalletId, m.ToWalletId, m.FromAmountMinor, m.ToAmountMinor, m.Rate, m.OccurredOn, m.Note, m.LabelIds)
	if err != nil {
		return nil, s.fail(ctx, err)
	}
	if in.Repeat, _, in.Today, err = recurringFields(m.Repeat, 0, m.Today); err != nil {
		return nil, s.fail(ctx, err)
	}
	t, err := s.svc.CreateTransfer(ctx, a, in)
	if err != nil {
		return nil, s.fail(ctx, err)
	}
	return connect.NewResponse(&v1.CreateTransferResponse{Transfer: transferToProto(t)}), nil
}

func (s *Server) UpdateTransfer(ctx context.Context, req *connect.Request[v1.UpdateTransferRequest]) (*connect.Response[v1.UpdateTransferResponse], error) {
	a, err := s.actor(ctx)
	if err != nil {
		return nil, err
	}
	m := req.Msg
	id, err := parseID("id", m.Id)
	if err != nil {
		return nil, s.fail(ctx, err)
	}
	in, err := transferInput(m.FromWalletId, m.ToWalletId, m.FromAmountMinor, m.ToAmountMinor, m.Rate, m.OccurredOn, m.Note, m.LabelIds)
	if err != nil {
		return nil, s.fail(ctx, err)
	}
	if in.Repeat, in.Scope, in.Today, err = recurringFields(m.Repeat, m.Scope, m.Today); err != nil {
		return nil, s.fail(ctx, err)
	}
	t, err := s.svc.UpdateTransfer(ctx, a, id, in)
	if err != nil {
		return nil, s.fail(ctx, err)
	}
	return connect.NewResponse(&v1.UpdateTransferResponse{Transfer: transferToProto(t)}), nil
}
