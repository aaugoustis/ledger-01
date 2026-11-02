package mcp

import (
	"context"
	"time"

	"github.com/franco-roura/ledger-01/backend/internal/ledger"
	"github.com/google/uuid"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

func readOnly(name, title, desc string) *sdk.Tool {
	f := false
	return &sdk.Tool{Name: name, Title: title, Description: desc,
		Annotations: &sdk.ToolAnnotations{Title: title, ReadOnlyHint: true, IdempotentHint: true, OpenWorldHint: &f}}
}

func (s *Server) addTools(srv *sdk.Server) {
	sdk.AddTool(srv, readOnly("get_profile", "Profile",
		"The user's name, base currency (reports are converted to it) and household."), tool(s, s.getProfile))
	sdk.AddTool(srv, readOnly("list_wallets", "Wallets",
		"Wallets (accounts) with their currency and balance as of today."), tool(s, s.listWallets))
	sdk.AddTool(srv, readOnly("list_categories", "Categories",
		"Expense and income categories with their ids, to use in other tools' category filters."), tool(s, s.listCategories))
	sdk.AddTool(srv, readOnly("list_labels", "Labels",
		"Labels (free tags on transactions) with their ids."), tool(s, s.listLabels))
	sdk.AddTool(srv, readOnly("search_transactions", "Search transactions",
		"Transactions across wallets in a date range, newest first, filtered by note text, categories, labels, "+
			"wallets, kind, amount and who entered them (member_ids from get_profile). Amounts are in each wallet's own currency. Pages of up to 200."), tool(s, s.searchTransactions))
	sdk.AddTool(srv, readOnly("spending_summary", "Spending summary",
		"Income, expenses and balance for a period in the base currency (or another), split into day/week/month "+
			"buckets and by category. Filters narrow the flows, e.g. note_query \"pizza\" with bucket month shows "+
			"monthly pizza spending."), tool(s, s.spendingSummary))
	sdk.AddTool(srv, readOnly("category_report", "Category report",
		"One category over a period: total, count, average transaction, busiest weekday, a series and its latest "+
			"transactions, in the base currency."), tool(s, s.categoryReport))
	sdk.AddTool(srv, readOnly("budget_status", "Budgets",
		"Without budget_id: every budget with its current period's spending. With budget_id: that budget's "+
			"period (the current one, or the one containing period_date), past periods and the categories spent on."), tool(s, s.budgetStatus))
	sdk.AddTool(srv, readOnly("list_upcoming", "Upcoming transactions",
		"Scheduled (recurring) transactions dated after today, soonest first."), tool(s, s.listUpcoming))
}

// --- get_profile -------------------------------------------------------------

type NoInput struct{}

type Profile struct {
	Name         string      `json:"name"`
	Email        string      `json:"email"`
	BaseCurrency string      `json:"base_currency"`
	Household    string      `json:"household"`
	Members      []MemberOut `json:"members" jsonschema:"people in the household, and former members who entered rows"`
	ServerToday  string      `json:"server_today" jsonschema:"today in UTC; prefer the user's local date"`
}

type MemberOut struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Email  string `json:"email"`
	Role   string `json:"role,omitempty" jsonschema:"owner or member; empty for former members"`
	Former bool   `json:"former,omitempty"`
}

func (s *Server) getProfile(ctx context.Context, a ledger.Actor, _ NoInput) (Profile, error) {
	u, h, err := s.svc.Me(ctx, a)
	if err != nil {
		return Profile{}, err
	}
	ms, err := s.svc.Members(ctx, a)
	if err != nil {
		return Profile{}, err
	}
	p := Profile{Name: u.Name, Email: u.Email, BaseCurrency: h.BaseCurrency, Household: h.Name, ServerToday: date(ledger.Today())}
	for _, m := range append(ms.Members, ms.Former...) {
		p.Members = append(p.Members, MemberOut{ID: m.UserID.String(), Name: m.Name, Email: m.Email, Role: m.Role, Former: m.Former})
	}
	return p, nil
}

