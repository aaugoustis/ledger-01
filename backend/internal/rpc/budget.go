package rpc

import (
	"context"
	"time"

	"connectrpc.com/connect"
	v1 "github.com/franco-roura/ledger-01/backend/gen/ledger/v1"
	"github.com/franco-roura/ledger-01/backend/internal/ledger"
	"github.com/franco-roura/ledger-01/backend/internal/money"
)

var repeatToProto = map[ledger.BudgetRepeat]v1.BudgetRepeat{
	ledger.RepeatOnce:     v1.BudgetRepeat_BUDGET_REPEAT_ONCE,
	ledger.RepeatDaily:    v1.BudgetRepeat_BUDGET_REPEAT_DAILY,
	ledger.RepeatWeekly:   v1.BudgetRepeat_BUDGET_REPEAT_WEEKLY,
	ledger.RepeatBiweekly: v1.BudgetRepeat_BUDGET_REPEAT_BIWEEKLY,
	ledger.RepeatMonthly:  v1.BudgetRepeat_BUDGET_REPEAT_MONTHLY,
	ledger.RepeatYearly:   v1.BudgetRepeat_BUDGET_REPEAT_YEARLY,
}

var stateToProto = map[ledger.BudgetState]v1.BudgetState{
	ledger.BudgetUpcoming: v1.BudgetState_BUDGET_STATE_UPCOMING,
	ledger.BudgetActive:   v1.BudgetState_BUDGET_STATE_ACTIVE,
	ledger.BudgetEnded:    v1.BudgetState_BUDGET_STATE_ENDED,
}

func budgetToProto(b ledger.Budget) *v1.Budget {
	out := &v1.Budget{
		Id: b.ID.String(), Name: b.Name, AmountMinor: b.Amount, Currency: b.Currency, Repeat: repeatToProto[b.Repeat],
		StartDate: money.FormatDate(b.Start), WalletIds: idStrings(b.WalletIDs), CategoryIds: idStrings(b.CategoryIDs),
	}
	if b.End != nil {
		out.EndDate = money.FormatDate(*b.End)
	}
	return out
}

func progressToProto(p ledger.BudgetProgress) *v1.BudgetProgress {
	return &v1.BudgetProgress{
		Period: &v1.DateRange{From: money.FormatDate(p.From), To: money.FormatDate(p.To)},
		State:  stateToProto[p.State], SpentMinor: p.Spent, RemainingMinor: p.Remaining,
		DailyAllowanceMinor: p.DailyAllowance, DaysTotal: int32(p.DaysTotal), DaysLeft: int32(p.DaysLeft),
	}
}

func budgetInputFromProto(in *v1.BudgetInput) (ledger.BudgetInput, error) {
	if in == nil {
		return ledger.BudgetInput{}, &ledger.InvalidError{Field: "budget", Msg: "is required"}
	}
	out := ledger.BudgetInput{Name: in.Name, Amount: in.AmountMinor, Currency: in.Currency}
	for k, v := range repeatToProto {
		if v == in.Repeat {
			out.Repeat = k
		}
	}
	var err error
	if out.Start, err = parseDate("start_date", in.StartDate); err != nil {
		return out, err
	}
	if in.EndDate != "" {
		end, err := parseDate("end_date", in.EndDate)
		if err != nil {
			return out, err
		}
		out.End = &end
	}
	if out.WalletIDs, err = parseIDs("wallet_ids", in.WalletIds); err != nil {
		return out, err
	}
	out.CategoryIDs, err = parseIDs("category_ids", in.CategoryIds)
	return out, err
}

func (s *Server) ListBudgets(ctx context.Context, req *connect.Request[v1.ListBudgetsRequest]) (*connect.Response[v1.ListBudgetsResponse], error) {
	a, err := s.actor(ctx)
	if err != nil {
		return nil, err
	}
	today, err := parseToday(req.Msg.Today)
	if err != nil {
		return nil, s.fail(ctx, err)
	}
	bs, unconverted, err := s.svc.ListBudgets(ctx, a, today)
	if err != nil {
		return nil, s.fail(ctx, err)
	}
	out := make([]*v1.BudgetSummary, len(bs))
	for i, b := range bs {
		out[i] = &v1.BudgetSummary{Budget: budgetToProto(b.Budget), Current: progressToProto(b.Current)}
	}
	return connect.NewResponse(&v1.ListBudgetsResponse{Budgets: out, UnconvertedCurrencies: unconverted}), nil
}

