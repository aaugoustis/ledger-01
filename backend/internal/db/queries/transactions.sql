-- name: ListTransactions :many
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
                 WHERE tl.transaction_id = t.id), '{}')::uuid[] AS label_ids,
       cw.id                             AS counterpart_wallet_id,
       coalesce(cw.name, '')::text       AS counterpart_wallet_name,
       coalesce(cw.currency, '')::text   AS counterpart_currency,
       coalesce(CASE WHEN t.kind = 'transfer_out' THEN tr.to_amount_minor ELSE tr.from_amount_minor END, 0)::bigint AS counterpart_amount_minor,
       coalesce(tr.rate::text, '')::text AS transfer_rate,
       t.recurring_rule_id,
       r.frequency                       AS rule_frequency,
       r.end_date                        AS rule_end_date
FROM transactions t
JOIN wallets w ON w.id = t.wallet_id
LEFT JOIN transfers tr ON tr.id = t.transfer_id
LEFT JOIN wallets cw ON cw.id = CASE WHEN t.kind = 'transfer_out' THEN tr.to_wallet_id ELSE tr.from_wallet_id END
LEFT JOIN recurring_rules r ON r.id = t.recurring_rule_id
WHERE t.household_id = @household_id
  AND t.wallet_id = @wallet_id
  AND t.occurred_on BETWEEN @from_date AND @to_date
  AND (cardinality(@category_ids::uuid[]) = 0 OR t.category_id = ANY(@category_ids::uuid[]))
  AND (cardinality(@member_ids::uuid[]) = 0 OR t.created_by = ANY(@member_ids::uuid[]))
  AND (@note_query::text = '' OR strpos(lower(t.note), lower(@note_query::text)) > 0)
  AND (sqlc.narg(min_amount)::bigint IS NULL OR t.amount_minor >= sqlc.narg(min_amount)::bigint)
  AND (sqlc.narg(max_amount)::bigint IS NULL OR t.amount_minor <= sqlc.narg(max_amount)::bigint)
  AND (sqlc.narg(cursor_date)::date IS NULL
       OR (t.occurred_on, t.created_at, t.id)
          < (sqlc.narg(cursor_date)::date, sqlc.narg(cursor_created_at)::timestamptz, sqlc.narg(cursor_id)::uuid))
ORDER BY t.occurred_on DESC, t.created_at DESC, t.id DESC
LIMIT @page_limit;

-- name: SearchTransactions :many
-- ListTransactions across the household's wallets (all of them when wallet_ids is empty), with
-- label and kind filters. Same columns, so rows convert to ListTransactionsRow.
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
                 WHERE tl.transaction_id = t.id), '{}')::uuid[] AS label_ids,
       cw.id                             AS counterpart_wallet_id,
       coalesce(cw.name, '')::text       AS counterpart_wallet_name,
       coalesce(cw.currency, '')::text   AS counterpart_currency,
       coalesce(CASE WHEN t.kind = 'transfer_out' THEN tr.to_amount_minor ELSE tr.from_amount_minor END, 0)::bigint AS counterpart_amount_minor,
       coalesce(tr.rate::text, '')::text AS transfer_rate,
       t.recurring_rule_id,
       r.frequency                       AS rule_frequency,
       r.end_date                        AS rule_end_date
FROM transactions t
JOIN wallets w ON w.id = t.wallet_id
LEFT JOIN transfers tr ON tr.id = t.transfer_id
LEFT JOIN wallets cw ON cw.id = CASE WHEN t.kind = 'transfer_out' THEN tr.to_wallet_id ELSE tr.from_wallet_id END
LEFT JOIN recurring_rules r ON r.id = t.recurring_rule_id
WHERE t.household_id = @household_id
  AND (cardinality(@wallet_ids::uuid[]) = 0 OR t.wallet_id = ANY(@wallet_ids::uuid[]))
  AND t.occurred_on BETWEEN @from_date AND @to_date
  AND (cardinality(@kinds::text[]) = 0 OR t.kind::text = ANY(@kinds::text[]))
  AND (cardinality(@category_ids::uuid[]) = 0 OR t.category_id = ANY(@category_ids::uuid[]))
  AND (cardinality(@member_ids::uuid[]) = 0 OR t.created_by = ANY(@member_ids::uuid[]))
  AND (cardinality(@label_ids::uuid[]) = 0
       OR EXISTS (SELECT 1 FROM transaction_labels tl
                  WHERE tl.transaction_id = t.id AND tl.label_id = ANY(@label_ids::uuid[])))
  AND (@note_query::text = '' OR strpos(lower(t.note), lower(@note_query::text)) > 0)
  AND (sqlc.narg(min_amount)::bigint IS NULL OR t.amount_minor >= sqlc.narg(min_amount)::bigint)
  AND (sqlc.narg(max_amount)::bigint IS NULL OR t.amount_minor <= sqlc.narg(max_amount)::bigint)
  AND (sqlc.narg(cursor_date)::date IS NULL
       OR (t.occurred_on, t.created_at, t.id)
          < (sqlc.narg(cursor_date)::date, sqlc.narg(cursor_created_at)::timestamptz, sqlc.narg(cursor_id)::uuid))
ORDER BY t.occurred_on DESC, t.created_at DESC, t.id DESC
LIMIT @page_limit;

