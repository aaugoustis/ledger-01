package ledger

import (
	"context"
	"regexp"
	"slices"

	"github.com/franco-roura/ledger-01/backend/internal/db/dbq"
	"github.com/google/uuid"
)

type CategoryKind = dbq.CategoryKind

const (
	CategoryExpense = dbq.CategoryKindExpense
	CategoryIncome  = dbq.CategoryKindIncome
)

type Category struct {
	ID                uuid.UUID
	Kind              CategoryKind
	Name              string
	Icon              string
	Color             string
	Position          int32
	Archived          bool
	TransactionCount  int64
	WalletCount       int64
	HiddenInWalletIDs []uuid.UUID
}

var (
	iconRe  = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)
	colorRe = regexp.MustCompile(`^cat-[a-z]+$`)
)

func checkIconColor(icon, color *string) error {
	if icon != nil && (len(*icon) > 40 || !iconRe.MatchString(*icon)) {
		return invalid("icon", "must be a Lucide icon name like \"shopping-cart\"")
	}
	if color != nil && !colorRe.MatchString(*color) {
		return invalid("color", "must be a palette token like \"cat-orange\"")
	}
	return nil
}

func categoryFrom(c dbq.Category) Category {
	return Category{
		ID: c.ID, Kind: c.Kind, Name: c.Name, Icon: c.Icon, Color: c.Color, Position: c.Position, Archived: c.ArchivedAt != nil,
	}
}

func (s *Service) ListCategories(ctx context.Context, a Actor, includeArchived bool) ([]Category, error) {
	rows, err := s.q.ListCategories(ctx, dbq.ListCategoriesParams{HouseholdID: a.HouseholdID, IncludeArchived: includeArchived})
	if err != nil {
		return nil, err
	}
	out := make([]Category, len(rows))
	for i, r := range rows {
		out[i] = Category{
			ID: r.ID, Kind: r.Kind, Name: r.Name, Icon: r.Icon, Color: r.Color, Position: r.Position,
			Archived: r.ArchivedAt != nil, TransactionCount: r.TransactionCount, WalletCount: r.WalletCount,
			HiddenInWalletIDs: r.HiddenInWalletIds,
		}
	}
	return out, nil
}

func (s *Service) getCategory(ctx context.Context, a Actor, id uuid.UUID) (Category, error) {
	c, err := s.q.GetCategory(ctx, dbq.GetCategoryParams{ID: id, HouseholdID: a.HouseholdID})
	if err != nil {
		return Category{}, dbErr(err, "category")
	}
	return categoryFrom(c), nil
}

// withCounts re-reads one category through the list query so counts are filled in.
func (s *Service) withCounts(ctx context.Context, a Actor, id uuid.UUID) (Category, error) {
	all, err := s.ListCategories(ctx, a, true)
	if err != nil {
		return Category{}, err
	}
	for _, c := range all {
		if c.ID == id {
			return c, nil
		}
	}
	return Category{}, ErrNotFound
}

func (s *Service) CreateCategory(ctx context.Context, a Actor, kind CategoryKind, name, icon, color string) (Category, error) {
	if kind != CategoryExpense && kind != CategoryIncome {
		return Category{}, invalid("kind", "must be expense or income")
	}
	name, err := cleanName("name", name, 40)
	if err != nil {
		return Category{}, err
	}
	if err := checkIconColor(&icon, &color); err != nil {
		return Category{}, err
	}
	pos, err := s.q.NextCategoryPosition(ctx, dbq.NextCategoryPositionParams{HouseholdID: a.HouseholdID, Kind: kind})
	if err != nil {
		return Category{}, err
	}
	c, err := s.q.CreateCategory(ctx, dbq.CreateCategoryParams{
		ID: newID(), HouseholdID: a.HouseholdID, Kind: kind, Name: name, Icon: icon, Color: color, Position: pos,
	})
	if err != nil {
		return Category{}, dbErr(err, "category")
	}
	return categoryFrom(c), nil
}

type CategoryPatch struct {
	Name     *string
	Icon     *string
	Color    *string
	Position *int32
	Archived *bool
}

func (s *Service) UpdateCategory(ctx context.Context, a Actor, id uuid.UUID, p CategoryPatch) (Category, error) {
	if p.Name != nil {
		n, err := cleanName("name", *p.Name, 40)
		if err != nil {
			return Category{}, err
		}
		p.Name = &n
	}
	if err := checkIconColor(p.Icon, p.Color); err != nil {
		return Category{}, err
	}
	if _, err := s.q.UpdateCategory(ctx, dbq.UpdateCategoryParams{
		ID: id, HouseholdID: a.HouseholdID, Name: p.Name, Icon: p.Icon, Color: p.Color, Position: p.Position, Archived: p.Archived,
	}); err != nil {
		return Category{}, dbErr(err, "category")
	}
	return s.withCounts(ctx, a, id)
}