func (s *Server) GetBudget(ctx context.Context, req *connect.Request[v1.GetBudgetRequest]) (*connect.Response[v1.GetBudgetResponse], error) {
	a, err := s.actor(ctx)
	if err != nil {
		return nil, err
	}
	id, err := parseID("id", req.Msg.Id)
	if err != nil {
		return nil, s.fail(ctx, err)
	}
	today, err := parseToday(req.Msg.Today)
	if err != nil {
		return nil, s.fail(ctx, err)
	}
	var periodDate *time.Time
	if req.Msg.PeriodDate != "" {
		d, err := parseDate("period_date", req.Msg.PeriodDate)
		if err != nil {
			return nil, s.fail(ctx, err)
		}
		periodDate = &d
	}
	d, err := s.svc.BudgetDetailFor(ctx, a, id, today, periodDate)
	if err != nil {
		return nil, s.fail(ctx, err)
	}
	history := make([]*v1.BudgetProgress, len(d.History))
	for i, p := range d.History {
		history[i] = progressToProto(p)
	}
	wallets := make([]*v1.WalletTotal, len(d.Wallets))
	for i, w := range d.Wallets {
		wallets[i] = &v1.WalletTotal{WalletId: w.WalletID.String(), AmountMinor: w.Amount, TransactionCount: w.Count}
	}
	txs := make([]*v1.Transaction, len(d.Transactions))
	for i, t := range d.Transactions {
		txs[i] = transactionToProto(t)
	}
	return connect.NewResponse(&v1.GetBudgetResponse{
		Budget: budgetToProto(d.Budget), Progress: progressToProto(d.Progress), History: history,
		Categories: totalsToProto(d.Categories), Wallets: wallets, Transactions: txs,
		UnconvertedCurrencies: d.Unconverted,
	}), nil
}

func (s *Server) CreateBudget(ctx context.Context, req *connect.Request[v1.CreateBudgetRequest]) (*connect.Response[v1.CreateBudgetResponse], error) {
	a, err := s.actor(ctx)
	if err != nil {
		return nil, err
	}
	in, err := budgetInputFromProto(req.Msg.Budget)
	if err != nil {
		return nil, s.fail(ctx, err)
	}
	b, err := s.svc.CreateBudget(ctx, a, in)
	if err != nil {
		return nil, s.fail(ctx, err)
	}
	return connect.NewResponse(&v1.CreateBudgetResponse{Budget: budgetToProto(b)}), nil
}

func (s *Server) UpdateBudget(ctx context.Context, req *connect.Request[v1.UpdateBudgetRequest]) (*connect.Response[v1.UpdateBudgetResponse], error) {
	a, err := s.actor(ctx)
	if err != nil {
		return nil, err
	}
	id, err := parseID("id", req.Msg.Id)
	if err != nil {
		return nil, s.fail(ctx, err)
	}
	in, err := budgetInputFromProto(req.Msg.Budget)
	if err != nil {
		return nil, s.fail(ctx, err)
	}
	b, err := s.svc.UpdateBudget(ctx, a, id, in)
	if err != nil {
		return nil, s.fail(ctx, err)
	}
	return connect.NewResponse(&v1.UpdateBudgetResponse{Budget: budgetToProto(b)}), nil
}

func (s *Server) DeleteBudget(ctx context.Context, req *connect.Request[v1.DeleteBudgetRequest]) (*connect.Response[v1.DeleteBudgetResponse], error) {
	a, err := s.actor(ctx)
	if err != nil {
		return nil, err
	}
	id, err := parseID("id", req.Msg.Id)
	if err != nil {
		return nil, s.fail(ctx, err)
	}
	if err := s.svc.DeleteBudget(ctx, a, id); err != nil {
		return nil, s.fail(ctx, err)
	}
	return connect.NewResponse(&v1.DeleteBudgetResponse{}), nil
}
