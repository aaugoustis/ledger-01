package ledger

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"time"

	"github.com/franco-roura/ledger-01/backend/internal/db/dbq"
	"github.com/franco-roura/ledger-01/backend/internal/money"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// A backup is one household's whole ledger as JSON (gzipped on the wire). Ids only link rows
// inside the file: a restore gives every row a new one. Money is minor units, rates are decimal
// strings and dates are "YYYY-MM-DD", as everywhere else.
const (
	BackupFormat = "ledger-01-backup"
	// Version 2 added members and creator emails; version 1 files still restore.
	BackupVersion = 2
	// BackupMaxBytes caps a backup once uncompressed, so a small gzip can't expand without end.
	BackupMaxBytes = 64 << 20
)

type Backup struct {
	Format     string          `json:"format"`
	Version    int             `json:"version"`
	Schema     int64           `json:"schema"`
	ExportedAt time.Time       `json:"exported_at"`
	Household  BackupHousehold `json:"household"`
	// Who was in the household when it was exported. Informational: a restore never changes
	// membership, it only matches creators to members by email.
	Members []BackupMember `json:"members,omitempty"`

	Wallets          []BackupWallet         `json:"wallets"`
	Categories       []BackupCategory       `json:"categories"`
	HiddenCategories []BackupWalletCategory `json:"hidden_categories"`
	Labels           []BackupLabel          `json:"labels"`
	Transfers        []BackupTransfer       `json:"transfers"`
	RecurringRules   []BackupRecurringRule  `json:"recurring_rules"`
	Transactions     []BackupTransaction    `json:"transactions"`
	Budgets          []BackupBudget         `json:"budgets"`
	ImportProfiles   []BackupImportProfile  `json:"import_profiles"`
	ImportRules      []BackupImportRule     `json:"import_rules"`
	ImportRefs       []BackupImportRef      `json:"import_refs"`
}

type BackupHousehold struct {
	Name         string `json:"name"`
	BaseCurrency string `json:"base_currency"`
}

type BackupMember struct {
	Email string `json:"email"`
	Name  string `json:"name"`
	Role  string `json:"role"`
}

type BackupWallet struct {
	ID                  uuid.UUID  `json:"id"`
	Name                string     `json:"name"`
	Currency            string     `json:"currency"`
	InitialBalanceMinor int64      `json:"initial_balance_minor"`
	Hidden              bool       `json:"hidden"`
	Position            int32      `json:"position"`
	ArchivedAt          *time.Time `json:"archived_at,omitempty"`
}

type BackupCategory struct {
	ID         uuid.UUID  `json:"id"`
	Kind       string     `json:"kind"`
	Name       string     `json:"name"`
	Icon       string     `json:"icon"`
	Color      string     `json:"color"`
	Position   int32      `json:"position"`
	ArchivedAt *time.Time `json:"archived_at,omitempty"`
}

type BackupWalletCategory struct {
	WalletID   uuid.UUID `json:"wallet_id"`
	CategoryID uuid.UUID `json:"category_id"`
}

type BackupLabel struct {
	ID    uuid.UUID `json:"id"`
	Name  string    `json:"name"`
	Color string    `json:"color"`
}

type BackupTransfer struct {
	ID              uuid.UUID `json:"id"`
	FromWalletID    uuid.UUID `json:"from_wallet_id"`
	ToWalletID      uuid.UUID `json:"to_wallet_id"`
	FromAmountMinor int64     `json:"from_amount_minor"`
	ToAmountMinor   int64     `json:"to_amount_minor"`
	Rate            string    `json:"rate"`
	OccurredOn      string    `json:"occurred_on"`
	Note            string    `json:"note"`
	CreatedAt       time.Time `json:"created_at"`
	CreatedBy       string    `json:"created_by,omitempty"`
}

type BackupRecurringRule struct {
	ID               uuid.UUID   `json:"id"`
	Kind             string      `json:"kind"`
	WalletID         uuid.UUID   `json:"wallet_id"`
	ToWalletID       *uuid.UUID  `json:"to_wallet_id,omitempty"`
	CategoryID       *uuid.UUID  `json:"category_id,omitempty"`
	AmountMinor      int64       `json:"amount_minor"`
	ToAmountMinor    *int64      `json:"to_amount_minor,omitempty"`
	Rate             string      `json:"rate,omitempty"`
	Note             string      `json:"note"`
	Frequency        string      `json:"frequency"`
	StartDate        string      `json:"start_date"`
	EndDate          string      `json:"end_date,omitempty"`
	GeneratedThrough string      `json:"generated_through"`
	LabelIDs         []uuid.UUID `json:"label_ids,omitempty"`
	CreatedBy        string      `json:"created_by,omitempty"`
}

