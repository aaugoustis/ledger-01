-- name: ListBudgets :many
SELECT b.*,
       coalesce((SELECT array_agg(bw.wallet_id ORDER BY bw.wallet_id) FROM budget_wallets bw WHERE bw.budget_id = b.id), '{}')::uuid[] AS wallet_ids,
       coalesce((SELECT array_agg(bc.category_id ORDER BY bc.category_id) FROM budget_categories bc WHERE bc.budget_id = b.id), '{}')::uuid[] AS category_ids
FROM budgets b
WHERE b.household_id = $1
ORDER BY lower(b.name), b.id;

-- name: GetBudget :one
SELECT b.*,
       coalesce((SELECT array_agg(bw.wallet_id ORDER BY bw.wallet_id) FROM budget_wallets bw WHERE bw.budget_id = b.id), '{}')::uuid[] AS wallet_ids,
       coalesce((SELECT array_agg(bc.category_id ORDER BY bc.category_id) FROM budget_categories bc WHERE bc.budget_id = b.id), '{}')::uuid[] AS category_ids
FROM budgets b
WHERE b.id = @id AND b.household_id = @household_id;

-- name: CreateBudget :one
INSERT INTO budgets (id, household_id, name, amount_minor, currency, repeat, start_date, end_date)
VALUES (@id, @household_id, @name, @amount_minor, @currency, @repeat, @start_date, sqlc.narg(end_date))
RETURNING id;

-- name: UpdateBudget :execrows
UPDATE budgets
SET name = @name, amount_minor = @amount_minor, currency = @currency, repeat = @repeat,
    start_date = @start_date, end_date = sqlc.narg(end_date), updated_at = now()
WHERE id = @id AND household_id = @household_id;

-- name: DeleteBudget :execrows
DELETE FROM budgets WHERE id = @id AND household_id = @household_id;

-- name: ClearBudgetWallets :exec
DELETE FROM budget_wallets WHERE budget_id = $1;

-- name: AddBudgetWallets :exec
INSERT INTO budget_wallets (budget_id, wallet_id)
SELECT @budget_id, unnest(@wallet_ids::uuid[]);

-- name: ClearBudgetCategories :exec
DELETE FROM budget_categories WHERE budget_id = $1;

-- name: AddBudgetCategories :exec
INSERT INTO budget_categories (budget_id, category_id)
SELECT @budget_id, unnest(@category_ids::uuid[]);

-- name: ReassignBudgetCategories :exec
-- Before categories are merged or deleted with a replacement, budgets that counted them count
-- the replacement instead.
INSERT INTO budget_categories (budget_id, category_id)
SELECT DISTINCT bc.budget_id, @to_id::uuid
FROM budget_categories bc
JOIN budgets b ON b.id = bc.budget_id
WHERE b.household_id = @household_id AND bc.category_id = ANY(@from_ids::uuid[])
ON CONFLICT DO NOTHING;

-- name: CountWalletsByIDs :one
SELECT count(*) FROM wallets WHERE household_id = @household_id AND id = ANY(@ids::uuid[]);

-- name: CountExpenseCategoriesByIDs :one
SELECT count(*) FROM categories
WHERE household_id = @household_id AND kind = 'expense' AND id = ANY(@ids::uuid[]);

-- name: BudgetExpenses :many
-- Expenses a budget counts in a date range: its wallets (or every non-archived wallet) and its
-- categories (or every expense category), up to today. Newest first.
SELECT t.id,
       t.wallet_id,
       t.kind,
       t.category_id,
       t.transfer_id,
       t.amount_minor,
       t.occurred_on,
       t.note,
       t.created_by,
       t.created_at,
       t.updated_at,
       w.currency,
       coalesce((SELECT array_agg(tl.label_id ORDER BY tl.label_id)
                 FROM transaction_labels tl
                 WHERE tl.transaction_id = t.id), '{}')::uuid[] AS label_ids
FROM transactions t
JOIN wallets w ON w.id = t.wallet_id
WHERE t.household_id = @household_id
  AND t.kind = 'expense'
  AND t.occurred_on BETWEEN @from_date AND @to_date
  AND t.occurred_on <= @today
  AND (CASE WHEN cardinality(@wallet_ids::uuid[]) = 0 THEN w.archived_at IS NULL ELSE t.wallet_id = ANY(@wallet_ids::uuid[]) END)
  AND (cardinality(@category_ids::uuid[]) = 0 OR t.category_id = ANY(@category_ids::uuid[]))
ORDER BY t.occurred_on DESC, t.created_at DESC;
