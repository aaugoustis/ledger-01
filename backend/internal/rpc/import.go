package rpc

import (
	"context"

	"connectrpc.com/connect"
	v1 "github.com/franco-roura/ledger-01/backend/gen/ledger/v1"
	"github.com/franco-roura/ledger-01/backend/internal/importer"
	"github.com/franco-roura/ledger-01/backend/internal/importer/banks"
	"github.com/franco-roura/ledger-01/backend/internal/ledger"
	"github.com/franco-roura/ledger-01/backend/internal/money"
)

// ImportReadMaxBytes caps an ImportService request: a 4 MB file is ~5.6 MB as base64 in JSON.
const ImportReadMaxBytes = 6 << 20

func optColumn(c *int32) int {
	if c == nil {
		return -1
	}
	return int(*c)
}

func protoColumn(c int) *int32 {
	if c < 0 {
		return nil
	}
	v := int32(c)
	return &v
}

func mappingFromProto(m *v1.ImportMapping) (importer.Mapping, error) {
	if m == nil {
		return importer.Mapping{}, &ledger.InvalidError{Field: "mapping", Msg: "is required"}
	}
	out := importer.Mapping{
		Skip: int(m.SkipRows), HasHeader: m.HasHeader, Date: int(m.DateColumn), DateFormat: m.DateFormat, Amount: optColumn(m.AmountColumn),
		Debit: optColumn(m.DebitColumn), Credit: optColumn(m.CreditColumn), Note: optColumn(m.NoteColumn),
		Category: optColumn(m.CategoryColumn), Bank: m.Bank,
	}
	switch m.Sign {
	case v1.ImportSign_IMPORT_SIGN_NEGATIVE_IS_EXPENSE:
		out.Sign = importer.NegativeIsExpense
	case v1.ImportSign_IMPORT_SIGN_POSITIVE_IS_EXPENSE:
		out.Sign = importer.PositiveIsExpense
	}
	switch m.DecimalSeparator {
	case v1.DecimalSeparator_DECIMAL_SEPARATOR_DOT:
		out.Decimal = '.'
	case v1.DecimalSeparator_DECIMAL_SEPARATOR_COMMA:
		out.Decimal = ','
	}
	return out, nil
}

func mappingToProto(m importer.Mapping) *v1.ImportMapping {
	out := &v1.ImportMapping{
		SkipRows: int32(m.Skip), HasHeader: m.HasHeader, DateColumn: int32(max(m.Date, 0)), DateFormat: m.DateFormat, AmountColumn: protoColumn(m.Amount),
		DebitColumn: protoColumn(m.Debit), CreditColumn: protoColumn(m.Credit), NoteColumn: protoColumn(m.Note),
		CategoryColumn: protoColumn(m.Category), Bank: m.Bank,
	}
	switch m.Sign {
	case importer.NegativeIsExpense:
		out.Sign = v1.ImportSign_IMPORT_SIGN_NEGATIVE_IS_EXPENSE
	case importer.PositiveIsExpense:
		out.Sign = v1.ImportSign_IMPORT_SIGN_POSITIVE_IS_EXPENSE
	}
	switch m.Decimal {
	case '.':
		out.DecimalSeparator = v1.DecimalSeparator_DECIMAL_SEPARATOR_DOT
	case ',':
		out.DecimalSeparator = v1.DecimalSeparator_DECIMAL_SEPARATOR_COMMA
	}
	return out
}

