-- Household backups (ledger.ExportBackup / RestoreBackup): every household-scoped ledger row,
-- read as-is and written back under new ids. Creators travel as emails and are matched to the
-- restoring household's members.

-- name: BackupWallets :many
SELECT id, name, currency, initial_balance_minor, hidden, position, archived_at FROM wallets
WHERE household_id = @household_id ORDER BY position, created_at, id;

-- name: BackupCategories :many
SELECT id, kind, name, icon, color, position, archived_at FROM categories
WHERE household_id = @household_id ORDER BY kind, position, created_at, id;

-- name: BackupCategoryHidden :many
SELECT h.wallet_id, h.category_id FROM category_wallet_hidden h
JOIN wallets w ON w.id = h.wallet_id
WHERE w.household_id = @household_id ORDER BY h.wallet_id, h.category_id;

-- name: BackupLabels :many
SELECT id, name, color FROM labels WHERE household_id = @household_id ORDER BY created_at, id;

-- name: BackupTransfers :many
SELECT id, from_wallet_id, to_wallet_id, from_amount_minor, to_amount_minor, rate::text AS rate, occurred_on,
       note, created_at, coalesce((SELECT u.email FROM users u WHERE u.id = created_by), '')::text AS created_by_email
FROM transfers WHERE household_id = @household_id ORDER BY occurred_on, created_at, id;

-- name: BackupTransactions :many
SELECT id, wallet_id, kind, category_id, transfer_id, amount_minor, occurred_on, note, recurring_rule_id,
       occurrence_date, created_at, coalesce((SELECT u.email FROM users u WHERE u.id = created_by), '')::text AS created_by_email
FROM transactions WHERE household_id = @household_id ORDER BY occurred_on, created_at, id;

-- name: BackupTransactionLabels :many
SELECT tl.transaction_id, tl.label_id FROM transaction_labels tl
JOIN transactions t ON t.id = tl.transaction_id
WHERE t.household_id = @household_id ORDER BY tl.transaction_id, tl.label_id;

-- name: BackupBudgets :many
SELECT id, name, amount_minor, currency, repeat, start_date, end_date FROM budgets
WHERE household_id = @household_id ORDER BY created_at, id;

-- name: BackupBudgetWallets :many
SELECT bw.budget_id, bw.wallet_id FROM budget_wallets bw
JOIN budgets b ON b.id = bw.budget_id
WHERE b.household_id = @household_id ORDER BY bw.budget_id, bw.wallet_id;

-- name: BackupBudgetCategories :many
SELECT bc.budget_id, bc.category_id FROM budget_categories bc
JOIN budgets b ON b.id = bc.budget_id
WHERE b.household_id = @household_id ORDER BY bc.budget_id, bc.category_id;

-- name: BackupRecurringRules :many
SELECT id, kind, wallet_id, to_wallet_id, category_id, amount_minor, to_amount_minor,
       coalesce(rate::text, '')::text AS rate, note,
       frequency, start_date, end_date, generated_through, coalesce((SELECT u.email FROM users u WHERE u.id = created_by), '')::text AS created_by_email
FROM recurring_rules WHERE household_id = @household_id ORDER BY created_at, id;

-- name: BackupRecurringRuleLabels :many
SELECT rl.rule_id, rl.label_id FROM recurring_rule_labels rl
JOIN recurring_rules r ON r.id = rl.rule_id
WHERE r.household_id = @household_id ORDER BY rl.rule_id, rl.label_id;

-- name: BackupImportProfiles :many
SELECT wallet_id, mapping, headers FROM import_profiles WHERE household_id = @household_id ORDER BY wallet_id;

-- name: BackupImportCategoryRules :many
SELECT id, pattern, category_id FROM import_category_rules WHERE household_id = @household_id
ORDER BY created_at, id;

-- name: BackupImportRefs :many
SELECT wallet_id, fingerprint, transaction_id, occurred_on, amount_minor, note FROM import_refs
WHERE household_id = @household_id ORDER BY wallet_id, fingerprint;

-- Clearing a household before a restore, children before parents. Deleting transactions takes
-- their labels and import refs; deleting wallets takes per-wallet hidden categories, budget
-- wallets and import profiles.

-- name: ClearTransactions :exec
DELETE FROM transactions WHERE household_id = @household_id;

-- name: ClearTransfers :exec
DELETE FROM transfers WHERE household_id = @household_id;

-- name: ClearRecurringRules :exec
DELETE FROM recurring_rules WHERE household_id = @household_id;

-- name: ClearBudgets :exec
DELETE FROM budgets WHERE household_id = @household_id;

-- name: ClearImportCategoryRules :exec
DELETE FROM import_category_rules WHERE household_id = @household_id;

-- name: ClearImportBatches :exec
DELETE FROM import_batches WHERE household_id = @household_id;

-- name: ClearWallets :exec
DELETE FROM wallets WHERE household_id = @household_id;

-- name: ClearLabels :exec
DELETE FROM labels WHERE household_id = @household_id;

-- name: ClearCategories :exec
DELETE FROM categories WHERE household_id = @household_id;

