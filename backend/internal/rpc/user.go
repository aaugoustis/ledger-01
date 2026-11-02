package rpc

import (
	"context"

	"connectrpc.com/connect"
	v1 "github.com/franco-roura/ledger-01/backend/gen/ledger/v1"
	"github.com/franco-roura/ledger-01/backend/internal/ledger"
)

func userToProto(u ledger.User) *v1.User {
	return &v1.User{Id: u.ID.String(), Email: u.Email, Name: u.Name, AvatarUrl: u.AvatarURL}
}

func householdToProto(h ledger.Household) *v1.Household {
	return &v1.Household{Id: h.ID.String(), Name: h.Name, BaseCurrency: h.BaseCurrency, Role: h.Role}
}

func (s *Server) GetMe(ctx context.Context, _ *connect.Request[v1.GetMeRequest]) (*connect.Response[v1.GetMeResponse], error) {
	a, err := s.actor(ctx)
	if err != nil {
		return nil, err
	}
	u, h, err := s.svc.Me(ctx, a)
	if err != nil {
		return nil, s.fail(ctx, err)
	}
	return connect.NewResponse(&v1.GetMeResponse{
		User:      userToProto(u),
		Household: householdToProto(h),
	}), nil
}

func (s *Server) UpdateProfile(ctx context.Context, req *connect.Request[v1.UpdateProfileRequest]) (*connect.Response[v1.UpdateProfileResponse], error) {
	a, err := s.actor(ctx)
	if err != nil {
		return nil, err
	}
	u, err := s.svc.UpdateProfile(ctx, a, req.Msg.Name)
	if err != nil {
		return nil, s.fail(ctx, err)
	}
	return connect.NewResponse(&v1.UpdateProfileResponse{User: userToProto(u)}), nil
}

func (s *Server) ListCurrencies(ctx context.Context, _ *connect.Request[v1.ListCurrenciesRequest]) (*connect.Response[v1.ListCurrenciesResponse], error) {
	cs, err := s.svc.Currencies(ctx)
	if err != nil {
		return nil, s.fail(ctx, err)
	}
	out := make([]*v1.Currency, len(cs))
	for i, c := range cs {
		out[i] = &v1.Currency{Code: c.Code, Name: c.Name, Symbol: c.Symbol, MinorUnits: int32(c.MinorUnits)}
	}
	return connect.NewResponse(&v1.ListCurrenciesResponse{Currencies: out}), nil
}
