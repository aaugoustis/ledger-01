package rpc

import (
	"context"

	"connectrpc.com/connect"
	v1 "github.com/franco-roura/ledger-01/backend/gen/ledger/v1"
	"github.com/franco-roura/ledger-01/backend/internal/ledger"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func memberToProto(m ledger.Member) *v1.Member {
	return &v1.Member{UserId: m.UserID.String(), Email: m.Email, Name: m.Name, AvatarUrl: m.AvatarURL, Role: m.Role, Former: m.Former}
}

func (s *Server) inviteToProto(i ledger.Invite) *v1.Invite {
	return &v1.Invite{Email: i.Email, CreatedAt: timestamppb.New(i.CreatedAt), Allowed: s.emailAllowed(i.Email)}
}

func (s *Server) ListMembers(ctx context.Context, _ *connect.Request[v1.ListMembersRequest]) (*connect.Response[v1.ListMembersResponse], error) {
	a, err := s.actor(ctx)
	if err != nil {
		return nil, err
	}
	ms, err := s.svc.Members(ctx, a)
	if err != nil {
		return nil, s.fail(ctx, err)
	}
	out := &v1.ListMembersResponse{}
	for _, m := range ms.Members {
		out.Members = append(out.Members, memberToProto(m))
	}
	for _, m := range ms.Former {
		out.FormerMembers = append(out.FormerMembers, memberToProto(m))
	}
	for _, i := range ms.Invites {
		out.Invites = append(out.Invites, s.inviteToProto(i))
	}
	return connect.NewResponse(out), nil
}

func (s *Server) UpdateHousehold(ctx context.Context, req *connect.Request[v1.UpdateHouseholdRequest]) (*connect.Response[v1.UpdateHouseholdResponse], error) {
	a, err := s.actor(ctx)
	if err != nil {
		return nil, err
	}
	h, err := s.svc.UpdateHousehold(ctx, a, req.Msg.Name, req.Msg.BaseCurrency)
	if err != nil {
		return nil, s.fail(ctx, err)
	}
	return connect.NewResponse(&v1.UpdateHouseholdResponse{Household: householdToProto(h)}), nil
}

func (s *Server) InviteMember(ctx context.Context, req *connect.Request[v1.InviteMemberRequest]) (*connect.Response[v1.InviteMemberResponse], error) {
	a, err := s.actor(ctx)
	if err != nil {
		return nil, err
	}
	inv, err := s.svc.InviteMember(ctx, a, req.Msg.Email)
	if err != nil {
		return nil, s.fail(ctx, err)
	}
	return connect.NewResponse(&v1.InviteMemberResponse{Invite: s.inviteToProto(inv)}), nil
}

func (s *Server) CancelInvite(ctx context.Context, req *connect.Request[v1.CancelInviteRequest]) (*connect.Response[v1.CancelInviteResponse], error) {
	a, err := s.actor(ctx)
	if err != nil {
		return nil, err
	}
	if err := s.svc.CancelInvite(ctx, a, req.Msg.Email); err != nil {
		return nil, s.fail(ctx, err)
	}
	return connect.NewResponse(&v1.CancelInviteResponse{}), nil
}

func (s *Server) RemoveMember(ctx context.Context, req *connect.Request[v1.RemoveMemberRequest]) (*connect.Response[v1.RemoveMemberResponse], error) {
	a, err := s.actor(ctx)
	if err != nil {
		return nil, err
	}
	id, err := parseID("user_id", req.Msg.UserId)
	if err != nil {
		return nil, s.fail(ctx, err)
	}
	if err := s.svc.RemoveMember(ctx, a, id); err != nil {
		return nil, s.fail(ctx, err)
	}
	return connect.NewResponse(&v1.RemoveMemberResponse{}), nil
}

func (s *Server) LeaveHousehold(ctx context.Context, _ *connect.Request[v1.LeaveHouseholdRequest]) (*connect.Response[v1.LeaveHouseholdResponse], error) {
	a, err := s.actor(ctx)
	if err != nil {
		return nil, err
	}
	if err := s.svc.LeaveHousehold(ctx, a); err != nil {
		return nil, s.fail(ctx, err)
	}
	return connect.NewResponse(&v1.LeaveHouseholdResponse{}), nil
}

func (s *Server) ListHouseholds(ctx context.Context, _ *connect.Request[v1.ListHouseholdsRequest]) (*connect.Response[v1.ListHouseholdsResponse], error) {
	a, err := s.actor(ctx)
	if err != nil {
		return nil, err
	}
	hs, err := s.svc.Households(ctx, a)
	if err != nil {
		return nil, s.fail(ctx, err)
	}
	out := make([]*v1.HouseholdMembership, len(hs))
	for i, h := range hs {
		out[i] = &v1.HouseholdMembership{Id: h.ID.String(), Name: h.Name, Role: h.Role, MemberCount: int32(h.MemberCount), Active: h.Active}
	}
	return connect.NewResponse(&v1.ListHouseholdsResponse{Households: out}), nil
}

func (s *Server) SwitchHousehold(ctx context.Context, req *connect.Request[v1.SwitchHouseholdRequest]) (*connect.Response[v1.SwitchHouseholdResponse], error) {
	a, err := s.actor(ctx)
	if err != nil {
		return nil, err
	}
	id, err := parseID("household_id", req.Msg.HouseholdId)
	if err != nil {
		return nil, s.fail(ctx, err)
	}
	if err := s.svc.SwitchHousehold(ctx, a, id); err != nil {
		return nil, s.fail(ctx, err)
	}
	return connect.NewResponse(&v1.SwitchHouseholdResponse{}), nil
}
