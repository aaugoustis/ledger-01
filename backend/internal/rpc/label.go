package rpc

import (
	"context"

	"connectrpc.com/connect"
	v1 "github.com/franco-roura/ledger-01/backend/gen/ledger/v1"
	"github.com/franco-roura/ledger-01/backend/internal/ledger"
)

func labelToProto(l ledger.Label) *v1.Label {
	return &v1.Label{Id: l.ID.String(), Name: l.Name, Color: l.Color}
}

func (s *Server) ListLabels(ctx context.Context, _ *connect.Request[v1.ListLabelsRequest]) (*connect.Response[v1.ListLabelsResponse], error) {
	a, err := s.actor(ctx)
	if err != nil {
		return nil, err
	}
	ls, err := s.svc.ListLabels(ctx, a)
	if err != nil {
		return nil, s.fail(ctx, err)
	}
	out := make([]*v1.Label, len(ls))
	for i, l := range ls {
		out[i] = labelToProto(l)
	}
	return connect.NewResponse(&v1.ListLabelsResponse{Labels: out}), nil
}

func (s *Server) CreateLabel(ctx context.Context, req *connect.Request[v1.CreateLabelRequest]) (*connect.Response[v1.CreateLabelResponse], error) {
	a, err := s.actor(ctx)
	if err != nil {
		return nil, err
	}
	l, err := s.svc.CreateLabel(ctx, a, req.Msg.Name, req.Msg.Color)
	if err != nil {
		return nil, s.fail(ctx, err)
	}
	return connect.NewResponse(&v1.CreateLabelResponse{Label: labelToProto(l)}), nil
}

func (s *Server) UpdateLabel(ctx context.Context, req *connect.Request[v1.UpdateLabelRequest]) (*connect.Response[v1.UpdateLabelResponse], error) {
	a, err := s.actor(ctx)
	if err != nil {
		return nil, err
	}
	id, err := parseID("id", req.Msg.Id)
	if err != nil {
		return nil, s.fail(ctx, err)
	}
	l, err := s.svc.UpdateLabel(ctx, a, id, req.Msg.Name, req.Msg.Color)
	if err != nil {
		return nil, s.fail(ctx, err)
	}
	return connect.NewResponse(&v1.UpdateLabelResponse{Label: labelToProto(l)}), nil
}

func (s *Server) DeleteLabel(ctx context.Context, req *connect.Request[v1.DeleteLabelRequest]) (*connect.Response[v1.DeleteLabelResponse], error) {
	a, err := s.actor(ctx)
	if err != nil {
		return nil, err
	}
	id, err := parseID("id", req.Msg.Id)
	if err != nil {
		return nil, s.fail(ctx, err)
	}
	if err := s.svc.DeleteLabel(ctx, a, id); err != nil {
		return nil, s.fail(ctx, err)
	}
	return connect.NewResponse(&v1.DeleteLabelResponse{}), nil
}
