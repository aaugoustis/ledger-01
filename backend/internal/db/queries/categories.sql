-- name: ListCategories :many
SELECT c.*,
       coalesce(t.cnt, 0)::bigint     AS transaction_count,
       coalesce(t.wallets, 0)::bigint AS wallet_count,
       coalesce((SELECT array_agg(h.wallet_id ORDER BY h.wallet_id)
                 FROM category_wallet_hidden h
                 WHERE h.category_id = c.id), '{}')::uuid[] AS hidden_in_wallet_ids
FROM categories c
LEFT JOIN (
    SELECT tx.category_id, count(*) AS cnt, count(DISTINCT tx.wallet_id) AS wallets
    FROM transactions tx
    WHERE tx.household_id = @household_id AND tx.category_id IS NOT NULL
    GROUP BY tx.category_id
) t ON t.category_id = c.id
WHERE c.household_id = @household_id
  AND (@include_archived::bool OR c.archived_at IS NULL)
ORDER BY c.kind, c.position, c.name;

-- name: GetCategory :one
SELECT * FROM categories WHERE id = @id AND household_id = @household_id;

-- name: GetCategoriesByIDs :many
SELECT * FROM categories WHERE household_id = @household_id AND id = ANY(@ids::uuid[]);

-- name: NextCategoryPosition :one
SELECT coalesce(max(position) + 1, 0)::int AS position
FROM categories
WHERE household_id = @household_id AND kind = @kind;

-- name: CreateCategory :one
INSERT INTO categories (id, household_id, kind, name, icon, color, position)
VALUES (@id, @household_id, @kind, @name, @icon, @color, @position)
RETURNING *;

-- name: UpdateCategory :one
UPDATE categories
SET name        = coalesce(sqlc.narg(name), name),
    icon        = coalesce(sqlc.narg(icon), icon),
    color       = coalesce(sqlc.narg(color), color),
    position    = coalesce(sqlc.narg(position), position),
    archived_at = CASE
                      WHEN sqlc.narg(archived)::bool IS NULL THEN archived_at
                      WHEN sqlc.narg(archived)::bool THEN coalesce(archived_at, now())
                      ELSE NULL
                  END,
    updated_at  = now()
WHERE id = @id AND household_id = @household_id
RETURNING *;

-- name: CountCategoryTransactions :one
SELECT count(*) FROM transactions WHERE category_id = ANY(@ids::uuid[]);

-- name: ReassignCategories :execrows
UPDATE transactions
SET category_id = @to_id, updated_at = now()
WHERE household_id = @household_id AND category_id = ANY(@from_ids::uuid[]);

-- name: CountCategoryRules :one
SELECT count(*) FROM recurring_rules WHERE category_id = ANY(@ids::uuid[]);

-- name: ReassignRuleCategories :exec
UPDATE recurring_rules
SET category_id = @to_id, updated_at = now()
WHERE household_id = @household_id AND category_id = ANY(@from_ids::uuid[]);

-- name: DeleteCategories :execrows
DELETE FROM categories WHERE household_id = @household_id AND id = ANY(@ids::uuid[]);

-- name: HideCategoryInWallet :exec
INSERT INTO category_wallet_hidden (wallet_id, category_id)
VALUES (@wallet_id, @category_id)
ON CONFLICT DO NOTHING;

-- name: ShowCategoryInWallet :exec
DELETE FROM category_wallet_hidden WHERE wallet_id = @wallet_id AND category_id = @category_id;