type BackupTransaction struct {
	ID              uuid.UUID   `json:"id"`
	WalletID        uuid.UUID   `json:"wallet_id"`
	Kind            string      `json:"kind"`
	CategoryID      *uuid.UUID  `json:"category_id,omitempty"`
	TransferID      *uuid.UUID  `json:"transfer_id,omitempty"`
	AmountMinor     int64       `json:"amount_minor"`
	OccurredOn      string      `json:"occurred_on"`
	Note            string      `json:"note"`
	RecurringRuleID *uuid.UUID  `json:"recurring_rule_id,omitempty"`
	OccurrenceDate  string      `json:"occurrence_date,omitempty"`
	CreatedAt       time.Time   `json:"created_at"`
	LabelIDs        []uuid.UUID `json:"label_ids,omitempty"`
	CreatedBy       string      `json:"created_by,omitempty"`
}

type BackupBudget struct {
	ID          uuid.UUID   `json:"id"`
	Name        string      `json:"name"`
	AmountMinor int64       `json:"amount_minor"`
	Currency    string      `json:"currency"`
	Repeat      string      `json:"repeat"`
	StartDate   string      `json:"start_date"`
	EndDate     string      `json:"end_date,omitempty"`
	WalletIDs   []uuid.UUID `json:"wallet_ids,omitempty"`
	CategoryIDs []uuid.UUID `json:"category_ids,omitempty"`
}

type BackupImportProfile struct {
	WalletID uuid.UUID       `json:"wallet_id"`
	Mapping  json.RawMessage `json:"mapping"`
	Headers  json.RawMessage `json:"headers"`
}

type BackupImportRule struct {
	ID         uuid.UUID `json:"id"`
	Pattern    string    `json:"pattern"`
	CategoryID uuid.UUID `json:"category_id"`
}

type BackupImportRef struct {
	WalletID      uuid.UUID `json:"wallet_id"`
	Fingerprint   string    `json:"fingerprint"`
	TransactionID uuid.UUID `json:"transaction_id"`
	OccurredOn    string    `json:"occurred_on"`
	AmountMinor   int64     `json:"amount_minor"`
	Note          string    `json:"note"`
}

// BackupCounts is what a backup holds.
type BackupCounts struct {
	Wallets, Categories, Labels, Transactions, Transfers, Budgets, RecurringRules, ImportRules int
}

func (b *Backup) Counts() BackupCounts {
	return BackupCounts{
		Wallets: len(b.Wallets), Categories: len(b.Categories), Labels: len(b.Labels),
		Transactions: len(b.Transactions), Transfers: len(b.Transfers), Budgets: len(b.Budgets),
		RecurringRules: len(b.RecurringRules), ImportRules: len(b.ImportRules),
	}
}

// schemaVersion is the newest applied goose migration.
func schemaVersion(ctx context.Context, db interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}) (int64, error) {
	var v int64
	err := db.QueryRow(ctx, `SELECT coalesce(max(version_id), 0) FROM goose_db_version WHERE is_applied`).Scan(&v)
	return v, err
}

func optDate(t *time.Time) string {
	if t == nil {
		return ""
	}
	return money.FormatDate(*t)
}

