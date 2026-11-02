-- name: ListWallets :many
SELECT w.*,
       c.minor_units,
       coalesce(m.movement, 0)::bigint AS movement_minor,
       coalesce(m.cnt, 0)::bigint      AS transaction_count
FROM wallets w
JOIN currencies c ON c.code = w.currency
LEFT JOIN (
    -- Upcoming rows (after today) don't move the balance yet, but they count as transactions.
    SELECT t.wallet_id,
           sum(CASE WHEN t.kind IN ('income', 'transfer_in') THEN t.amount_minor ELSE -t.amount_minor END)
               FILTER (WHERE t.occurred_on <= @today) AS movement,
           count(*) AS cnt
    FROM transactions t
    WHERE t.household_id = @household_id
    GROUP BY t.wallet_id
) m ON m.wallet_id = w.id
WHERE w.household_id = @household_id
  AND (@include_hidden::bool OR NOT w.hidden)
  AND (@include_archived::bool OR w.archived_at IS NULL)
ORDER BY w.position, w.created_at;

-- name: GetWallet :one
SELECT w.*,
       c.minor_units,
       coalesce((SELECT sum(CASE WHEN t.kind IN ('income', 'transfer_in') THEN t.amount_minor ELSE -t.amount_minor END)
                 FROM transactions t WHERE t.wallet_id = w.id AND t.occurred_on <= @today), 0)::bigint AS movement_minor,
       (SELECT count(*) FROM transactions t WHERE t.wallet_id = w.id)::bigint AS transaction_count
FROM wallets w
JOIN currencies c ON c.code = w.currency
WHERE w.id = @id AND w.household_id = @household_id;

-- name: NextWalletPosition :one
SELECT coalesce(max(position) + 1, 0)::int AS position FROM wallets WHERE household_id = $1;

-- name: CreateWallet :one
INSERT INTO wallets (id, household_id, name, currency, initial_balance_minor, position)
VALUES (@id, @household_id, @name, @currency, @initial_balance_minor, @position)
RETURNING *;

-- name: UpdateWallet :one
UPDATE wallets
SET name                  = coalesce(sqlc.narg(name), name),
    hidden                = coalesce(sqlc.narg(hidden), hidden),
    position              = coalesce(sqlc.narg(position), position),
    initial_balance_minor = coalesce(sqlc.narg(initial_balance_minor), initial_balance_minor),
    updated_at            = now()
WHERE id = @id AND household_id = @household_id
RETURNING *;

-- name: ArchiveWallet :execrows
UPDATE wallets SET archived_at = coalesce(archived_at, now()), updated_at = now()
WHERE id = @id AND household_id = @household_id;

-- name: DeleteEmptyWallet :execrows
DELETE FROM wallets w
WHERE w.id = @id
  AND w.household_id = @household_id
  AND NOT EXISTS (SELECT 1 FROM transactions t WHERE t.wallet_id = w.id);

-- name: GetWalletsByIDs :many
SELECT w.id, w.name, w.currency, w.archived_at, c.minor_units
FROM wallets w
JOIN currencies c ON c.code = w.currency
WHERE w.household_id = @household_id AND w.id = ANY(@ids::uuid[]);
