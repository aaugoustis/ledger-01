package rpc

import (
	"context"

	"connectrpc.com/connect"
	v1 "github.com/franco-roura/ledger-01/backend/gen/ledger/v1"
	"github.com/franco-roura/ledger-01/backend/internal/ledger"
	"github.com/franco-roura/ledger-01/backend/internal/money"
)

func reportFilterFromProto(f *v1.DashboardFilter) (ledger.ReportFilter, error) {
	if f == nil || f.Period == nil {
		return ledger.ReportFilter{}, &ledger.InvalidError{Field: "filter.period", Msg: "is required"}
	}
	var out ledger.ReportFilter
	var err error
	if out.From, err = parseDate("filter.period.from", f.Period.From); err != nil {
		return out, err
	}
	if out.To, err = parseDate("filter.period.to", f.Period.To); err != nil {
		return out, err
	}
	if out.WalletIDs, err = parseIDs("filter.wallet_ids", f.WalletIds); err != nil {
		return out, err
	}
	out.CategoryIDs, err = parseIDs("filter.category_ids", f.CategoryIds)
	out.NoteQuery, out.MinAmount, out.MaxAmount = f.NoteQuery, f.MinAmountMinor, f.MaxAmountMinor
	return out, err
}

func bucketFromProto(b v1.Bucket) ledger.Bucket {
	switch b {
	case v1.Bucket_BUCKET_WEEK:
		return ledger.BucketWeek
	case v1.Bucket_BUCKET_MONTH:
		return ledger.BucketMonth
	}
	return ledger.BucketDay
}

func seriesToProto(ps []ledger.SeriesPoint) []*v1.SeriesPoint {
	out := make([]*v1.SeriesPoint, len(ps))
	for i, p := range ps {
		out[i] = &v1.SeriesPoint{
			Date: money.FormatDate(p.Date), BalanceMinor: p.Balance, IncomeMinor: p.Income, ExpenseMinor: p.Expense,
		}
	}
	return out
}

func totalsToProto(ts []ledger.CategoryTotal) []*v1.CategoryTotal {
	out := make([]*v1.CategoryTotal, len(ts))
	for i, t := range ts {
		out[i] = &v1.CategoryTotal{CategoryId: t.CategoryID.String(), AmountMinor: t.Amount, TransactionCount: t.Count}
	}
	return out
}

func (s *Server) GetDashboard(ctx context.Context, req *connect.Request[v1.GetDashboardRequest]) (*connect.Response[v1.GetDashboardResponse], error) {
	a, err := s.actor(ctx)
	if err != nil {
		return nil, err
	}
	f, err := reportFilterFromProto(req.Msg.Filter)
	if err != nil {
		return nil, s.fail(ctx, err)
	}
	if f.Today, err = parseToday(req.Msg.Today); err != nil {
		return nil, s.fail(ctx, err)
	}
	f.Currency = req.Msg.Currency
	d, err := s.svc.Dashboard(ctx, a, f, bucketFromProto(req.Msg.Bucket))
	if err != nil {
		return nil, s.fail(ctx, err)
	}
	return connect.NewResponse(&v1.GetDashboardResponse{
		Currency: d.Currency, BalanceMinor: d.Balance, ChangeMinor: d.Change,
		ExpensesMinor: d.Expenses, IncomeMinor: d.Income, Series: seriesToProto(d.Series),
		ExpenseCategories: totalsToProto(d.ExpenseCategories), IncomeCategories: totalsToProto(d.IncomeCategories),
		TransfersInMinor: d.TransfersIn, TransfersOutMinor: d.TransfersOut,
		TransfersInCount: d.TransfersInCount, TransfersOutCount: d.TransfersOutCount,
		UnconvertedCurrencies: d.Unconverted,
	}), nil
}

func (s *Server) GetCategoryReport(ctx context.Context, req *connect.Request[v1.GetCategoryReportRequest]) (*connect.Response[v1.GetCategoryReportResponse], error) {
	a, err := s.actor(ctx)
	if err != nil {
		return nil, err
	}
	id, err := parseID("category_id", req.Msg.CategoryId)
	if err != nil {
		return nil, s.fail(ctx, err)
	}
	f, err := reportFilterFromProto(req.Msg.Filter)
	if err != nil {
		return nil, s.fail(ctx, err)
	}
	if f.Today, err = parseToday(req.Msg.Today); err != nil {
		return nil, s.fail(ctx, err)
	}
	f.Currency = req.Msg.Currency
	r, err := s.svc.CategoryReportFor(ctx, a, id, f, bucketFromProto(req.Msg.Bucket))
	if err != nil {
		return nil, s.fail(ctx, err)
	}
	txs := make([]*v1.Transaction, len(r.Transactions))
	for i, t := range r.Transactions {
		txs[i] = transactionToProto(t)
	}
	return connect.NewResponse(&v1.GetCategoryReportResponse{
		Currency: r.Currency, TotalMinor: r.Total, TransactionCount: r.Count, AverageMinor: r.Average,
		BusiestWeekday: int32(r.BusiestWeekday), Series: seriesToProto(r.Series), Transactions: txs,
		UnconvertedCurrencies: r.Unconverted,
	}), nil
}