-- name: ListUpcomingTransactions :many
-- Rows dated after today, soonest first: one wallet's, or every non-archived wallet's with each
-- transfer listed once (its outgoing leg).
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
                 WHERE tl.transaction_id = t.id), '{}')::uuid[] AS label_ids,
       cw.id                             AS counterpart_wallet_id,
       coalesce(cw.name, '')::text       AS counterpart_wallet_name,
       coalesce(cw.currency, '')::text   AS counterpart_currency,
       coalesce(CASE WHEN t.kind = 'transfer_out' THEN tr.to_amount_minor ELSE tr.from_amount_minor END, 0)::bigint AS counterpart_amount_minor,
       coalesce(tr.rate::text, '')::text AS transfer_rate,
       t.recurring_rule_id,
       r.frequency                       AS rule_frequency,
       r.end_date                        AS rule_end_date
FROM transactions t
JOIN wallets w ON w.id = t.wallet_id
LEFT JOIN transfers tr ON tr.id = t.transfer_id
LEFT JOIN wallets cw ON cw.id = CASE WHEN t.kind = 'transfer_out' THEN tr.to_wallet_id ELSE tr.from_wallet_id END
LEFT JOIN recurring_rules r ON r.id = t.recurring_rule_id
WHERE t.household_id = @household_id
  AND t.occurred_on > @today
  AND (CASE WHEN sqlc.narg(wallet_id)::uuid IS NULL
            THEN w.archived_at IS NULL AND t.kind <> 'transfer_in'
            ELSE t.wallet_id = sqlc.narg(wallet_id)::uuid END)
  AND (cardinality(@member_ids::uuid[]) = 0 OR t.created_by = ANY(@member_ids::uuid[]))
ORDER BY t.occurred_on, t.created_at, t.id
LIMIT 500;

-- name: PeriodSummary :one
SELECT coalesce(sum(CASE WHEN t.kind = 'expense' THEN t.amount_minor END), 0)::bigint AS expenses_minor,
       coalesce(sum(CASE WHEN t.kind = 'income' THEN t.amount_minor END), 0)::bigint  AS income_minor,
       coalesce(sum(CASE WHEN t.kind IN ('income', 'transfer_in') THEN t.amount_minor ELSE -t.amount_minor END), 0)::bigint AS change_minor,
       count(*)::bigint AS transaction_count
FROM transactions t
WHERE t.household_id = @household_id
  AND t.wallet_id = @wallet_id
  AND t.occurred_on BETWEEN @from_date AND @to_date
  AND (cardinality(@category_ids::uuid[]) = 0 OR t.category_id = ANY(@category_ids::uuid[]))
  AND (cardinality(@member_ids::uuid[]) = 0 OR t.created_by = ANY(@member_ids::uuid[]))
  AND (@note_query::text = '' OR strpos(lower(t.note), lower(@note_query::text)) > 0)
  AND (sqlc.narg(min_amount)::bigint IS NULL OR t.amount_minor >= sqlc.narg(min_amount)::bigint)
  AND (sqlc.narg(max_amount)::bigint IS NULL OR t.amount_minor <= sqlc.narg(max_amount)::bigint);

-- name: GetTransaction :one
SELECT * FROM transactions WHERE id = @id AND household_id = @household_id;

-- name: CreateTransaction :one
INSERT INTO transactions (id, household_id, wallet_id, kind, category_id, transfer_id, amount_minor, occurred_on, note, created_by,
                          recurring_rule_id, occurrence_date)
VALUES (@id, @household_id, @wallet_id, @kind, sqlc.narg(category_id), sqlc.narg(transfer_id), @amount_minor, @occurred_on, @note, @created_by,
        sqlc.narg(recurring_rule_id), sqlc.narg(occurrence_date))
RETURNING *;

-- name: UpdateTransaction :one
UPDATE transactions
SET kind         = @kind,
    category_id  = @category_id,
    amount_minor = @amount_minor,
    occurred_on  = @occurred_on,
    note         = @note,
    updated_at   = now()
WHERE id = @id AND household_id = @household_id AND kind IN ('expense', 'income')
RETURNING *;

-- name: ClearTransactionLabels :exec
DELETE FROM transaction_labels WHERE transaction_id = ANY(@ids::uuid[]);

-- name: AddTransactionLabels :exec
INSERT INTO transaction_labels (transaction_id, label_id)
SELECT @transaction_id::uuid, unnest(@label_ids::uuid[])
ON CONFLICT DO NOTHING;

-- name: ListTransactionLabelIDs :many
SELECT label_id FROM transaction_labels WHERE transaction_id = $1 ORDER BY label_id;

-- name: GetTransactionRefs :many
SELECT id, transfer_id, recurring_rule_id, occurrence_date
FROM transactions
WHERE household_id = @household_id AND id = ANY(@ids::uuid[]);

-- name: DeletePlainTransactions :execrows
DELETE FROM transactions
WHERE household_id = @household_id AND id = ANY(@ids::uuid[]) AND transfer_id IS NULL;

-- name: CountTransferLegs :one
SELECT count(*) FROM transactions WHERE transfer_id = ANY(@transfer_ids::uuid[]);

-- name: DeleteTransfers :execrows
DELETE FROM transfers WHERE household_id = @household_id AND id = ANY(@ids::uuid[]);