func importRowToProto(r ledger.ImportRow) *v1.ImportRow {
	out := &v1.ImportRow{Index: int32(r.Index), Cells: r.Cells, Error: r.Err}
	if r.Err != "" {
		return out
	}
	out.OccurredOn = money.FormatDate(r.Date)
	out.AmountMinor = r.Amount
	if out.AmountMinor < 0 {
		out.AmountMinor = -out.AmountMinor
	}
	out.Kind = kindToProto(r.Kind)
	out.Note = r.Note
	if r.CategoryID != nil {
		out.CategoryId = r.CategoryID.String()
	}
	out.CategorySource = r.CategorySource
	out.Duplicate = r.Duplicate()
	out.DuplicateReason = map[string]v1.ImportDuplicate{
		ledger.DupImported: v1.ImportDuplicate_IMPORT_DUPLICATE_IMPORTED,
		ledger.DupSame:     v1.ImportDuplicate_IMPORT_DUPLICATE_SAME,
		ledger.DupSimilar:  v1.ImportDuplicate_IMPORT_DUPLICATE_SIMILAR,
		ledger.DupFile:     v1.ImportDuplicate_IMPORT_DUPLICATE_FILE,
	}[r.DupReason]
	if !r.MatchDate.IsZero() {
		out.MatchNote, out.MatchDate = r.MatchNote, money.FormatDate(r.MatchDate)
	}
	out.DiscountMinor, out.DiscountNotes = r.Discount, r.DiscountNotes
	if r.MergedInto >= 0 {
		v := int32(r.MergedInto)
		out.MergedInto = &v
	}
	if r.DiscountTarget != nil {
		out.DiscountsTransactionId = r.DiscountTarget.String()
		out.DiscountsNote, out.DiscountsDate = r.TargetNote, money.FormatDate(r.TargetDate)
	}
	if r.RefundedBy >= 0 {
		v := int32(r.RefundedBy)
		out.RefundedBy = &v
	}
	if r.RefundOf >= 0 {
		v := int32(r.RefundOf)
		out.RefundOf = &v
	}
	if r.RefundsNote != "" {
		out.RefundsNote, out.RefundsDate = r.RefundsNote, money.FormatDate(r.RefundsDate)
	}
	return out
}

func (s *Server) UploadImportFile(ctx context.Context, req *connect.Request[v1.UploadImportFileRequest]) (*connect.Response[v1.UploadImportFileResponse], error) {
	a, err := s.actor(ctx)
	if err != nil {
		return nil, err
	}
	wallet, err := parseID("wallet_id", req.Msg.WalletId)
	if err != nil {
		return nil, s.fail(ctx, err)
	}
	u, err := s.svc.UploadImport(ctx, a, wallet, req.Msg.FileName, req.Msg.Content)
	if err != nil {
		return nil, s.fail(ctx, err)
	}
	sample := make([]*v1.ImportCells, len(u.Sample))
	for i, row := range u.Sample {
		sample[i] = &v1.ImportCells{Cells: row}
	}
	return connect.NewResponse(&v1.UploadImportFileResponse{
		BatchId: u.BatchID.String(), ColumnCount: int32(u.Columns), RowCount: int32(u.Rows), Sample: sample,
		Mapping: mappingToProto(u.Mapping), SavedMapping: u.SavedMapping, Format: u.Info.Format,
		Encoding: u.Info.Encoding, Delimiter: u.Info.Delimiter, DateFormats: importer.DateFormats,
		Banks: banksToProto(),
	}), nil
}

func (s *Server) PreviewImport(ctx context.Context, req *connect.Request[v1.PreviewImportRequest]) (*connect.Response[v1.PreviewImportResponse], error) {
	a, err := s.actor(ctx)
	if err != nil {
		return nil, err
	}
	batch, err := parseID("batch_id", req.Msg.BatchId)
	if err != nil {
		return nil, s.fail(ctx, err)
	}
	m, err := mappingFromProto(req.Msg.Mapping)
	if err != nil {
		return nil, s.fail(ctx, err)
	}
	rows, total, err := s.svc.PreviewImport(ctx, a, batch, m, int(req.Msg.Limit))
	if err != nil {
		return nil, s.fail(ctx, err)
	}
	out := make([]*v1.ImportRow, len(rows))
	for i, r := range rows {
		out[i] = importRowToProto(r)
	}
	return connect.NewResponse(&v1.PreviewImportResponse{Rows: out, Total: int32(total)}), nil
}