// ExportBackup reads the caller's household in one snapshot.
func (s *Service) ExportBackup(ctx context.Context, a Actor) (*Backup, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // read-only
	q := s.q.WithTx(tx)
	h := a.HouseholdID

	b := &Backup{Format: BackupFormat, Version: BackupVersion, ExportedAt: time.Now().UTC().Truncate(time.Second)}
	if b.Schema, err = schemaVersion(ctx, tx); err != nil {
		return nil, err
	}
	hh, err := q.BackupHousehold(ctx, h)
	if err != nil {
		return nil, dbErr(err, "household")
	}
	b.Household = BackupHousehold{Name: hh.Name, BaseCurrency: strings.TrimSpace(hh.BaseCurrency)}
	members, err := q.ListHouseholdMembers(ctx, h)
	if err != nil {
		return nil, err
	}
	for _, m := range members {
		b.Members = append(b.Members, BackupMember{Email: m.Email, Name: m.Name, Role: m.Role})
	}

	wallets, err := q.BackupWallets(ctx, h)
	if err != nil {
		return nil, err
	}
	for _, w := range wallets {
		b.Wallets = append(b.Wallets, BackupWallet{ID: w.ID, Name: w.Name, Currency: strings.TrimSpace(w.Currency),
			InitialBalanceMinor: w.InitialBalanceMinor, Hidden: w.Hidden, Position: w.Position, ArchivedAt: w.ArchivedAt})
	}
	cats, err := q.BackupCategories(ctx, h)
	if err != nil {
		return nil, err
	}
	for _, c := range cats {
		b.Categories = append(b.Categories, BackupCategory{ID: c.ID, Kind: string(c.Kind), Name: c.Name, Icon: c.Icon,
			Color: c.Color, Position: c.Position, ArchivedAt: c.ArchivedAt})
	}
	hidden, err := q.BackupCategoryHidden(ctx, h)
	if err != nil {
		return nil, err
	}
	for _, x := range hidden {
		b.HiddenCategories = append(b.HiddenCategories, BackupWalletCategory{WalletID: x.WalletID, CategoryID: x.CategoryID})
	}
	labels, err := q.BackupLabels(ctx, h)
	if err != nil {
		return nil, err
	}
	for _, l := range labels {
		b.Labels = append(b.Labels, BackupLabel{ID: l.ID, Name: l.Name, Color: l.Color})
	}
	transfers, err := q.BackupTransfers(ctx, h)
	if err != nil {
		return nil, err
	}
	for _, t := range transfers {
		b.Transfers = append(b.Transfers, BackupTransfer{ID: t.ID, FromWalletID: t.FromWalletID, ToWalletID: t.ToWalletID,
			FromAmountMinor: t.FromAmountMinor, ToAmountMinor: t.ToAmountMinor, Rate: t.Rate,
			OccurredOn: money.FormatDate(t.OccurredOn), Note: t.Note, CreatedAt: t.CreatedAt, CreatedBy: t.CreatedByEmail})
	}

	rules, err := q.BackupRecurringRules(ctx, h)
	if err != nil {
		return nil, err
	}
	ruleLabels, err := q.BackupRecurringRuleLabels(ctx, h)
	if err != nil {
		return nil, err
	}
	byRule := map[uuid.UUID][]uuid.UUID{}
	for _, rl := range ruleLabels {
		byRule[rl.RuleID] = append(byRule[rl.RuleID], rl.LabelID)
	}
	for _, r := range rules {
		b.RecurringRules = append(b.RecurringRules, BackupRecurringRule{ID: r.ID, Kind: string(r.Kind), WalletID: r.WalletID,
			ToWalletID: r.ToWalletID, CategoryID: r.CategoryID, AmountMinor: r.AmountMinor, ToAmountMinor: r.ToAmountMinor,
			Rate: r.Rate, Note: r.Note, Frequency: string(r.Frequency), StartDate: money.FormatDate(r.StartDate),
			EndDate: optDate(r.EndDate), GeneratedThrough: money.FormatDate(r.GeneratedThrough), LabelIDs: byRule[r.ID],
			CreatedBy: r.CreatedByEmail})
	}

	txs, err := q.BackupTransactions(ctx, h)
	if err != nil {
		return nil, err
	}
	txLabels, err := q.BackupTransactionLabels(ctx, h)
	if err != nil {
		return nil, err
	}
	byTx := map[uuid.UUID][]uuid.UUID{}
	for _, tl := range txLabels {
		byTx[tl.TransactionID] = append(byTx[tl.TransactionID], tl.LabelID)
	}
	for _, t := range txs {
		b.Transactions = append(b.Transactions, BackupTransaction{ID: t.ID, WalletID: t.WalletID, Kind: string(t.Kind),
			CategoryID: t.CategoryID, TransferID: t.TransferID, AmountMinor: t.AmountMinor,
			OccurredOn: money.FormatDate(t.OccurredOn), Note: t.Note, RecurringRuleID: t.RecurringRuleID,
			OccurrenceDate: optDate(t.OccurrenceDate), CreatedAt: t.CreatedAt, LabelIDs: byTx[t.ID],
			CreatedBy: t.CreatedByEmail})
	}

	budgets, err := q.BackupBudgets(ctx, h)
	if err != nil {
		return nil, err
	}
	bWallets, err := q.BackupBudgetWallets(ctx, h)
	if err != nil {
		return nil, err
	}
	bCats, err := q.BackupBudgetCategories(ctx, h)
	if err != nil {
		return nil, err
	}
	budgetWallets, budgetCats := map[uuid.UUID][]uuid.UUID{}, map[uuid.UUID][]uuid.UUID{}
	for _, x := range bWallets {
		budgetWallets[x.BudgetID] = append(budgetWallets[x.BudgetID], x.WalletID)
	}
	for _, x := range bCats {
		budgetCats[x.BudgetID] = append(budgetCats[x.BudgetID], x.CategoryID)
	}
	for _, bu := range budgets {
		b.Budgets = append(b.Budgets, BackupBudget{ID: bu.ID, Name: bu.Name, AmountMinor: bu.AmountMinor,
			Currency: strings.TrimSpace(bu.Currency), Repeat: string(bu.Repeat), StartDate: money.FormatDate(bu.StartDate),
			EndDate: optDate(bu.EndDate), WalletIDs: budgetWallets[bu.ID], CategoryIDs: budgetCats[bu.ID]})
	}

	profiles, err := q.BackupImportProfiles(ctx, h)
	if err != nil {
		return nil, err
	}
	for _, p := range profiles {
		b.ImportProfiles = append(b.ImportProfiles, BackupImportProfile{WalletID: p.WalletID, Mapping: p.Mapping, Headers: p.Headers})
	}
	importRules, err := q.BackupImportCategoryRules(ctx, h)
	if err != nil {
		return nil, err
	}
	for _, r := range importRules {
		b.ImportRules = append(b.ImportRules, BackupImportRule{ID: r.ID, Pattern: r.Pattern, CategoryID: r.CategoryID})
	}
	refs, err := q.BackupImportRefs(ctx, h)
	if err != nil {
		return nil, err
	}
	for _, r := range refs {
		b.ImportRefs = append(b.ImportRefs, BackupImportRef{WalletID: r.WalletID, Fingerprint: r.Fingerprint,
			TransactionID: r.TransactionID, OccurredOn: money.FormatDate(r.OccurredOn), AmountMinor: r.AmountMinor, Note: r.Note})
	}
	return b, nil
}

