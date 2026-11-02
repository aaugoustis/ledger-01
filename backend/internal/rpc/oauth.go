package rpc

import (
	"context"

	"connectrpc.com/connect"
	v1 "github.com/franco-roura/ledger-01/backend/gen/ledger/v1"
	"github.com/franco-roura/ledger-01/backend/internal/oauth"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// OAuth implements OAuthService: the consent page and connected apps, for the signed-in user.
type OAuth struct {
	*Server
	store  *oauth.Store
	issuer string
}

func NewOAuth(s *Server, store *oauth.Store, issuer string) *OAuth {
	return &OAuth{Server: s, store: store, issuer: issuer}
}

func (s *OAuth) GetAuthorizationRequest(ctx context.Context, req *connect.Request[v1.GetAuthorizationRequestRequest]) (*connect.Response[v1.GetAuthorizationRequestResponse], error) {
	if _, err := s.actor(ctx); err != nil {
		return nil, err
	}
	id, err := parseID("id", req.Msg.Id)
	if err != nil {
		return nil, s.fail(ctx, err)
	}
	p, err := s.store.PendingRequest(ctx, id)
	if err != nil {
		return nil, s.fail(ctx, err)
	}
	return connect.NewResponse(&v1.GetAuthorizationRequestResponse{Request: &v1.AuthorizationRequest{
		Id: p.ID.String(), ClientName: p.ClientName, RedirectHost: p.RedirectHost, Scope: p.Scope,
	}}), nil
}

func (s *OAuth) ApproveAuthorization(ctx context.Context, req *connect.Request[v1.ApproveAuthorizationRequest]) (*connect.Response[v1.ApproveAuthorizationResponse], error) {
	a, err := s.actor(ctx)
	if err != nil {
		return nil, err
	}
	id, err := parseID("id", req.Msg.Id)
	if err != nil {
		return nil, s.fail(ctx, err)
	}
	to, err := s.store.Approve(ctx, a.UserID, id, s.issuer)
	if err != nil {
		return nil, s.fail(ctx, err)
	}
	return connect.NewResponse(&v1.ApproveAuthorizationResponse{RedirectUrl: to}), nil
}

func (s *OAuth) DenyAuthorization(ctx context.Context, req *connect.Request[v1.DenyAuthorizationRequest]) (*connect.Response[v1.DenyAuthorizationResponse], error) {
	if _, err := s.actor(ctx); err != nil {
		return nil, err
	}
	id, err := parseID("id", req.Msg.Id)
	if err != nil {
		return nil, s.fail(ctx, err)
	}
	to, err := s.store.Deny(ctx, id, s.issuer)
	if err != nil {
		return nil, s.fail(ctx, err)
	}
	return connect.NewResponse(&v1.DenyAuthorizationResponse{RedirectUrl: to}), nil
}

func (s *OAuth) ListConnections(ctx context.Context, _ *connect.Request[v1.ListConnectionsRequest]) (*connect.Response[v1.ListConnectionsResponse], error) {
	a, err := s.actor(ctx)
	if err != nil {
		return nil, err
	}
	cs, err := s.store.Connections(ctx, a.UserID)
	if err != nil {
		return nil, s.fail(ctx, err)
	}
	out := make([]*v1.Connection, len(cs))
	for i, c := range cs {
		out[i] = &v1.Connection{
			Id: c.ID.String(), ClientName: c.ClientName, RedirectHost: c.RedirectHost, Scope: c.Scope,
			CreatedAt: timestamppb.New(c.CreatedAt),
		}
		if c.LastUsedAt != nil {
			out[i].LastUsedAt = timestamppb.New(*c.LastUsedAt)
		}
	}
	return connect.NewResponse(&v1.ListConnectionsResponse{Connections: out}), nil
}

func (s *OAuth) RevokeConnection(ctx context.Context, req *connect.Request[v1.RevokeConnectionRequest]) (*connect.Response[v1.RevokeConnectionResponse], error) {
	a, err := s.actor(ctx)
	if err != nil {
		return nil, err
	}
	id, err := parseID("id", req.Msg.Id)
	if err != nil {
		return nil, s.fail(ctx, err)
	}
	if err := s.store.RevokeConnection(ctx, a.UserID, id); err != nil {
		return nil, s.fail(ctx, err)
	}
	return connect.NewResponse(&v1.RevokeConnectionResponse{}), nil
}