// DeleteCategory deletes a category. Transactions using it move to replacement, which is
// required when there are any.
func (s *Service) DeleteCategory(ctx context.Context, a Actor, id uuid.UUID, replacement *uuid.UUID) (int64, error) {
	var moved int64
	err := s.inTx(ctx, func(q *dbq.Queries) error {
		c, err := q.GetCategory(ctx, dbq.GetCategoryParams{ID: id, HouseholdID: a.HouseholdID})
		if err != nil {
			return dbErr(err, "category")
		}
		used, err := q.CountCategoryTransactions(ctx, []uuid.UUID{id})
		if err != nil {
			return err
		}
		rules, err := q.CountCategoryRules(ctx, []uuid.UUID{id})
		if err != nil {
			return err
		}
		if used > 0 || rules > 0 {
			if replacement == nil && used == 0 {
				return invalid("replacement_id", "category is used by %d recurring rules; choose a replacement or archive it", rules)
			}
			if replacement == nil {
				return invalid("replacement_id", "category has %d transactions; choose a replacement or archive it", used)
			}
			if err := checkTarget(ctx, q, a, *replacement, c.Kind, []uuid.UUID{id}); err != nil {
				return err
			}
			if moved, err = q.ReassignCategories(ctx, dbq.ReassignCategoriesParams{
				ToID: replacement, HouseholdID: a.HouseholdID, FromIds: []uuid.UUID{id},
			}); err != nil {
				return err
			}
		}
		if replacement != nil {
			if err := q.ReassignBudgetCategories(ctx, dbq.ReassignBudgetCategoriesParams{
				ToID: *replacement, HouseholdID: a.HouseholdID, FromIds: []uuid.UUID{id},
			}); err != nil {
				return err
			}
			if err := q.ReassignRuleCategories(ctx, dbq.ReassignRuleCategoriesParams{
				ToID: replacement, HouseholdID: a.HouseholdID, FromIds: []uuid.UUID{id},
			}); err != nil {
				return err
			}
			if err := q.ReassignImportRuleCategories(ctx, dbq.ReassignImportRuleCategoriesParams{
				ToID: *replacement, HouseholdID: a.HouseholdID, FromIds: []uuid.UUID{id},
			}); err != nil {
				return err
			}
		}
		_, err = q.DeleteCategories(ctx, dbq.DeleteCategoriesParams{HouseholdID: a.HouseholdID, Ids: []uuid.UUID{id}})
		return err
	})
	return moved, err
}

// MergeCategories moves every transaction of the sources into target, then deletes the sources.
func (s *Service) MergeCategories(ctx context.Context, a Actor, sources []uuid.UUID, target uuid.UUID) (Category, int64, error) {
	sources = uniqueIDs(sources)
	if len(sources) == 0 {
		return Category{}, 0, invalid("source_ids", "choose at least one category to merge")
	}
	if slices.Contains(sources, target) {
		return Category{}, 0, invalid("target_id", "can't merge a category into itself")
	}
	var moved int64
	err := s.inTx(ctx, func(q *dbq.Queries) error {
		t, err := q.GetCategory(ctx, dbq.GetCategoryParams{ID: target, HouseholdID: a.HouseholdID})
		if err != nil {
			return dbErr(err, "target category")
		}
		src, err := q.GetCategoriesByIDs(ctx, dbq.GetCategoriesByIDsParams{HouseholdID: a.HouseholdID, Ids: sources})
		if err != nil {
			return err
		}
		if len(src) != len(sources) {
			return invalid("source_ids", "some categories don't exist")
		}
		for _, c := range src {
			if c.Kind != t.Kind {
				return invalid("source_ids", "can only merge categories of the same kind")
			}
		}
		if moved, err = q.ReassignCategories(ctx, dbq.ReassignCategoriesParams{
			ToID: &target, HouseholdID: a.HouseholdID, FromIds: sources,
		}); err != nil {
			return err
		}
		if err := q.ReassignBudgetCategories(ctx, dbq.ReassignBudgetCategoriesParams{
			ToID: target, HouseholdID: a.HouseholdID, FromIds: sources,
		}); err != nil {
			return err
		}
		if err := q.ReassignRuleCategories(ctx, dbq.ReassignRuleCategoriesParams{
			ToID: &target, HouseholdID: a.HouseholdID, FromIds: sources,
		}); err != nil {
			return err
		}
		if err := q.ReassignImportRuleCategories(ctx, dbq.ReassignImportRuleCategoriesParams{
			ToID: target, HouseholdID: a.HouseholdID, FromIds: sources,
		}); err != nil {
			return err
		}
		_, err = q.DeleteCategories(ctx, dbq.DeleteCategoriesParams{HouseholdID: a.HouseholdID, Ids: sources})
		return err
	})
	if err != nil {
		return Category{}, 0, err
	}
	c, err := s.withCounts(ctx, a, target)
	return c, moved, err
}

func checkTarget(ctx context.Context, q *dbq.Queries, a Actor, target uuid.UUID, kind CategoryKind, notIn []uuid.UUID) error {
	if slices.Contains(notIn, target) {
		return invalid("replacement_id", "must be a different category")
	}
	t, err := q.GetCategory(ctx, dbq.GetCategoryParams{ID: target, HouseholdID: a.HouseholdID})
	if err != nil {
		return invalid("replacement_id", "category doesn't exist")
	}
	if t.Kind != kind {
		return invalid("replacement_id", "must be a %s category", kind)
	}
	return nil
}

// SetCategoryHidden hides or shows a category in one wallet's pickers.
func (s *Service) SetCategoryHidden(ctx context.Context, a Actor, walletID, categoryID uuid.UUID, hidden bool) error {
	if _, err := s.GetWallet(ctx, a, walletID); err != nil {
		return err
	}
	if _, err := s.getCategory(ctx, a, categoryID); err != nil {
		return err
	}
	if hidden {
		return s.q.HideCategoryInWallet(ctx, dbq.HideCategoryInWalletParams{WalletID: walletID, CategoryID: categoryID})
	}
	return s.q.ShowCategoryInWallet(ctx, dbq.ShowCategoryInWalletParams{WalletID: walletID, CategoryID: categoryID})
}