// --- list_wallets ------------------------------------------------------------

type WalletsInput struct {
	Today           string `json:"today,omitempty" jsonschema:"the user's local date, YYYY-MM-DD"`
	IncludeArchived bool   `json:"include_archived,omitempty"`
}

type WalletOut struct {
	ID               string `json:"id"`
	Name             string `json:"name"`
	Currency         string `json:"currency"`
	Balance          Money  `json:"balance"`
	Hidden           bool   `json:"hidden"`
	Archived         bool   `json:"archived"`
	TransactionCount int64  `json:"transaction_count"`
}

type WalletsOut struct {
	Wallets []WalletOut `json:"wallets"`
}

func (s *Server) listWallets(ctx context.Context, a ledger.Actor, in WalletsInput) (WalletsOut, error) {
	today, err := parseToday(in.Today)
	if err != nil {
		return WalletsOut{}, err
	}
	m, err := s.moneyFmt(ctx)
	if err != nil {
		return WalletsOut{}, err
	}
	ws, err := s.svc.ListWallets(ctx, a, true, in.IncludeArchived, today)
	if err != nil {
		return WalletsOut{}, err
	}
	out := WalletsOut{Wallets: make([]WalletOut, len(ws))}
	for i, w := range ws {
		out.Wallets[i] = WalletOut{ID: w.ID.String(), Name: w.Name, Currency: w.Currency, Balance: m.of(w.Balance, w.Currency),
			Hidden: w.Hidden, Archived: w.Archived, TransactionCount: w.TransactionCount}
	}
	return out, nil
}

// --- list_categories / list_labels -------------------------------------------

type CategoriesInput struct {
	IncludeArchived bool `json:"include_archived,omitempty"`
}

type CategoryOut struct {
	ID               string `json:"id"`
	Name             string `json:"name"`
	Kind             string `json:"kind" jsonschema:"expense or income"`
	Archived         bool   `json:"archived"`
	TransactionCount int64  `json:"transaction_count"`
}

type CategoriesOut struct {
	Categories []CategoryOut `json:"categories"`
}

func (s *Server) listCategories(ctx context.Context, a ledger.Actor, in CategoriesInput) (CategoriesOut, error) {
	cs, err := s.svc.ListCategories(ctx, a, in.IncludeArchived)
	if err != nil {
		return CategoriesOut{}, err
	}
	out := CategoriesOut{Categories: make([]CategoryOut, len(cs))}
	for i, c := range cs {
		out.Categories[i] = CategoryOut{ID: c.ID.String(), Name: c.Name, Kind: string(c.Kind), Archived: c.Archived, TransactionCount: c.TransactionCount}
	}
	return out, nil
}

type LabelOut struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type LabelsOut struct {
	Labels []LabelOut `json:"labels"`
}

func (s *Server) listLabels(ctx context.Context, a ledger.Actor, _ NoInput) (LabelsOut, error) {
	ls, err := s.svc.ListLabels(ctx, a)
	if err != nil {
		return LabelsOut{}, err
	}
	out := LabelsOut{Labels: make([]LabelOut, len(ls))}
	for i, l := range ls {
		out.Labels[i] = LabelOut{ID: l.ID.String(), Name: l.Name}
	}
	return out, nil
}

// --- transactions --------------------------------------------------------------

// names resolves ids in transaction rows for the model.
type names struct {
	wallets, categories, labels, people map[uuid.UUID]string
}