-- name: RestoreWallet :exec
INSERT INTO wallets (id, household_id, name, currency, initial_balance_minor, hidden, position, archived_at)
VALUES (@id, @household_id, @name, @currency, @initial_balance_minor, @hidden, @position, @archived_at);

-- name: RestoreCategory :exec
INSERT INTO categories (id, household_id, kind, name, icon, color, position, archived_at)
VALUES (@id, @household_id, @kind, @name, @icon, @color, @position, @archived_at);

-- name: RestoreCategoryHidden :copyfrom
INSERT INTO category_wallet_hidden (wallet_id, category_id) VALUES ($1, $2);

-- name: RestoreLabel :exec
INSERT INTO labels (id, household_id, name, color) VALUES (@id, @household_id, @name, @color);

-- name: RestoreTransfer :exec
INSERT INTO transfers (id, household_id, from_wallet_id, to_wallet_id, from_amount_minor, to_amount_minor, rate,
                       occurred_on, note, created_by, created_at)
VALUES (@id, @household_id, @from_wallet_id, @to_wallet_id, @from_amount_minor, @to_amount_minor,
        (@rate::text)::numeric, @occurred_on, @note, @created_by, @created_at);

-- name: RestoreRecurringRule :exec
INSERT INTO recurring_rules (id, household_id, kind, wallet_id, to_wallet_id, category_id, amount_minor,
                             to_amount_minor, rate, note, frequency, start_date, end_date, generated_through,
                             created_by)
VALUES (@id, @household_id, @kind, @wallet_id, sqlc.narg(to_wallet_id), sqlc.narg(category_id), @amount_minor,
        sqlc.narg(to_amount_minor), (sqlc.narg(rate)::text)::numeric, @note, @frequency, @start_date,
        sqlc.narg(end_date), @generated_through, @created_by);

-- name: RestoreRecurringRuleLabels :copyfrom
INSERT INTO recurring_rule_labels (rule_id, label_id) VALUES ($1, $2);

-- name: RestoreTransactions :copyfrom
INSERT INTO transactions (id, household_id, wallet_id, kind, category_id, transfer_id, amount_minor, occurred_on,
                          note, recurring_rule_id, occurrence_date, created_by, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13);

-- name: RestoreTransactionLabels :copyfrom
INSERT INTO transaction_labels (transaction_id, label_id) VALUES ($1, $2);

-- name: RestoreBudget :exec
INSERT INTO budgets (id, household_id, name, amount_minor, currency, repeat, start_date, end_date)
VALUES (@id, @household_id, @name, @amount_minor, @currency, @repeat, @start_date, sqlc.narg(end_date));

-- name: RestoreBudgetWallets :copyfrom
INSERT INTO budget_wallets (budget_id, wallet_id) VALUES ($1, $2);

-- name: RestoreBudgetCategories :copyfrom
INSERT INTO budget_categories (budget_id, category_id) VALUES ($1, $2);

-- name: RestoreImportProfile :exec
INSERT INTO import_profiles (wallet_id, household_id, mapping, headers)
VALUES (@wallet_id, @household_id, @mapping, @headers);

-- name: RestoreImportCategoryRule :exec
INSERT INTO import_category_rules (id, household_id, pattern, category_id)
VALUES (@id, @household_id, @pattern, @category_id);

-- name: RestoreImportRefs :copyfrom
INSERT INTO import_refs (wallet_id, household_id, fingerprint, transaction_id, occurred_on, amount_minor, note)
VALUES ($1, $2, $3, $4, $5, $6, $7);

-- name: RestoreHouseholdSettings :exec
UPDATE households SET name = @name, base_currency = @base_currency WHERE id = @household_id;

-- name: BackupHousehold :one
SELECT h.name, h.base_currency FROM households h WHERE h.id = @household_id;

-- name: ExportTransactionRows :many
-- One CSV row per transaction, with the other leg's wallet for transfers.
SELECT t.occurred_on, w.name AS wallet, w.currency, t.kind, coalesce(c.name, '')::text AS category, t.amount_minor,
       t.note, coalesce(ow.name, '')::text AS counterpart,
       coalesce((SELECT string_agg(l.name, ', ' ORDER BY lower(l.name))
                 FROM transaction_labels tl JOIN labels l ON l.id = tl.label_id
                 WHERE tl.transaction_id = t.id), '')::text AS labels
FROM transactions t
JOIN wallets w ON w.id = t.wallet_id
LEFT JOIN categories c ON c.id = t.category_id
LEFT JOIN transactions o ON o.transfer_id = t.transfer_id AND o.id <> t.id
LEFT JOIN wallets ow ON ow.id = o.wallet_id
WHERE t.household_id = @household_id
  AND t.occurred_on <= @today
  AND (sqlc.narg(from_date)::date IS NULL OR t.occurred_on >= sqlc.narg(from_date)::date)
  AND (sqlc.narg(to_date)::date IS NULL OR t.occurred_on <= sqlc.narg(to_date)::date)
  AND (cardinality(@wallet_ids::uuid[]) = 0 OR t.wallet_id = ANY (@wallet_ids::uuid[]))
ORDER BY t.occurred_on, t.created_at, t.id;
