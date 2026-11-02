package rpc

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	v1 "github.com/franco-roura/ledger-01/backend/gen/ledger/v1"
	"github.com/franco-roura/ledger-01/backend/internal/ledger"
	"github.com/franco-roura/ledger-01/backend/internal/money"
)

// BackupMaxBytes caps a backup upload; BackupReadMaxBytes is that file as base64 in JSON.
const (
	BackupMaxBytes     = 5 << 20
	BackupReadMaxBytes = 7 << 20
)

func (s *Server) ExportBackup(ctx context.Context, _ *connect.Request[v1.ExportBackupRequest]) (*connect.Response[v1.ExportBackupResponse], error) {
	a, err := s.actor(ctx)
	if err != nil {
		return nil, err
	}
	b, err := s.svc.ExportBackup(ctx, a)
	if err != nil {
		return nil, s.fail(ctx, err)
	}
	content, err := ledger.EncodeBackup(b)
	if err != nil {
		return nil, s.fail(ctx, err)
	}
	name := fmt.Sprintf("ledger-01-backup-%s.json.gz", money.FormatDate(b.ExportedAt))
	return connect.NewResponse(&v1.ExportBackupResponse{Content: content, FileName: name}), nil
}

func (s *Server) RestoreBackup(ctx context.Context, req *connect.Request[v1.RestoreBackupRequest]) (*connect.Response[v1.RestoreBackupResponse], error) {
	a, err := s.actor(ctx)
	if err != nil {
		return nil, err
	}
	if !req.Msg.Replace {
		return nil, s.fail(ctx, &ledger.InvalidError{Field: "replace", Msg: "must be true: restoring deletes the current data"})
	}
	if len(req.Msg.Content) > BackupMaxBytes {
		return nil, s.fail(ctx, &ledger.InvalidError{Field: "content", Msg: "is larger than 5 MB"})
	}
	b, err := ledger.DecodeBackup(req.Msg.Content)
	if err != nil {
		return nil, s.fail(ctx, err)
	}
	c, err := s.svc.RestoreBackup(ctx, a, b)
	if err != nil {
		return nil, s.fail(ctx, err)
	}
	return connect.NewResponse(&v1.RestoreBackupResponse{Counts: &v1.BackupCounts{
		Wallets: int32(c.Wallets), Categories: int32(c.Categories), Labels: int32(c.Labels),
		Transactions: int32(c.Transactions), Transfers: int32(c.Transfers), Budgets: int32(c.Budgets),
		RecurringRules: int32(c.RecurringRules), ImportRules: int32(c.ImportRules),
	}}), nil
}

func (s *Server) ExportTransactionsCsv(ctx context.Context, req *connect.Request[v1.ExportTransactionsCsvRequest]) (*connect.Response[v1.ExportTransactionsCsvResponse], error) {
	a, err := s.actor(ctx)
	if err != nil {
		return nil, err
	}
	today, err := parseToday(req.Msg.Today)
	if err != nil {
		return nil, s.fail(ctx, err)
	}
	f := ledger.CsvFilter{Today: today}
	if req.Msg.From != nil && *req.Msg.From != "" {
		d, err := parseDate("from", *req.Msg.From)
		if err != nil {
			return nil, s.fail(ctx, err)
		}
		f.From = &d
	}
	if req.Msg.To != nil && *req.Msg.To != "" {
		d, err := parseDate("to", *req.Msg.To)
		if err != nil {
			return nil, s.fail(ctx, err)
		}
		f.To = &d
	}
	if f.WalletIDs, err = parseIDs("wallet_ids", req.Msg.WalletIds); err != nil {
		return nil, s.fail(ctx, err)
	}
	content, n, err := s.svc.ExportTransactionsCsv(ctx, a, f)
	if err != nil {
		return nil, s.fail(ctx, err)
	}
	name := fmt.Sprintf("ledger-01-transactions-%s.csv", money.FormatDate(today))
	return connect.NewResponse(&v1.ExportTransactionsCsvResponse{Content: content, FileName: name, RowCount: int32(n)}), nil
}