func (s *Server) names(ctx context.Context, a ledger.Actor) (names, error) {
	n := names{wallets: map[uuid.UUID]string{}, categories: map[uuid.UUID]string{}, labels: map[uuid.UUID]string{},
		people: map[uuid.UUID]string{}}
	ws, err := s.svc.ListWallets(ctx, a, true, true, ledger.Today())
	if err != nil {
		return n, err
	}
	for _, w := range ws {
		n.wallets[w.ID] = w.Name
	}
	cs, err := s.svc.ListCategories(ctx, a, true)
	if err != nil {
		return n, err
	}
	for _, c := range cs {
		n.categories[c.ID] = c.Name
	}
	ls, err := s.svc.ListLabels(ctx, a)
	if err != nil {
		return n, err
	}
	for _, l := range ls {
		n.labels[l.ID] = l.Name
	}
	ms, err := s.svc.Members(ctx, a)
	if err != nil {
		return n, err
	}
	for _, m := range append(ms.Members, ms.Former...) {
		n.people[m.UserID] = m.Name
	}
	return n, nil
}

type TransactionOut struct {
	ID         string   `json:"id"`
	Date       string   `json:"date"`
	Kind       string   `json:"kind" jsonschema:"expense, income, transfer_out or transfer_in"`
	Amount     Money    `json:"amount"`
	Wallet     string   `json:"wallet"`
	WalletID   string   `json:"wallet_id"`
	Category   string   `json:"category,omitempty"`
	CategoryID string   `json:"category_id,omitempty"`
	Note       string   `json:"note,omitempty"`
	Labels     []string `json:"labels,omitempty"`
	// For transfers: the other wallet and the amount it received or sent.
	CounterpartWallet string `json:"counterpart_wallet,omitempty"`
	CounterpartAmount *Money `json:"counterpart_amount,omitempty"`
	Recurring         string `json:"recurring,omitempty" jsonschema:"how often it repeats, for scheduled rows"`
	CreatedBy         string `json:"created_by,omitempty" jsonschema:"who entered it, when the household has more than one person"`
}

func (n names) transaction(m moneyFmt, t ledger.Transaction) TransactionOut {
	out := TransactionOut{
		ID: t.ID.String(), Date: date(t.OccurredOn), Kind: string(t.Kind), Amount: m.of(t.Amount, t.Currency),
		Wallet: n.wallets[t.WalletID], WalletID: t.WalletID.String(), Note: t.Note,
	}
	if t.CategoryID != nil {
		out.CategoryID, out.Category = t.CategoryID.String(), n.categories[*t.CategoryID]
	}
	for _, id := range t.LabelIDs {
		out.Labels = append(out.Labels, n.labels[id])
	}
	if c := t.Counterpart; c != nil {
		amt := m.of(c.Amount, c.Currency)
		out.CounterpartWallet, out.CounterpartAmount = c.WalletName, &amt
	}
	if t.Recurrence != nil {
		out.Recurring = string(t.Recurrence.Frequency)
	}
	if t.CreatedBy != nil && len(n.people) > 1 {
		out.CreatedBy = n.people[*t.CreatedBy]
	}
	return out
}

type SearchInput struct {
	From        string   `json:"from" jsonschema:"first day, YYYY-MM-DD"`
	To          string   `json:"to" jsonschema:"last day, YYYY-MM-DD (inclusive)"`
	Today       string   `json:"today,omitempty" jsonschema:"the user's local date, YYYY-MM-DD; later rows are left out"`
	NoteQuery   string   `json:"note_query,omitempty" jsonschema:"case-insensitive text the note contains, e.g. a merchant"`
	CategoryIDs []string `json:"category_ids,omitempty"`
	LabelIDs    []string `json:"label_ids,omitempty" jsonschema:"rows with any of these labels"`
	WalletIDs   []string `json:"wallet_ids,omitempty" jsonschema:"empty means every wallet"`
	Kinds       []string `json:"kinds,omitempty" jsonschema:"any of expense, income, transfer_out, transfer_in"`
	MinAmount   *int64   `json:"min_amount_minor,omitempty" jsonschema:"smallest amount in minor units (amounts are positive)"`
	MaxAmount   *int64   `json:"max_amount_minor,omitempty" jsonschema:"largest amount in minor units"`
	MemberIDs   []string `json:"member_ids,omitempty" jsonschema:"rows entered by any of these people (ids from get_profile)"`
	Limit       int      `json:"limit,omitempty" jsonschema:"page size, 1 to 200 (default 50)"`
	PageToken   string   `json:"page_token,omitempty" jsonschema:"next_page_token from the previous page"`
}

