package rpc

import (
	"context"

	"connectrpc.com/connect"
	v1 "github.com/franco-roura/ledger-01/backend/gen/ledger/v1"
	"github.com/franco-roura/ledger-01/backend/internal/ledger"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func walletToProto(w ledger.Wallet) *v1.Wallet {
	return &v1.Wallet{
		Id:                  w.ID.String(),
		Name:                w.Name,
		Currency:            w.Currency,
		InitialBalanceMinor: w.InitialBalance,
		BalanceMinor:        w.Balance,
		Hidden:              w.Hidden,
		Position:            w.Position,
		Archived:            w.Archived,
		TransactionCount:    w.TransactionCount,
		CreatedAt:           timestamppb.New(w.CreatedAt),
	}
}

func (s *Server) ListWallets(ctx context.Context, req *connect.Request[v1.ListWalletsRequest]) (*connect.Response[v1.ListWalletsResponse], error) {
	a, err := s.actor(ctx)
	if err != nil {
		return nil, err
	}
	today, err := parseToday(req.Msg.Today)
	if err != nil {
		return nil, s.fail(ctx, err)
	}
	ws, err := s.svc.ListWallets(ctx, a, req.Msg.IncludeHidden, req.Msg.IncludeArchived, today)
	if err != nil {
		return nil, s.fail(ctx, err)
	}
	out := make([]*v1.Wallet, len(ws))
	for i, w := range ws {
		out[i] = walletToProto(w)
	}
	return connect.NewResponse(&v1.ListWalletsResponse{Wallets: out}), nil
}

func (s *Server) GetWallet(ctx context.Context, req *connect.Request[v1.GetWalletRequest]) (*connect.Response[v1.GetWalletResponse], error) {
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
	w, err := s.svc.GetWalletOn(ctx, a, id, today)
	if err != nil {
		return nil, s.fail(ctx, err)
	}
	return connect.NewResponse(&v1.GetWalletResponse{Wallet: walletToProto(w)}), nil
}

func (s *Server) CreateWallet(ctx context.Context, req *connect.Request[v1.CreateWalletRequest]) (*connect.Response[v1.CreateWalletResponse], error) {
	a, err := s.actor(ctx)
	if err != nil {
		return nil, err
	}
	w, err := s.svc.CreateWallet(ctx, a, req.Msg.Name, req.Msg.Currency, req.Msg.InitialBalanceMinor)
	if err != nil {
		return nil, s.fail(ctx, err)
	}
	return connect.NewResponse(&v1.CreateWalletResponse{Wallet: walletToProto(w)}), nil
}

func (s *Server) UpdateWallet(ctx context.Context, req *connect.Request[v1.UpdateWalletRequest]) (*connect.Response[v1.UpdateWalletResponse], error) {
	a, err := s.actor(ctx)
	if err != nil {
		return nil, err
	}
	id, err := parseID("id", req.Msg.Id)
	if err != nil {
		return nil, s.fail(ctx, err)
	}
	w, err := s.svc.UpdateWallet(ctx, a, id, ledger.WalletPatch{
		Name: req.Msg.Name, Hidden: req.Msg.Hidden, Position: req.Msg.Position, InitialBalance: req.Msg.InitialBalanceMinor,
	})
	if err != nil {
		return nil, s.fail(ctx, err)
	}
	return connect.NewResponse(&v1.UpdateWalletResponse{Wallet: walletToProto(w)}), nil
}

func (s *Server) ArchiveWallet(ctx context.Context, req *connect.Request[v1.ArchiveWalletRequest]) (*connect.Response[v1.ArchiveWalletResponse], error) {
	a, err := s.actor(ctx)
	if err != nil {
		return nil, err
	}
	id, err := parseID("id", req.Msg.Id)
	if err != nil {
		return nil, s.fail(ctx, err)
	}
	if err := s.svc.ArchiveWallet(ctx, a, id); err != nil {
		return nil, s.fail(ctx, err)
	}
	return connect.NewResponse(&v1.ArchiveWalletResponse{}), nil
}

func (s *Server) DeleteWallet(ctx context.Context, req *connect.Request[v1.DeleteWalletRequest]) (*connect.Response[v1.DeleteWalletResponse], error) {
	a, err := s.actor(ctx)
	if err != nil {
		return nil, err
	}
	id, err := parseID("id", req.Msg.Id)
	if err != nil {
		return nil, s.fail(ctx, err)
	}
	if err := s.svc.DeleteWallet(ctx, a, id); err != nil {
		return nil, s.fail(ctx, err)
	}
	return connect.NewResponse(&v1.DeleteWalletResponse{}), nil
}