// EncodeBackup writes a backup as gzipped JSON.
func EncodeBackup(b *Backup) ([]byte, error) {
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	if err := json.NewEncoder(zw).Encode(b); err != nil {
		return nil, err
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// DecodeBackup reads a backup, gzipped or plain JSON, and checks its header.
func DecodeBackup(content []byte) (*Backup, error) {
	var r io.Reader = bytes.NewReader(content)
	if len(content) >= 2 && content[0] == 0x1f && content[1] == 0x8b {
		zr, err := gzip.NewReader(r)
		if err != nil {
			return nil, invalid("content", "isn't a readable gzip file")
		}
		defer zr.Close()
		r = zr
	}
	data, err := io.ReadAll(io.LimitReader(r, BackupMaxBytes+1))
	if err != nil {
		return nil, invalid("content", "isn't a readable gzip file")
	}
	if len(data) > BackupMaxBytes {
		return nil, invalid("content", "is larger than %d MB once uncompressed", BackupMaxBytes>>20)
	}
	var b Backup
	if err := json.Unmarshal(data, &b); err != nil {
		return nil, invalid("content", "isn't a Ledger/01 backup")
	}
	if b.Format != BackupFormat {
		return nil, invalid("content", "isn't a Ledger/01 backup")
	}
	if b.Version < 1 || b.Version > BackupVersion {
		return nil, invalid("content", "is backup version %d; this server reads up to %d", b.Version, BackupVersion)
	}
	return &b, nil
}

// idMap gives each id from a backup a new one and resolves references to it.
type idMap struct {
	what string
	ids  map[uuid.UUID]uuid.UUID
}

func newIDMap(what string) *idMap { return &idMap{what: what, ids: map[uuid.UUID]uuid.UUID{}} }

func (m *idMap) add(old uuid.UUID) (uuid.UUID, error) {
	if _, dup := m.ids[old]; dup {
		return uuid.Nil, invalid("content", "has %s %s twice", m.what, old)
	}
	id := newID()
	m.ids[old] = id
	return id, nil
}

func (m *idMap) get(old uuid.UUID) (uuid.UUID, error) {
	id, ok := m.ids[old]
	if !ok {
		return uuid.Nil, invalid("content", "refers to %s %s, which isn't in the file", m.what, old)
	}
	return id, nil
}

func (m *idMap) opt(old *uuid.UUID) (*uuid.UUID, error) {
	if old == nil {
		return nil, nil
	}
	id, err := m.get(*old)
	return &id, err
}

func (m *idMap) all(olds []uuid.UUID) ([]uuid.UUID, error) {
	out := make([]uuid.UUID, 0, len(olds))
	for _, o := range olds {
		id, err := m.get(o)
		if err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return uniqueIDs(out), nil
}

func backupDate(field, s string) (time.Time, error) {
	t, err := money.ParseDate(s)
	if err != nil {
		return time.Time{}, invalid("content", "%s %q isn't a date", field, s)
	}
	return t, nil
}

func backupOptDate(field, s string) (*time.Time, error) {
	if s == "" {
		return nil, nil
	}
	t, err := backupDate(field, s)
	return &t, err
}

func backupRate(field, s string) error {
	if _, err := money.ParseRate(s); err != nil {
		return invalid("content", "%s %q isn't a rate", field, s)
	}
	return nil
}

// RestoreBackup replaces the caller's household ledger with the backup's in one transaction.
// Nothing changes when the backup is rejected.
func (s *Service) RestoreBackup(ctx context.Context, a Actor, b *Backup) (BackupCounts, error) {
	if err := requireOwner(a); err != nil {
		return BackupCounts{}, err
	}
	currencies, err := s.Currencies(ctx)
	if err != nil {
		return BackupCounts{}, err
	}
	v, err := schemaVersion(ctx, s.pool)
	if err != nil {
		return BackupCounts{}, err
	}
	if b.Schema > v {
		return BackupCounts{}, invalid("content", "comes from a newer Ledger/01 (schema %d, this server has %d); update this server first", b.Schema, v)
	}
	known := map[string]bool{}
	for _, c := range currencies {
		known[c.Code] = true
	}
	currency := func(field, code string) (string, error) {
		if !known[code] {
			return "", invalid("content", "%s %q isn't a supported currency", field, code)
		}
		return code, nil
	}

	err = s.inTx(ctx, func(q *dbq.Queries) error {
		return restoreBackup(ctx, q, a, b, currency)
	})
	if err != nil {
		var pg *pgconn.PgError
		if errors.As(err, &pg) {
			// A value the checks above let through but the schema doesn't accept (an unknown
			// kind, a name too long): the whole restore is rolled back.
			return BackupCounts{}, invalid("content", "has a row the database rejects: %s", pg.Message)
		}
		return BackupCounts{}, err
	}
	return b.Counts(), nil
}

func restoreBackup(ctx context.Context, q *dbq.Queries, a Actor, b *Backup, currency func(field, code string) (string, error)) error {
	h := a.HouseholdID
	if _, err := currency("household.base_currency", b.Household.BaseCurrency); err != nil {
		return err
	}
	// Creators are matched to current members by email; anyone else becomes the restoring user.
	members, err := q.ListHouseholdMembers(ctx, h)
	if err != nil {
		return err
	}
	memberByEmail := make(map[string]*uuid.UUID, len(members))
	for _, m := range members {
		memberByEmail[strings.ToLower(m.Email)] = &m.ID
	}
	by := func(email string) *uuid.UUID {
		if id, ok := memberByEmail[strings.ToLower(email)]; ok {
			return id
		}
		return &a.UserID
	}

	for _, clear := range []func(context.Context, uuid.UUID) error{
		q.ClearTransactions, q.ClearTransfers, q.ClearRecurringRules, q.ClearBudgets, q.ClearImportCategoryRules,
		q.ClearImportBatches, q.ClearWallets, q.ClearLabels, q.ClearCategories,
	} {
		if err := clear(ctx, h); err != nil {
			return err
		}
	}
	name := strings.TrimSpace(b.Household.Name)
	if name == "" {
		hh, err := q.GetHousehold(ctx, h)
		if err != nil {
			return err
		}
		name = hh.Name
	}
	if err := q.RestoreHouseholdSettings(ctx, dbq.RestoreHouseholdSettingsParams{
		Name: name, BaseCurrency: b.Household.BaseCurrency, HouseholdID: h,
	}); err != nil {
		return err
	}

	wallets := newIDMap("wallet")
	for _, w := range b.Wallets {
		id, err := wallets.add(w.ID)
		if err != nil {
			return err
		}
		cur, err := currency("wallet currency", w.Currency)
		if err != nil {
			return err
		}
		if err := q.RestoreWallet(ctx, dbq.RestoreWalletParams{ID: id, HouseholdID: h, Name: w.Name, Currency: cur,
			InitialBalanceMinor: w.InitialBalanceMinor, Hidden: w.Hidden, Position: w.Position, ArchivedAt: w.ArchivedAt}); err != nil {
			return err
		}
	}
	cats := newIDMap("category")
	for _, c := range b.Categories {
		id, err := cats.add(c.ID)
		if err != nil {
			return err
		}
		if err := q.RestoreCategory(ctx, dbq.RestoreCategoryParams{ID: id, HouseholdID: h, Kind: dbq.CategoryKind(c.Kind),
			Name: c.Name, Icon: c.Icon, Color: c.Color, Position: c.Position, ArchivedAt: c.ArchivedAt}); err != nil {
			return err
		}
	}
	hidden := make([]dbq.RestoreCategoryHiddenParams, 0, len(b.HiddenCategories))
	for _, x := range b.HiddenCategories {
		w, err := wallets.get(x.WalletID)
		if err != nil {
			return err
		}
		c, err := cats.get(x.CategoryID)
		if err != nil {
			return err
		}
		hidden = append(hidden, dbq.RestoreCategoryHiddenParams{WalletID: w, CategoryID: c})
	}
	if _, err := q.RestoreCategoryHidden(ctx, hidden); err != nil {
		return err
	}
	labels := newIDMap("label")
	for _, l := range b.Labels {
		id, err := labels.add(l.ID)
		if err != nil {
			return err
		}
		if err := q.RestoreLabel(ctx, dbq.RestoreLabelParams{ID: id, HouseholdID: h, Name: l.Name, Color: l.Color}); err != nil {
			return err
		}
	}

	transfers := newIDMap("transfer")
	for _, t := range b.Transfers {
		id, err := transfers.add(t.ID)
		if err != nil {
			return err
		}
		from, err := wallets.get(t.FromWalletID)
		if err != nil {
			return err
		}
		to, err := wallets.get(t.ToWalletID)
		if err != nil {
			return err
		}
		on, err := backupDate("transfer date", t.OccurredOn)
		if err != nil {
			return err
		}
		if err := backupRate("transfer rate", t.Rate); err != nil {
			return err
		}
		if err := q.RestoreTransfer(ctx, dbq.RestoreTransferParams{ID: id, HouseholdID: h, FromWalletID: from, ToWalletID: to,
			FromAmountMinor: t.FromAmountMinor, ToAmountMinor: t.ToAmountMinor, Rate: t.Rate, OccurredOn: on, Note: t.Note,
			CreatedBy: by(t.CreatedBy), CreatedAt: t.CreatedAt}); err != nil {
			return err
		}
	}

	rules := newIDMap("recurring rule")
	var ruleLabels []dbq.RestoreRecurringRuleLabelsParams
	for _, r := range b.RecurringRules {
		id, err := rules.add(r.ID)
		if err != nil {
			return err
		}
		w, err := wallets.get(r.WalletID)
		if err != nil {
			return err
		}
		to, err := wallets.opt(r.ToWalletID)
		if err != nil {
			return err
		}
		cat, err := cats.opt(r.CategoryID)
		if err != nil {
			return err
		}
		start, err := backupDate("recurring start date", r.StartDate)
		if err != nil {
			return err
		}
		end, err := backupOptDate("recurring end date", r.EndDate)
		if err != nil {
			return err
		}
		through, err := backupDate("recurring generated_through", r.GeneratedThrough)
		if err != nil {
			return err
		}
		var rate *string
		if r.Rate != "" {
			if err := backupRate("recurring rate", r.Rate); err != nil {
				return err
			}
			rate = &r.Rate
		}
		if err := q.RestoreRecurringRule(ctx, dbq.RestoreRecurringRuleParams{ID: id, HouseholdID: h, Kind: dbq.RecurringKind(r.Kind),
			WalletID: w, ToWalletID: to, CategoryID: cat, AmountMinor: r.AmountMinor, ToAmountMinor: r.ToAmountMinor, Rate: rate,
			Note: r.Note, Frequency: dbq.RecurringFrequency(r.Frequency), StartDate: start, EndDate: end,
			GeneratedThrough: through, CreatedBy: by(r.CreatedBy)}); err != nil {
			return err
		}
		ls, err := labels.all(r.LabelIDs)
		if err != nil {
			return err
		}
		for _, l := range ls {
			ruleLabels = append(ruleLabels, dbq.RestoreRecurringRuleLabelsParams{RuleID: id, LabelID: l})
		}
	}
	if _, err := q.RestoreRecurringRuleLabels(ctx, ruleLabels); err != nil {
		return err
	}

	txs := newIDMap("transaction")
	rows := make([]dbq.RestoreTransactionsParams, 0, len(b.Transactions))
	var txLabels []dbq.RestoreTransactionLabelsParams
	for _, t := range b.Transactions {
		id, err := txs.add(t.ID)
		if err != nil {
			return err
		}
		w, err := wallets.get(t.WalletID)
		if err != nil {
			return err
		}
		cat, err := cats.opt(t.CategoryID)
		if err != nil {
			return err
		}
		tr, err := transfers.opt(t.TransferID)
		if err != nil {
			return err
		}
		rule, err := rules.opt(t.RecurringRuleID)
		if err != nil {
			return err
		}
		on, err := backupDate("transaction date", t.OccurredOn)
		if err != nil {
			return err
		}
		occ, err := backupOptDate("occurrence date", t.OccurrenceDate)
		if err != nil {
			return err
		}
		rows = append(rows, dbq.RestoreTransactionsParams{ID: id, HouseholdID: h, WalletID: w, Kind: dbq.TransactionKind(t.Kind),
			CategoryID: cat, TransferID: tr, AmountMinor: t.AmountMinor, OccurredOn: on, Note: t.Note, RecurringRuleID: rule,
			OccurrenceDate: occ, CreatedBy: by(t.CreatedBy), CreatedAt: t.CreatedAt})
		ls, err := labels.all(t.LabelIDs)
		if err != nil {
			return err
		}
		for _, l := range ls {
			txLabels = append(txLabels, dbq.RestoreTransactionLabelsParams{TransactionID: id, LabelID: l})
		}
	}
	if _, err := q.RestoreTransactions(ctx, rows); err != nil {
		return err
	}
	if _, err := q.RestoreTransactionLabels(ctx, txLabels); err != nil {
		return err
	}

	var budgetWallets []dbq.RestoreBudgetWalletsParams
	var budgetCats []dbq.RestoreBudgetCategoriesParams
	budgets := newIDMap("budget")
	for _, bu := range b.Budgets {
		id, err := budgets.add(bu.ID)
		if err != nil {
			return err
		}
		cur, err := currency("budget currency", bu.Currency)
		if err != nil {
			return err
		}
		start, err := backupDate("budget start date", bu.StartDate)
		if err != nil {
			return err
		}
		end, err := backupOptDate("budget end date", bu.EndDate)
		if err != nil {
			return err
		}
		if err := q.RestoreBudget(ctx, dbq.RestoreBudgetParams{ID: id, HouseholdID: h, Name: bu.Name, AmountMinor: bu.AmountMinor,
			Currency: cur, Repeat: dbq.BudgetRepeat(bu.Repeat), StartDate: start, EndDate: end}); err != nil {
			return err
		}
		ws, err := wallets.all(bu.WalletIDs)
		if err != nil {
			return err
		}
		for _, w := range ws {
			budgetWallets = append(budgetWallets, dbq.RestoreBudgetWalletsParams{BudgetID: id, WalletID: w})
		}
		cs, err := cats.all(bu.CategoryIDs)
		if err != nil {
			return err
		}
		for _, c := range cs {
			budgetCats = append(budgetCats, dbq.RestoreBudgetCategoriesParams{BudgetID: id, CategoryID: c})
		}
	}
	if _, err := q.RestoreBudgetWallets(ctx, budgetWallets); err != nil {
		return err
	}
	if _, err := q.RestoreBudgetCategories(ctx, budgetCats); err != nil {
		return err
	}

	for _, p := range b.ImportProfiles {
		w, err := wallets.get(p.WalletID)
		if err != nil {
			return err
		}
		headers := p.Headers
		if len(headers) == 0 {
			headers = json.RawMessage("[]")
		}
		if err := q.RestoreImportProfile(ctx, dbq.RestoreImportProfileParams{WalletID: w, HouseholdID: h, Mapping: p.Mapping,
			Headers: headers}); err != nil {
			return err
		}
	}
	seen := map[uuid.UUID]bool{}
	for _, r := range b.ImportRules {
		if seen[r.ID] {
			return invalid("content", "has import rule %s twice", r.ID)
		}
		seen[r.ID] = true
		c, err := cats.get(r.CategoryID)
		if err != nil {
			return err
		}
		if err := q.RestoreImportCategoryRule(ctx, dbq.RestoreImportCategoryRuleParams{ID: newID(), HouseholdID: h,
			Pattern: r.Pattern, CategoryID: c}); err != nil {
			return err
		}
	}
	refs := make([]dbq.RestoreImportRefsParams, 0, len(b.ImportRefs))
	for _, r := range b.ImportRefs {
		w, err := wallets.get(r.WalletID)
		if err != nil {
			return err
		}
		t, err := txs.get(r.TransactionID)
		if err != nil {
			return err
		}
		on, err := backupDate("import ref date", r.OccurredOn)
		if err != nil {
			return err
		}
		refs = append(refs, dbq.RestoreImportRefsParams{WalletID: w, HouseholdID: h, Fingerprint: r.Fingerprint,
			TransactionID: t, OccurredOn: on, AmountMinor: r.AmountMinor, Note: r.Note})
	}
	_, err = q.RestoreImportRefs(ctx, refs)
	return err
}

// --- CSV export ----------------------------------------------------------------------------------

type CsvFilter struct {
	Today     time.Time
	From, To  *time.Time
	WalletIDs []uuid.UUID
}

var csvKinds = map[dbq.TransactionKind]string{
	dbq.TransactionKindExpense: "expense", dbq.TransactionKindIncome: "income",
	dbq.TransactionKindTransferOut: "transfer out", dbq.TransactionKindTransferIn: "transfer in",
}

// ExportTransactionsCsv writes the household's transactions up to today, oldest first. Amounts
// are signed decimals in the wallet's currency (money going out is negative).
func (s *Service) ExportTransactionsCsv(ctx context.Context, a Actor, f CsvFilter) ([]byte, int, error) {
	currencies, err := s.Currencies(ctx)
	if err != nil {
		return nil, 0, err
	}
	units := map[string]int{}
	for _, c := range currencies {
		units[c.Code] = c.MinorUnits
	}
	ids := f.WalletIDs
	if ids == nil {
		ids = []uuid.UUID{}
	}
	rows, err := s.q.ExportTransactionRows(ctx, dbq.ExportTransactionRowsParams{HouseholdID: a.HouseholdID,
		Today: orToday(f.Today), FromDate: f.From, ToDate: f.To, WalletIds: ids})
	if err != nil {
		return nil, 0, err
	}
	var buf bytes.Buffer
	buf.WriteString("\xEF\xBB\xBF") // Excel reads the file as UTF-8 only with a BOM.
	w := csv.NewWriter(&buf)
	_ = w.Write([]string{"Date", "Wallet", "Type", "Category", "Amount", "Currency", "Labels", "Note", "Transfer wallet"})
	for _, r := range rows {
		cur := strings.TrimSpace(r.Currency)
		amount := r.AmountMinor
		if r.Kind == dbq.TransactionKindExpense || r.Kind == dbq.TransactionKindTransferOut {
			amount = -amount
		}
		_ = w.Write([]string{money.FormatDate(r.OccurredOn), r.Wallet, csvKinds[r.Kind], r.Category,
			money.FormatMinor(amount, units[cur]), cur, r.Labels, r.Note, r.Counterpart})
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return nil, 0, err
	}
	return buf.Bytes(), len(rows), nil
}
