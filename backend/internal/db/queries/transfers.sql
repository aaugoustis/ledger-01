-- name: CreateTransfer :exec
INSERT INTO transfers (id, household_id, from_wallet_id, to_wallet_id, from_amount_minor, to_amount_minor, rate, occurred_on, note, created_by)
VALUES (@id, @household_id, @from_wallet_id, @to_wallet_id, @from_amount_minor, @to_amount_minor,
        (@rate::text)::numeric, @occurred_on, @note, @created_by);

-- name: UpdateTransfer :execrows
UPDATE transfers
SET from_wallet_id    = @from_wallet_id,
    to_wallet_id      = @to_wallet_id,
    from_amount_minor = @from_amount_minor,
    to_amount_minor   = @to_amount_minor,
    rate              = (@rate::text)::numeric,
    occurred_on       = @occurred_on,
    note              = @note,
    updated_at        = now()
WHERE id = @id AND household_id = @household_id;

-- name: GetTransfer :one
SELECT id, from_wallet_id, to_wallet_id, from_amount_minor, to_amount_minor, rate::text AS rate, occurred_on, note
FROM transfers
WHERE id = @id AND household_id = @household_id;

-- name: UpdateTransferLeg :one
UPDATE transactions
SET wallet_id    = @wallet_id,
    amount_minor = @amount_minor,
    occurred_on  = @occurred_on,
    note         = @note,
    updated_at   = now()
WHERE transfer_id = @transfer_id AND kind = @kind
RETURNING id;

-- name: GetTransferOccurrence :one
-- The recurring rule and scheduled date of a transfer (both legs carry them).
SELECT recurring_rule_id, occurrence_date FROM transactions
WHERE transfer_id = @transfer_id AND household_id = @household_id AND kind = 'transfer_out';
