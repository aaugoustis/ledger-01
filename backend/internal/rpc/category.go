package rpc

import (
	"context"

	"connectrpc.com/connect"
	v1 "github.com/franco-roura/ledger-01/backend/gen/ledger/v1"
	"github.com/franco-roura/ledger-01/backend/internal/ledger"
)

func categoryKindToProto(k ledger.CategoryKind) v1.CategoryKind {
	switch k {
	case ledger.CategoryExpense:
		return v1.CategoryKind_CATEGORY_KIND_EXPENSE
	case ledger.CategoryIncome:
		return v1.CategoryKind_CATEGORY_KIND_INCOME
	}
	return v1.CategoryKind_CATEGORY_KIND_UNSPECIFIED
}

func categoryKindFromProto(k v1.CategoryKind) ledger.CategoryKind {
	switch k {
	case v1.CategoryKind_CATEGORY_KIND_EXPENSE:
		return ledger.CategoryExpense
	case v1.CategoryKind_CATEGORY_KIND_INCOME:
		return ledger.CategoryIncome
	}
	return ""
}

func categoryToProto(c ledger.Category) *v1.Category {
	return &v1.Category{
		Id:                c.ID.String(),
		Kind:              categoryKindToProto(c.Kind),
		Name:              c.Name,
		Icon:              c.Icon,
		Color:             c.Color,
		Position:          c.Position,
		Archived:          c.Archived,
		TransactionCount:  c.TransactionCount,
		WalletCount:       c.WalletCount,
		HiddenInWalletIds: idStrings(c.HiddenInWalletIDs),
	}
}

func (s *Server) ListCategories(ctx context.Context, req *connect.Request[v1.ListCategoriesRequest]) (*connect.Response[v1.ListCategoriesResponse], error) {
	a, err := s.actor(ctx)
	if err != nil {
		return nil, err
	}
	cs, err := s.svc.ListCategories(ctx, a, req.Msg.IncludeArchived)
	if err != nil {
		return nil, s.fail(ctx, err)
	}
	out := make([]*v1.Category, len(cs))
	for i, c := range cs {
		out[i] = categoryToProto(c)
	}
	return connect.NewResponse(&v1.ListCategoriesResponse{Categories: out}), nil
}

func (s *Server) CreateCategory(ctx context.Context, req *connect.Request[v1.CreateCategoryRequest]) (*connect.Response[v1.CreateCategoryResponse], error) {
	a, err := s.actor(ctx)
	if err != nil {
		return nil, err
	}
	m := req.Msg
	c, err := s.svc.CreateCategory(ctx, a, categoryKindFromProto(m.Kind), m.Name, m.Icon, m.Color)
	if err != nil {
		return nil, s.fail(ctx, err)
	}
	return connect.NewResponse(&v1.CreateCategoryResponse{Category: categoryToProto(c)}), nil
}

func (s *Server) UpdateCategory(ctx context.Context, req *connect.Request[v1.UpdateCategoryRequest]) (*connect.Response[v1.UpdateCategoryResponse], error) {
	a, err := s.actor(ctx)
	if err != nil {
		return nil, err
	}
	m := req.Msg
	id, err := parseID("id", m.Id)
	if err != nil {
		return nil, s.fail(ctx, err)
	}
	c, err := s.svc.UpdateCategory(ctx, a, id, ledger.CategoryPatch{
		Name: m.Name, Icon: m.Icon, Color: m.Color, Position: m.Position, Archived: m.Archived,
	})
	if err != nil {
		return nil, s.fail(ctx, err)
	}
	return connect.NewResponse(&v1.UpdateCategoryResponse{Category: categoryToProto(c)}), nil
}

func (s *Server) DeleteCategory(ctx context.Context, req *connect.Request[v1.DeleteCategoryRequest]) (*connect.Response[v1.DeleteCategoryResponse], error) {
	a, err := s.actor(ctx)
	if err != nil {
		return nil, err
	}
	id, err := parseID("id", req.Msg.Id)
	if err != nil {
		return nil, s.fail(ctx, err)
	}
	repl, err := parseOptID("replacement_id", req.Msg.ReplacementId)
	if err != nil {
		return nil, s.fail(ctx, err)
	}
	moved, err := s.svc.DeleteCategory(ctx, a, id, repl)
	if err != nil {
		return nil, s.fail(ctx, err)
	}
	return connect.NewResponse(&v1.DeleteCategoryResponse{MovedTransactionCount: moved}), nil
}

func (s *Server) MergeCategories(ctx context.Context, req *connect.Request[v1.MergeCategoriesRequest]) (*connect.Response[v1.MergeCategoriesResponse], error) {
	a, err := s.actor(ctx)
	if err != nil {
		return nil, err
	}
	sources, err := parseIDs("source_ids", req.Msg.SourceIds)
	if err != nil {
		return nil, s.fail(ctx, err)
	}
	target, err := parseID("target_id", req.Msg.TargetId)
	if err != nil {
		return nil, s.fail(ctx, err)
	}
	c, moved, err := s.svc.MergeCategories(ctx, a, sources, target)
	if err != nil {
		return nil, s.fail(ctx, err)
	}
	return connect.NewResponse(&v1.MergeCategoriesResponse{Target: categoryToProto(c), MovedTransactionCount: moved}), nil
}

func (s *Server) SetCategoryHidden(ctx context.Context, req *connect.Request[v1.SetCategoryHiddenRequest]) (*connect.Response[v1.SetCategoryHiddenResponse], error) {
	a, err := s.actor(ctx)
	if err != nil {
		return nil, err
	}
	wallet, err := parseID("wallet_id", req.Msg.WalletId)
	if err != nil {
		return nil, s.fail(ctx, err)
	}
	cat, err := parseID("category_id", req.Msg.CategoryId)
	if err != nil {
		return nil, s.fail(ctx, err)
	}
	if err := s.svc.SetCategoryHidden(ctx, a, wallet, cat, req.Msg.Hidden); err != nil {
		return nil, s.fail(ctx, err)
	}
	return connect.NewResponse(&v1.SetCategoryHiddenResponse{}), nil
}