type TransactionsOut struct {
	Transactions  []TransactionOut `json:"transactions"`
	NextPageToken string           `json:"next_page_token,omitempty"`
}

func (s *Server) searchTransactions(ctx context.Context, a ledger.Actor, in SearchInput) (TransactionsOut, error) {
	var sr ledger.Search
	var err error
	if sr.From, err = parseDate("from", in.From); err != nil {
		return TransactionsOut{}, err
	}
	if sr.To, err = parseDate("to", in.To); err != nil {
		return TransactionsOut{}, err
	}
	if sr.Today, err = parseToday(in.Today); err != nil {
		return TransactionsOut{}, err
	}
	if sr.CategoryIDs, err = parseIDs("category_ids", in.CategoryIDs); err != nil {
		return TransactionsOut{}, err
	}
	if sr.LabelIDs, err = parseIDs("label_ids", in.LabelIDs); err != nil {
		return TransactionsOut{}, err
	}
	if sr.WalletIDs, err = parseIDs("wallet_ids", in.WalletIDs); err != nil {
		return TransactionsOut{}, err
	}
	if sr.MemberIDs, err = parseIDs("member_ids", in.MemberIDs); err != nil {
		return TransactionsOut{}, err
	}
	for _, k := range in.Kinds {
		sr.Kinds = append(sr.Kinds, ledger.TransactionKind(k))
	}
	sr.NoteQuery, sr.MinAmount, sr.MaxAmount = in.NoteQuery, in.MinAmount, in.MaxAmount
	ts, next, err := s.svc.SearchTransactions(ctx, a, sr, in.Limit, in.PageToken)
	if err != nil {
		return TransactionsOut{}, err
	}
	return s.transactionsOut(ctx, a, ts, next)
}

func (s *Server) transactionsOut(ctx context.Context, a ledger.Actor, ts []ledger.Transaction, next string) (TransactionsOut, error) {
	m, err := s.moneyFmt(ctx)
	if err != nil {
		return TransactionsOut{}, err
	}
	n, err := s.names(ctx, a)
	if err != nil {
		return TransactionsOut{}, err
	}
	out := TransactionsOut{Transactions: make([]TransactionOut, len(ts)), NextPageToken: next}
	for i, t := range ts {
		out.Transactions[i] = n.transaction(m, t)
	}
	return out, nil
}

type UpcomingInput struct {
	Today    string `json:"today,omitempty" jsonschema:"the user's local date, YYYY-MM-DD"`
	WalletID string `json:"wallet_id,omitempty" jsonschema:"one wallet; empty means every wallet"`
	Limit    int    `json:"limit,omitempty" jsonschema:"at most this many, 1 to 200 (default 50)"`
}

func (s *Server) listUpcoming(ctx context.Context, a ledger.Actor, in UpcomingInput) (TransactionsOut, error) {
	today, err := parseToday(in.Today)
	if err != nil {
		return TransactionsOut{}, err
	}
	var wallet *uuid.UUID
	if in.WalletID != "" {
		ids, err := parseIDs("wallet_id", []string{in.WalletID})
		if err != nil {
			return TransactionsOut{}, err
		}
		wallet = &ids[0]
	}
	ts, err := s.svc.ListUpcoming(ctx, a, wallet, nil, today)
	if err != nil {
		return TransactionsOut{}, err
	}
	limit := in.Limit
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	if len(ts) > limit {
		ts = ts[:limit]
	}
	return s.transactionsOut(ctx, a, ts, "")
}

// --- reports -----------------------------------------------------------------