func (s *Server) CommitImport(ctx context.Context, req *connect.Request[v1.CommitImportRequest]) (*connect.Response[v1.CommitImportResponse], error) {
	a, err := s.actor(ctx)
	if err != nil {
		return nil, err
	}
	m := req.Msg
	batch, err := parseID("batch_id", m.BatchId)
	if err != nil {
		return nil, s.fail(ctx, err)
	}
	mapping, err := mappingFromProto(m.Mapping)
	if err != nil {
		return nil, s.fail(ctx, err)
	}
	choices := make([]ledger.ImportChoice, len(m.Rows))
	for i, r := range m.Rows {
		choices[i].Index = int(r.Index)
		// Empty only for discounts of earlier purchases; the service checks the rest.
		if r.CategoryId == "" {
			continue
		}
		if choices[i].CategoryID, err = parseID("rows.category_id", r.CategoryId); err != nil {
			return nil, s.fail(ctx, err)
		}
	}
	n, err := s.svc.CommitImport(ctx, a, batch, mapping, choices, m.SaveMapping)
	if err != nil {
		return nil, s.fail(ctx, err)
	}
	return connect.NewResponse(&v1.CommitImportResponse{ImportedCount: n}), nil
}

func ruleToProto(r ledger.CategoryRule) *v1.CategoryRule {
	return &v1.CategoryRule{Id: r.ID.String(), Pattern: r.Pattern, CategoryId: r.CategoryID.String()}
}

func (s *Server) ListCategoryRules(ctx context.Context, _ *connect.Request[v1.ListCategoryRulesRequest]) (*connect.Response[v1.ListCategoryRulesResponse], error) {
	a, err := s.actor(ctx)
	if err != nil {
		return nil, err
	}
	rules, err := s.svc.ListCategoryRules(ctx, a)
	if err != nil {
		return nil, s.fail(ctx, err)
	}
	out := make([]*v1.CategoryRule, len(rules))
	for i, r := range rules {
		out[i] = ruleToProto(r)
	}
	return connect.NewResponse(&v1.ListCategoryRulesResponse{Rules: out}), nil
}

func (s *Server) CreateCategoryRule(ctx context.Context, req *connect.Request[v1.CreateCategoryRuleRequest]) (*connect.Response[v1.CreateCategoryRuleResponse], error) {
	a, err := s.actor(ctx)
	if err != nil {
		return nil, err
	}
	cat, err := parseID("category_id", req.Msg.CategoryId)
	if err != nil {
		return nil, s.fail(ctx, err)
	}
	r, err := s.svc.CreateCategoryRule(ctx, a, req.Msg.Pattern, cat)
	if err != nil {
		return nil, s.fail(ctx, err)
	}
	return connect.NewResponse(&v1.CreateCategoryRuleResponse{Rule: ruleToProto(r)}), nil
}

func (s *Server) DeleteCategoryRule(ctx context.Context, req *connect.Request[v1.DeleteCategoryRuleRequest]) (*connect.Response[v1.DeleteCategoryRuleResponse], error) {
	a, err := s.actor(ctx)
	if err != nil {
		return nil, err
	}
	id, err := parseID("id", req.Msg.Id)
	if err != nil {
		return nil, s.fail(ctx, err)
	}
	if err := s.svc.DeleteCategoryRule(ctx, a, id); err != nil {
		return nil, s.fail(ctx, err)
	}
	return connect.NewResponse(&v1.DeleteCategoryRuleResponse{}), nil
}

func banksToProto() []*v1.ImportBank {
	all := banks.All()
	out := make([]*v1.ImportBank, len(all))
	for i, b := range all {
		out[i] = &v1.ImportBank{Id: b.ID, Name: b.Name, Country: b.Country}
	}
	return out
}
