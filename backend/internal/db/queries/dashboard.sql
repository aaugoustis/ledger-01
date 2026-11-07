-- name: ReportWallets :many
-- Wallets in a report: the requested ones, or every visible one. Opening balance is the balance
-- just before the period starts. Rows after today are upcoming and never count in reports.
SELECT w.id,
       w.currency,
       (w.initial_balance_minor + coalesce((
           SELECT sum(CASE WHEN t.kind IN ('income', 'transfer_in') THEN t.amount_minor ELSE -t.amount_minor END)
           FROM transactions t
           WHERE t.wallet_id = w.id AND t.occurred_on < @from_date AND t.occurred_on <= @today), 0))::bigint AS opening_minor
FROM wallets w
WHERE w.household_id = @household_id
  AND w.archived_at IS NULL
  AND (CASE WHEN cardinality(@wallet_ids::uuid[]) = 0 THEN NOT w.hidden ELSE w.id = ANY(@wallet_ids::uuid[]) END)
ORDER BY w.position;

-- name: ReportTransactions :many
SELECT t.id, t.wallet_id, t.kind, t.category_id, t.amount_minor, t.occurred_on, t.note
FROM transactions t
WHERE t.household_id = @household_id
  AND t.wallet_id = ANY(@wallet_ids::uuid[])
  AND t.occurred_on BETWEEN @from_date AND @to_date
  AND t.occurred_on <= @today
ORDER BY t.occurred_on;

-- name: TransferRates :many
-- Every cross-currency transfer of the household up to today, oldest first, as rate points.
SELECT tr.occurred_on, fw.currency AS from_currency, tw.currency AS to_currency, tr.rate::text AS rate
FROM transfers tr
JOIN wallets fw ON fw.id = tr.from_wallet_id
JOIN wallets tw ON tw.id = tr.to_wallet_id
WHERE tr.household_id = @household_id AND fw.currency <> tw.currency AND tr.occurred_on <= @today
ORDER BY tr.occurred_on, tr.created_at;

-- name: CategoryTransactions :many
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
  AND t.category_id = @category_id
  AND t.wallet_id = ANY(@wallet_ids::uuid[])
  AND t.occurred_on BETWEEN @from_date AND @to_date
  AND t.occurred_on <= @today
ORDER BY t.occurred_on DESC, t.created_at DESC;