type ReportInput struct {
	From        string   `json:"from" jsonschema:"first day, YYYY-MM-DD"`
	To          string   `json:"to" jsonschema:"last day, YYYY-MM-DD (inclusive)"`
	Today       string   `json:"today,omitempty" jsonschema:"the user's local date, YYYY-MM-DD; later rows never count"`
	Bucket      string   `json:"bucket,omitempty" jsonschema:"day, week or month (default month)"`
	WalletIDs   []string `json:"wallet_ids,omitempty" jsonschema:"empty means every visible wallet"`
	CategoryIDs []string `json:"category_ids,omitempty" jsonschema:"narrows income and expenses"`
	NoteQuery   string   `json:"note_query,omitempty" jsonschema:"case-insensitive text the note contains; narrows income and expenses"`
	MinAmount   *int64   `json:"min_amount_minor,omitempty" jsonschema:"in the report currency's minor units"`
	MaxAmount   *int64   `json:"max_amount_minor,omitempty"`
	Currency    string   `json:"currency,omitempty" jsonschema:"report currency (ISO 4217); default is the base currency"`
}

func (in ReportInput) filter() (ledger.ReportFilter, ledger.Bucket, error) {
	var f ledger.ReportFilter
	var err error
	if f.From, err = parseDate("from", in.From); err != nil {
		return f, 0, err
	}
	if f.To, err = parseDate("to", in.To); err != nil {
		return f, 0, err
	}
	if f.Today, err = parseToday(in.Today); err != nil {
		return f, 0, err
	}
	if f.WalletIDs, err = parseIDs("wallet_ids", in.WalletIDs); err != nil {
		return f, 0, err
	}
	if f.CategoryIDs, err = parseIDs("category_ids", in.CategoryIDs); err != nil {
		return f, 0, err
	}
	b, err := parseBucket(in.Bucket)
	if err != nil {
		return f, 0, err
	}
	if err := checkBuckets(f.From, f.To, b); err != nil {
		return f, 0, err
	}
	f.NoteQuery, f.MinAmount, f.MaxAmount, f.Currency = in.NoteQuery, in.MinAmount, in.MaxAmount, in.Currency
	return f, b, nil
}

type PointOut struct {
	Start    string `json:"start" jsonschema:"first day of the bucket"`
	Income   Money  `json:"income"`
	Expenses Money  `json:"expenses"`
	Balance  Money  `json:"balance" jsonschema:"balance at the end of the bucket"`
}

func series(m moneyFmt, cur string, ps []ledger.SeriesPoint) []PointOut {
	out := make([]PointOut, len(ps))
	for i, p := range ps {
		out[i] = PointOut{Start: date(p.Date), Income: m.of(p.Income, cur), Expenses: m.of(p.Expense, cur), Balance: m.of(p.Balance, cur)}
	}
	return out
}

type CategoryTotalOut struct {
	CategoryID string `json:"category_id"`
	Category   string `json:"category"`
	Amount     Money  `json:"amount"`
	Count      int64  `json:"count"`
}

func categoryTotals(m moneyFmt, n names, cur string, ts []ledger.CategoryTotal) []CategoryTotalOut {
	out := make([]CategoryTotalOut, len(ts))
	for i, t := range ts {
		out[i] = CategoryTotalOut{CategoryID: t.CategoryID.String(), Category: n.categories[t.CategoryID], Amount: m.of(t.Amount, cur), Count: t.Count}
	}
	return out
}

type SummaryOut struct {
	Currency          string             `json:"currency"`
	Balance           Money              `json:"balance" jsonschema:"total balance at the end of the period"`
	Change            Money              `json:"change"`
	Income            Money              `json:"income"`
	Expenses          Money              `json:"expenses"`
	TransfersIn       Money              `json:"transfers_in"`
	TransfersOut      Money              `json:"transfers_out"`
	Series            []PointOut         `json:"series"`
	ExpenseCategories []CategoryTotalOut `json:"expense_categories"`
	IncomeCategories  []CategoryTotalOut `json:"income_categories"`
	Unconverted       []string           `json:"unconverted_currencies,omitempty" jsonschema:"currencies left out for lack of a transfer rate"`
}

