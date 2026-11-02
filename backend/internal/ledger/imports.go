package ledger

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/franco-roura/ledger-01/backend/internal/db/dbq"
	"github.com/franco-roura/ledger-01/backend/internal/importer"
	"github.com/franco-roura/ledger-01/backend/internal/importer/banks"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Bank statement import (see proto/ledger/v1/import.proto). An upload is parsed once and its
// rows are staged in import_batches for importTTL, so the mapping, review and commit steps are
// stateless: each one reads the batch again with the caller's mapping.

const (
	importTTL     = 24 * time.Hour
	importSample  = 30
	maxRulePatLen = 100
)

type ImportUpload struct {
	BatchID      uuid.UUID
	Columns      int
	Rows         int
	Sample       [][]string
	Mapping      importer.Mapping
	SavedMapping bool
	Info         importer.FileInfo
}

// ImportRow is a data row read with a mapping. Row.Amount is signed (negative is an expense) and,
// for a purchase with a discount merged in, net of it; Gross is the amount in the file.
type ImportRow struct {
	importer.Row
	Gross          int64
	Fingerprint    string
	Kind           TransactionKind
	CategoryID     *uuid.UUID
	CategorySource string // "column", "rule" or ""
	// Why the row looks like a duplicate (DupImported, DupSame, DupSimilar, DupFile), or "".
	DupReason string
	// The wallet transaction it matches, as it is now.
	MatchNote string
	MatchDate time.Time
	// Discounts and partial refunds of the file merged into this purchase (indexes), and their
	// total (positive).
	DiscountRows  []int
	Discount      int64
	DiscountNotes []string
	// A discount or partial refund merged into another row of the file (its index), or -1.
	MergedInto int
	// A discount or partial refund for a purchase imported earlier: importing it lowers that
	// expense.
	DiscountTarget *uuid.UUID
	TargetNote     string
	TargetDate     time.Time
	// A purchase refunded in full by a later row of the file (RefundedBy), and that refund
	// (RefundOf): both start unticked, as if neither happened. -1 when unpaired.
	RefundedBy int
	RefundOf   int
	// A full refund of a purchase already in the wallet, as it is now. Starts unticked; importing
	// it adds the refund as income.
	RefundsNote string
	RefundsDate time.Time
}

// LeftOut says the row starts unticked without being a duplicate: a refund pair.
func (r ImportRow) LeftOut() bool { return r.RefundedBy >= 0 || r.RefundOf >= 0 || r.RefundsNote != "" }

// Duplicate says the row starts unticked in the review.
func (r ImportRow) Duplicate() bool { return r.DupReason != "" }

const (
	// DupImported: this very bank row was imported before (same fingerprint), whatever the
	// transaction has been renamed or repriced to since.
	DupImported = "imported"
	// DupSame: same date, amount and note as a wallet transaction.
	DupSame = "same"
	// DupSimilar: same amount as a wallet transaction within similarDays, with another note.
	DupSimilar = "similar"
	// DupFile: same date, amount and note as an earlier row of the file.
	DupFile = "file"

	similarDays = 2
)

type ImportChoice struct {
	Index      int
	CategoryID uuid.UUID
}

type CategoryRule struct {
	ID         uuid.UUID
	Pattern    string
	CategoryID uuid.UUID
}

func mappingErr(err error) error {
	var fe *importer.FieldError
	if errors.As(err, &fe) {
		return invalid("mapping."+fe.Field, "%s", fe.Msg)
	}
	return err
}