func (s *Server) spendingSummary(ctx context.Context, a ledger.Actor, in ReportInput) (SummaryOut, error) {
	f, b, err := in.filter()
	if err != nil {
		return SummaryOut{}, err
	}
	d, err := s.svc.Dashboard(ctx, a, f, b)
	if err != nil {
		return SummaryOut{}, err
	}
	m, err := s.moneyFmt(ctx)
	if err != nil {
		return SummaryOut{}, err
	}
	n, err := s.names(ctx, a)
	if err != nil {
		return SummaryOut{}, err
	}
	c := d.Currency
	return SummaryOut{
		Currency: c, Balance: m.of(d.Balance, c), Change: m.of(d.Change, c), Income: m.of(d.Income, c), Expenses: m.of(d.Expenses, c),
		TransfersIn: m.of(d.TransfersIn, c), TransfersOut: m.of(d.TransfersOut, c), Series: series(m, c, d.Series),
		ExpenseCategories: categoryTotals(m, n, c, d.ExpenseCategories), IncomeCategories: categoryTotals(m, n, c, d.IncomeCategories),
		Unconverted: d.Unconverted,
	}, nil
}

type CategoryReportInput struct {
	CategoryID string `json:"category_id"`
	ReportInput
}

type CategoryReportOut struct {
	Category       string           `json:"category"`
	Currency       string           `json:"currency"`
	Total          Money            `json:"total"`
	Count          int64            `json:"count"`
	Average        Money            `json:"average"`
	BusiestWeekday string           `json:"busiest_weekday,omitempty"`
	Series         []PointOut       `json:"series"`
	Transactions   []TransactionOut `json:"latest_transactions" jsonschema:"newest first, at most 50"`
	Unconverted    []string         `json:"unconverted_currencies,omitempty"`
}

func (s *Server) categoryReport(ctx context.Context, a ledger.Actor, in CategoryReportInput) (CategoryReportOut, error) {
	ids, err := parseIDs("category_id", []string{in.CategoryID})
	if err != nil {
		return CategoryReportOut{}, err
	}
	f, b, err := in.filter()
	if err != nil {
		return CategoryReportOut{}, err
	}
	r, err := s.svc.CategoryReportFor(ctx, a, ids[0], f, b)
	if err != nil {
		return CategoryReportOut{}, err
	}
	m, err := s.moneyFmt(ctx)
	if err != nil {
		return CategoryReportOut{}, err
	}
	n, err := s.names(ctx, a)
	if err != nil {
		return CategoryReportOut{}, err
	}
	c := r.Currency
	out := CategoryReportOut{
		Category: n.categories[ids[0]], Currency: c, Total: m.of(r.Total, c), Count: r.Count, Average: m.of(r.Average, c),
		Series: series(m, c, r.Series), Unconverted: r.Unconverted,
	}
	if r.BusiestWeekday >= 0 {
		out.BusiestWeekday = time.Weekday(r.BusiestWeekday).String()
	}
	ts := r.Transactions
	if len(ts) > 50 {
		ts = ts[:50]
	}
	out.Transactions = make([]TransactionOut, len(ts))
	for i, t := range ts {
		out.Transactions[i] = n.transaction(m, t)
	}
	return out, nil
}

// --- budgets -----------------------------------------------------------------

type BudgetInput struct {
	Today      string `json:"today,omitempty" jsonschema:"the user's local date, YYYY-MM-DD"`
	BudgetID   string `json:"budget_id,omitempty" jsonschema:"one budget's detail; empty lists every budget"`
	PeriodDate string `json:"period_date,omitempty" jsonschema:"with budget_id: show the period containing this date"`
}

type PeriodOut struct {
	From           string `json:"from"`
	To             string `json:"to"`
	State          string `json:"state" jsonschema:"upcoming, active or ended"`
	Spent          Money  `json:"spent"`
	Remaining      Money  `json:"remaining"`
	DailyAllowance Money  `json:"daily_allowance" jsonschema:"what can still be spent per day"`
	DaysLeft       int    `json:"days_left"`
}