// UploadImport parses a bank file for a wallet and stages its rows. It suggests the wallet's
// saved mapping when the file has the same header row (or, without a header, enough columns),
// else a mapping guessed from the file.
func (s *Service) UploadImport(ctx context.Context, a Actor, walletID uuid.UUID, fileName string, content []byte) (ImportUpload, error) {
	w, err := s.GetWallet(ctx, a, walletID)
	if err != nil {
		return ImportUpload{}, err
	}
	if w.Archived {
		return ImportUpload{}, invalid("wallet_id", "wallet is archived")
	}
	name := strings.TrimSpace(filepath.Base(strings.ReplaceAll(fileName, `\`, "/")))
	if utf8.RuneCountInString(name) > 200 {
		name = string([]rune(name)[:200])
	}
	g, info, err := importer.Parse(name, content)
	if err != nil {
		return ImportUpload{}, invalid("content", "%s", err.Error())
	}
	rows, err := json.Marshal(g)
	if err != nil {
		return ImportUpload{}, err
	}
	id := newID()
	if err := s.q.CreateImportBatch(ctx, dbq.CreateImportBatchParams{
		ID: id, HouseholdID: a.HouseholdID, WalletID: walletID, FileName: name, Rows: rows, CreatedBy: &a.UserID,
		ExpiresAt: time.Now().Add(importTTL),
	}); err != nil {
		return ImportUpload{}, dbErr(err, "import")
	}
	m, saved, err := s.suggestMapping(ctx, a, walletID, g)
	if err != nil {
		return ImportUpload{}, err
	}
	// A bank only when detection is sure; empty means "detect again next time" (a tiny first file
	// shouldn't pin the wallet to no bank). Choosing "Other bank" saves "generic".
	if m.Bank == "" {
		if b := importer.DetectBank(importer.Apply(g, m, w.MinorUnits), banks.All()); b.ID != importer.Generic.ID {
			m.Bank = b.ID
		}
	}
	return ImportUpload{
		BatchID: id, Columns: g.Columns(), Rows: len(g), Sample: g[:min(len(g), importSample)], Mapping: m,
		SavedMapping: saved, Info: info,
	}, nil
}

func (s *Service) suggestMapping(ctx context.Context, a Actor, walletID uuid.UUID, g importer.Grid) (importer.Mapping, bool, error) {
	p, err := s.q.GetImportProfile(ctx, dbq.GetImportProfileParams{WalletID: walletID, HouseholdID: a.HouseholdID})
	if errors.Is(err, pgx.ErrNoRows) {
		return importer.GuessMapping(g), false, nil
	}
	if err != nil {
		return importer.Mapping{}, false, err
	}
	var m importer.Mapping
	var headers []string
	if json.Unmarshal(p.Mapping, &m) == nil && json.Unmarshal(p.Headers, &headers) == nil &&
		m.Validate(g.Columns(), len(g)) == nil && (!m.HasHeader || sameHeaders(headers, m.Header(g))) {
		return m, true, nil
	}
	return importer.GuessMapping(g), false, nil
}

func sameHeaders(a, b []string) bool {
	return slices.EqualFunc(a, b, func(x, y string) bool { return strings.EqualFold(x, y) })
}

// matchImport compares rows with the wallet's transactions and with each other:
//  1. rows imported before, by fingerprint;
//  2. adjustments of a purchase, as the bank words them (discounts, refunds): partial ones
//     merged into their purchase row of the file, or marked to lower the purchase if it was
//     imported earlier; full refunds paired with their purchase, both left out;
//  3. the rest by heuristics: same date, amount and note as a transaction (or an earlier row),
//     then same amount within similarDays. Each transaction matches one row at most.
func (s *Service) matchImport(ctx context.Context, q *dbq.Queries, a Actor, w Wallet, bank importer.Bank, rows []ImportRow, from, to time.Time) error {
	fps := make([]string, 0, len(rows))
	for _, r := range rows {
		if r.Fingerprint != "" {
			fps = append(fps, r.Fingerprint)
		}
	}
	refs, err := q.ImportRefsByFingerprint(ctx, dbq.ImportRefsByFingerprintParams{HouseholdID: a.HouseholdID, WalletID: w.ID, Fingerprints: fps})
	if err != nil {
		return err
	}
	known := map[string]dbq.ImportRefsByFingerprintRow{}
	for _, r := range refs {
		known[r.Fingerprint] = r
	}
	cands, err := q.ImportCandidates(ctx, dbq.ImportCandidatesParams{
		HouseholdID: a.HouseholdID, WalletID: w.ID,
		FromDate: from.Add(-max(bank.MaxWindow(), similarDays*24*time.Hour)), ToDate: to.AddDate(0, 0, similarDays),
	})
	if err != nil {
		return err
	}
	used := map[uuid.UUID]bool{}
	match := func(r *ImportRow, reason string, c dbq.ImportCandidatesRow) {
		r.DupReason, r.MatchNote, r.MatchDate = reason, c.Note, c.OccurredOn
		used[c.ID] = true
	}

	// 1. Fingerprints.
	for i := range rows {
		r := &rows[i]
		if ref, ok := known[r.Fingerprint]; ok && r.Fingerprint != "" {
			r.DupReason, r.MatchNote, r.MatchDate = DupImported, ref.Note, ref.OccurredOn
			used[ref.TransactionID] = true
		}
	}
	open := func(i int) bool { return rows[i].Err == "" && rows[i].DupReason == "" }

	// 2. Adjustments: paired with a purchase row of the file, else with one imported earlier.
	plain := make([]importer.Row, len(rows))
	for i, r := range rows {
		plain[i] = r.Row
		plain[i].Amount = r.Gross
	}
	pairs := importer.PairAdjustments(plain, bank, func(i int) bool { return !open(i) })
	for d, p := range pairs.Merged {
		rows[d].MergedInto = p
		rows[p].DiscountRows = append(rows[p].DiscountRows, d)
		rows[p].Discount += rows[d].Gross
		rows[p].DiscountNotes = append(rows[p].DiscountNotes, rows[d].Note)
		rows[p].Amount += rows[d].Gross
	}
	for refund, p := range pairs.Refunded {
		rows[refund].RefundOf, rows[p].RefundedBy = p, refund
	}
	for i := range rows {
		r := &rows[i]
		role, merchant, window, ok := bank.Adjustment(r.Note)
		if !ok || !open(i) || r.MergedInto >= 0 || r.RefundOf >= 0 {
			continue
		}
		var best *dbq.ImportCandidatesRow
		bestFit := importer.None
		for j := range cands {
			c := &cands[j]
			if used[c.ID] || c.Kind != KindExpense {
				continue
			}
			fit := importer.Fit(role, window, c.OccurredOn, c.SignedMinor, r.Date, r.Gross)
			if fit == importer.None || (bestFit == importer.Full && fit != importer.Full) {
				continue
			}
			isPurchase, adjusted := false, false
			for _, n := range append([]string{c.Note}, c.SourceNotes...) {
				if pm, ok := bank.Purchase(n); ok && importer.SameMerchant(merchant, pm) {
					isPurchase = true
				}
				if _, _, _, ok := bank.Adjustment(n); ok {
					adjusted = true
				}
			}
			if isPurchase && !adjusted && (best == nil || fit > bestFit || c.OccurredOn.After(best.OccurredOn)) {
				best, bestFit = c, fit
			}
		}
		switch bestFit {
		case importer.Partial:
			id := best.ID
			r.DiscountTarget, r.TargetNote, r.TargetDate = &id, best.Note, best.OccurredOn
			used[id] = true
		case importer.Full:
			r.RefundsNote, r.RefundsDate = best.Note, best.OccurredOn
			used[best.ID] = true
		}
	}

	// 3. Heuristics, exact matches first so they win over near ones.
	amountMatches := func(r ImportRow, c dbq.ImportCandidatesRow) bool {
		return c.SignedMinor == r.Amount || c.SignedMinor == r.Gross
	}
	heuristic := func(i int) bool {
		return open(i) && rows[i].MergedInto < 0 && rows[i].DiscountTarget == nil && !rows[i].LeftOut()
	}
	seen := map[dupKey]bool{}
	for i := range rows {
		if !heuristic(i) {
			continue
		}
		r := &rows[i]
		for _, c := range cands {
			if !used[c.ID] && c.OccurredOn.Equal(r.Date) && amountMatches(*r, c) && normNote(c.Note) == normNote(r.Note) {
				match(r, DupSame, c)
				break
			}
		}
		k := dupKey{r.Date, r.Gross, normNote(r.Note)}
		if r.DupReason == "" && seen[k] {
			r.DupReason = DupFile
		}
		seen[k] = true
	}
	for i := range rows {
		if !heuristic(i) || rows[i].DupReason != "" {
			continue
		}
		r := &rows[i]
		var best *dbq.ImportCandidatesRow
		for j := range cands {
			c := &cands[j]
			gap := c.OccurredOn.Sub(r.Date).Abs()
			if used[c.ID] || !amountMatches(*r, *c) || gap > similarDays*24*time.Hour {
				continue
			}
			if best == nil || gap < best.OccurredOn.Sub(r.Date).Abs() {
				best = c
			}
		}
		if best != nil {
			match(r, DupSimilar, *best)
		}
	}
	return nil
}

func normNote(s string) string { return importer.NormNote(s) }

type dupKey struct {
	on     time.Time
	amount int64
	note   string
}

// readImport loads a staged batch and reads it with m: kinds, suggested categories (column, then
// rules) and duplicates (against the wallet's transactions and earlier rows of the file).
func (s *Service) readImport(ctx context.Context, q *dbq.Queries, a Actor, batchID uuid.UUID, m importer.Mapping) (
	dbq.GetImportBatchRow, importer.Grid, Wallet, []ImportRow, error,
) {
	b, err := q.GetImportBatch(ctx, dbq.GetImportBatchParams{ID: batchID, HouseholdID: a.HouseholdID})
	if errors.Is(err, pgx.ErrNoRows) {
		return b, nil, Wallet{}, nil, invalid("batch_id", "this upload has expired; upload the file again")
	}
	if err != nil {
		return b, nil, Wallet{}, nil, err
	}
	var g importer.Grid
	if err := json.Unmarshal(b.Rows, &g); err != nil {
		return b, nil, Wallet{}, nil, fmt.Errorf("import batch %s: %w", batchID, err)
	}
	if err := m.Validate(g.Columns(), len(g)); err != nil {
		return b, nil, Wallet{}, nil, mappingErr(err)
	}
	if m.Bank != "" && banks.ByID(m.Bank).ID != m.Bank {
		return b, nil, Wallet{}, nil, invalid("mapping.bank", "unknown bank %q", m.Bank)
	}
	w, err := s.GetWallet(ctx, a, b.WalletID)
	if err != nil {
		return b, nil, Wallet{}, nil, err
	}

	cats, err := q.ListCategories(ctx, dbq.ListCategoriesParams{HouseholdID: a.HouseholdID, IncludeArchived: false})
	if err != nil {
		return b, nil, Wallet{}, nil, err
	}
	byName := map[TransactionKind]map[string]uuid.UUID{KindExpense: {}, KindIncome: {}}
	kindOf := map[uuid.UUID]TransactionKind{}
	for _, c := range cats {
		k := KindExpense
		if c.Kind == CategoryIncome {
			k = KindIncome
		}
		byName[k][strings.ToLower(strings.TrimSpace(c.Name))] = c.ID
		kindOf[c.ID] = k
	}
	rules, err := q.ListImportCategoryRules(ctx, a.HouseholdID)
	if err != nil {
		return b, nil, Wallet{}, nil, err
	}
	// The longest (most specific) pattern wins.
	slices.SortStableFunc(rules, func(x, y dbq.ListImportCategoryRulesRow) int { return len(y.Pattern) - len(x.Pattern) })

	mapped := importer.Apply(g, m, w.MinorUnits)
	fingerprints := importer.Fingerprints(mapped)
	out := make([]ImportRow, len(mapped))
	var from, to time.Time
	for i, r := range mapped {
		row := ImportRow{Row: r, Gross: r.Amount, Fingerprint: fingerprints[i], MergedInto: -1, RefundedBy: -1, RefundOf: -1}
		if r.Err == "" && abs(r.Amount) > MaxAmount {
			row.Err = "the amount is too large"
		}
		if row.Err == "" {
			if utf8.RuneCountInString(row.Note) > 500 {
				row.Note = string([]rune(row.Note)[:500])
			}
			row.Kind = KindIncome
			if r.Amount < 0 {
				row.Kind = KindExpense
			}
			if id, ok := byName[row.Kind][strings.ToLower(strings.TrimSpace(r.Category))]; ok && r.Category != "" {
				row.CategoryID, row.CategorySource = &id, "column"
			} else if note := strings.ToLower(row.Note); note != "" {
				for _, rule := range rules {
					if kindOf[rule.CategoryID] == row.Kind && strings.Contains(note, strings.ToLower(rule.Pattern)) {
						id := rule.CategoryID
						row.CategoryID, row.CategorySource = &id, "rule"
						break
					}
				}
			}
			if from.IsZero() || r.Date.Before(from) {
				from = r.Date
			}
			if r.Date.After(to) {
				to = r.Date
			}
		} else {
			row.Fingerprint = ""
		}
		out[i] = row
	}
	if from.IsZero() {
		return b, g, w, out, nil
	}
	bank := banks.ByID(m.Bank)
	if m.Bank == "" {
		bank = importer.DetectBank(mapped, banks.All())
	}
	if err := s.matchImport(ctx, q, a, w, bank, out, from, to); err != nil {
		return b, g, w, nil, err
	}
	return b, g, w, out, nil
}

// PreviewImport reads a staged upload with a mapping: every data row, or the first limit ones.
// Returns the rows and how many data rows the file has.
func (s *Service) PreviewImport(ctx context.Context, a Actor, batchID uuid.UUID, m importer.Mapping, limit int) ([]ImportRow, int, error) {
	_, _, _, rows, err := s.readImport(ctx, s.q, a, batchID, m)
	if err != nil {
		return nil, 0, err
	}
	total := len(rows)
	if limit > 0 && limit < len(rows) {
		rows = rows[:limit]
	}
	return rows, total, nil
}

// CommitImport imports the chosen rows of a staged upload in one database transaction, each
// with its chosen category (which must exist and match the row's kind), then deletes the upload.
// With saveMapping the mapping becomes the wallet's suggestion for its next import.
func (s *Service) CommitImport(ctx context.Context, a Actor, batchID uuid.UUID, m importer.Mapping, choices []ImportChoice, saveMapping bool) (int64, error) {
	if len(choices) == 0 {
		return 0, invalid("rows", "choose at least one row to import")
	}
	var imported int64
	err := s.inTx(ctx, func(q *dbq.Queries) error {
		b, g, w, rows, err := s.readImport(ctx, q, a, batchID, m)
		if err != nil {
			return err
		}
		if w.Archived {
			return invalid("wallet_id", "wallet is archived")
		}
		catIDs := make([]uuid.UUID, 0, len(choices))
		for _, c := range choices {
			catIDs = append(catIDs, c.CategoryID)
		}
		cats, err := q.GetCategoriesByIDs(ctx, dbq.GetCategoriesByIDsParams{HouseholdID: a.HouseholdID, Ids: uniqueIDs(catIDs)})
		if err != nil {
			return err
		}
		kindOf := map[uuid.UUID]TransactionKind{}
		for _, c := range cats {
			kindOf[c.ID] = KindExpense
			if c.Kind == CategoryIncome {
				kindOf[c.ID] = KindIncome
			}
		}
		p := dbq.InsertImportedTransactionsParams{HouseholdID: a.HouseholdID, WalletID: w.ID, CreatedBy: &a.UserID}
		refs := importRefs{InsertImportRefsParams: dbq.InsertImportRefsParams{WalletID: w.ID, HouseholdID: a.HouseholdID}}
		chosen := map[int]bool{}
		for i, c := range choices {
			field := fmt.Sprintf("rows[%d]", i)
			if c.Index < 0 || c.Index >= len(rows) || chosen[c.Index] {
				return invalid(field+".index", "isn't a row of the file, or is chosen twice")
			}
			chosen[c.Index] = true
			r := rows[c.Index]
			if r.Err != "" {
				return invalid(field+".index", "row %d can't be imported: %s", c.Index+1, r.Err)
			}
			if r.MergedInto >= 0 {
				return invalid(field+".index", "row %d is a discount already counted in row %d", c.Index+1, r.MergedInto+1)
			}
			if r.DiscountTarget != nil {
				n, err := q.ApplyImportDiscount(ctx, dbq.ApplyImportDiscountParams{
					ID: *r.DiscountTarget, HouseholdID: a.HouseholdID, WalletID: w.ID, DiscountMinor: r.Gross,
				})
				if err != nil {
					return err
				}
				if n == 0 {
					return invalid(field+".index", "row %d: the purchase it discounts has changed; upload the file again", c.Index+1)
				}
				refs.add(r, *r.DiscountTarget)
				continue
			}
			kind, ok := kindOf[c.CategoryID]
			if !ok {
				return invalid(field+".category_id", "choose a category for row %d", c.Index+1)
			}
			if kind != r.Kind {
				return invalid(field+".category_id", "row %d is %s, so it needs an %s category", c.Index+1, r.Kind, r.Kind)
			}
			txID := newID()
			p.Ids = append(p.Ids, txID)
			refs.add(r, txID)
			for _, d := range r.DiscountRows {
				refs.add(rows[d], txID)
			}
			p.Kinds = append(p.Kinds, string(r.Kind))
			p.CategoryIds = append(p.CategoryIds, c.CategoryID)
			p.Amounts = append(p.Amounts, abs(r.Amount))
			p.Dates = append(p.Dates, r.Date)
			p.Notes = append(p.Notes, r.Note)
		}
		if imported, err = q.InsertImportedTransactions(ctx, p); err != nil {
			return dbErr(err, "imported transaction")
		}
		if len(refs.Fingerprints) > 0 {
			if err := q.InsertImportRefs(ctx, refs.InsertImportRefsParams); err != nil {
				return err
			}
		}
		if saveMapping {
			mapping, err := json.Marshal(m)
			if err != nil {
				return err
			}
			headers := m.Header(g)
			if headers == nil {
				headers = []string{}
			}
			hb, err := json.Marshal(headers)
			if err != nil {
				return err
			}
			if err := q.SaveImportProfile(ctx, dbq.SaveImportProfileParams{WalletID: w.ID, HouseholdID: a.HouseholdID, Mapping: mapping, Headers: hb}); err != nil {
				return err
			}
		}
		return q.DeleteImportBatch(ctx, dbq.DeleteImportBatchParams{ID: b.ID, HouseholdID: a.HouseholdID})
	})
	return imported, err
}

// importRefs collects the source rows of a commit's transactions.
type importRefs struct{ dbq.InsertImportRefsParams }

func (p *importRefs) add(r ImportRow, txID uuid.UUID) {
	if r.Fingerprint == "" {
		return
	}
	p.Fingerprints = append(p.Fingerprints, r.Fingerprint)
	p.TransactionIds = append(p.TransactionIds, txID)
	p.Dates = append(p.Dates, r.Date)
	p.Amounts = append(p.Amounts, r.Gross)
	p.Notes = append(p.Notes, r.Note)
}

// PurgeExpiredImports deletes staged uploads that expired before now (run by cmd/jobs).
func (s *Service) PurgeExpiredImports(ctx context.Context, now time.Time) (int64, error) {
	return s.q.DeleteExpiredImportBatches(ctx, now)
}

// --- category rules ----------------------------------------------------------------------------

func (s *Service) ListCategoryRules(ctx context.Context, a Actor) ([]CategoryRule, error) {
	rows, err := s.q.ListImportCategoryRules(ctx, a.HouseholdID)
	if err != nil {
		return nil, err
	}
	out := make([]CategoryRule, len(rows))
	for i, r := range rows {
		out[i] = CategoryRule{ID: r.ID, Pattern: r.Pattern, CategoryID: r.CategoryID}
	}
	return out, nil
}

func (s *Service) CreateCategoryRule(ctx context.Context, a Actor, pattern string, categoryID uuid.UUID) (CategoryRule, error) {
	pattern = strings.Join(strings.Fields(pattern), " ")
	if pattern == "" || utf8.RuneCountInString(pattern) > maxRulePatLen {
		return CategoryRule{}, invalid("pattern", "must be 1-%d characters", maxRulePatLen)
	}
	if _, err := s.getCategory(ctx, a, categoryID); err != nil {
		return CategoryRule{}, invalid("category_id", "category doesn't exist")
	}
	r, err := s.q.CreateImportCategoryRule(ctx, dbq.CreateImportCategoryRuleParams{
		ID: newID(), HouseholdID: a.HouseholdID, Pattern: pattern, CategoryID: categoryID,
	})
	if err != nil {
		return CategoryRule{}, dbErr(err, "a rule for that text")
	}
	return CategoryRule{ID: r.ID, Pattern: r.Pattern, CategoryID: r.CategoryID}, nil
}

func (s *Service) DeleteCategoryRule(ctx context.Context, a Actor, id uuid.UUID) error {
	n, err := s.q.DeleteImportCategoryRule(ctx, dbq.DeleteImportCategoryRuleParams{ID: id, HouseholdID: a.HouseholdID})
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}