func period(m moneyFmt, cur string, p ledger.BudgetProgress) PeriodOut {
	state := map[ledger.BudgetState]string{ledger.BudgetUpcoming: "upcoming", ledger.BudgetActive: "active", ledger.BudgetEnded: "ended"}[p.State]
	return PeriodOut{From: date(p.From), To: date(p.To), State: state, Spent: m.of(p.Spent, cur), Remaining: m.of(p.Remaining, cur),
		DailyAllowance: m.of(p.DailyAllowance, cur), DaysLeft: p.DaysLeft}
}

type BudgetOut struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	Amount     Money     `json:"amount"`
	Repeat     string    `json:"repeat"`
	Start      string    `json:"start"`
	End        string    `json:"end,omitempty"`
	Categories []string  `json:"categories" jsonschema:"empty means every expense category"`
	Wallets    []string  `json:"wallets" jsonschema:"empty means every wallet"`
	Period     PeriodOut `json:"period"`
}

type BudgetsOut struct {
	Budgets []BudgetOut `json:"budgets,omitempty"`
	// Detail of one budget.
	History     []PeriodOut        `json:"history,omitempty" jsonschema:"earlier periods, newest first (at most 12)"`
	SpentOn     []CategoryTotalOut `json:"spent_on,omitempty"`
	Unconverted []string           `json:"unconverted_currencies,omitempty"`
}

func budgetOut(m moneyFmt, n names, b ledger.Budget, p ledger.BudgetProgress) BudgetOut {
	out := BudgetOut{ID: b.ID.String(), Name: b.Name, Amount: m.of(b.Amount, b.Currency), Repeat: string(b.Repeat),
		Start: date(b.Start), End: optDate(b.End), Categories: []string{}, Wallets: []string{}, Period: period(m, b.Currency, p)}
	for _, id := range b.CategoryIDs {
		out.Categories = append(out.Categories, n.categories[id])
	}
	for _, id := range b.WalletIDs {
		out.Wallets = append(out.Wallets, n.wallets[id])
	}
	return out
}

func (s *Server) budgetStatus(ctx context.Context, a ledger.Actor, in BudgetInput) (BudgetsOut, error) {
	today, err := parseToday(in.Today)
	if err != nil {
		return BudgetsOut{}, err
	}
	m, err := s.moneyFmt(ctx)
	if err != nil {
		return BudgetsOut{}, err
	}
	n, err := s.names(ctx, a)
	if err != nil {
		return BudgetsOut{}, err
	}
	if in.BudgetID == "" {
		bs, unconverted, err := s.svc.ListBudgets(ctx, a, today)
		if err != nil {
			return BudgetsOut{}, err
		}
		out := BudgetsOut{Budgets: make([]BudgetOut, len(bs)), Unconverted: unconverted}
		for i, b := range bs {
			out.Budgets[i] = budgetOut(m, n, b.Budget, b.Current)
		}
		return out, nil
	}
	ids, err := parseIDs("budget_id", []string{in.BudgetID})
	if err != nil {
		return BudgetsOut{}, err
	}
	var at *time.Time
	if in.PeriodDate != "" {
		d, err := parseDate("period_date", in.PeriodDate)
		if err != nil {
			return BudgetsOut{}, err
		}
		at = &d
	}
	d, err := s.svc.BudgetDetailFor(ctx, a, ids[0], today, at)
	if err != nil {
		return BudgetsOut{}, err
	}
	cur := d.Budget.Currency
	out := BudgetsOut{Budgets: []BudgetOut{budgetOut(m, n, d.Budget, d.Progress)}, SpentOn: categoryTotals(m, n, cur, d.Categories),
		Unconverted: d.Unconverted}
	for i, p := range d.History {
		if i == 12 {
			break
		}
		out.History = append(out.History, period(m, cur, p))
	}
	return out, nil
}
