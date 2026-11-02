-- name: CreateRecurringRule :exec
INSERT INTO recurring_rules (id, household_id, kind, wallet_id, to_wallet_id, category_id, amount_minor, to_amount_minor,
                             rate, note, frequency, start_date, end_date, generated_through, created_by)
VALUES (@id, @household_id, @kind, @wallet_id, sqlc.narg(to_wallet_id), sqlc.narg(category_id), @amount_minor,
        sqlc.narg(to_amount_minor), (sqlc.narg(rate)::text)::numeric, @note, @frequency, @start_date,
        sqlc.narg(end_date), @generated_through, @created_by);

-- name: AddRecurringRuleLabels :exec
INSERT INTO recurring_rule_labels (rule_id, label_id)
SELECT @rule_id::uuid, unnest(@label_ids::uuid[])
ON CONFLICT DO NOTHING;

-- name: LockRecurringRule :one
-- Serialises everything that writes a rule's occurrences (requests and the top-up job).
SELECT r.id, r.household_id, r.kind, r.wallet_id, r.to_wallet_id, r.category_id, r.amount_minor, r.to_amount_minor,
       coalesce(r.rate::text, '')::text AS rate, r.note, r.frequency, r.start_date, r.end_date, r.generated_through,
       r.created_by
FROM recurring_rules r
WHERE r.id = @id AND r.household_id = @household_id
FOR UPDATE;

-- name: ListRecurringRuleLabelIDs :many
SELECT label_id FROM recurring_rule_labels WHERE rule_id = $1 ORDER BY label_id;

-- name: UpdateRecurringRule :exec
-- Replaces a rule's template and schedule (a series edit from its first occurrence).
UPDATE recurring_rules
SET wallet_id         = @wallet_id,
    to_wallet_id      = sqlc.narg(to_wallet_id),
    category_id       = sqlc.narg(category_id),
    amount_minor      = @amount_minor,
    to_amount_minor   = sqlc.narg(to_amount_minor),
    rate              = (sqlc.narg(rate)::text)::numeric,
    note              = @note,
    frequency         = @frequency,
    start_date        = @start_date,
    end_date          = sqlc.narg(end_date),
    generated_through = @generated_through,
    updated_at        = now()
WHERE id = @id AND household_id = @household_id;

-- name: ClearRecurringRuleLabels :exec
DELETE FROM recurring_rule_labels WHERE rule_id = $1;

-- name: SetRuleEnd :exec
UPDATE recurring_rules SET end_date = @end_date, updated_at = now()
WHERE id = @id AND household_id = @household_id;

-- name: SetRuleGeneratedThrough :exec
UPDATE recurring_rules SET generated_through = @generated_through WHERE id = @id;

-- name: DeleteRecurringRule :exec
DELETE FROM recurring_rules WHERE id = @id AND household_id = @household_id;

-- name: CountRuleOccurrences :one
SELECT count(*) FROM transactions
WHERE recurring_rule_id = @rule_id AND kind <> 'transfer_in';

-- name: LinkTransaction :exec
UPDATE transactions SET recurring_rule_id = @rule_id, occurrence_date = @occurrence_date
WHERE id = @id AND household_id = @household_id;

-- name: LinkTransferLegs :exec
UPDATE transactions SET recurring_rule_id = @rule_id, occurrence_date = @occurrence_date
WHERE transfer_id = @transfer_id AND household_id = @household_id;

-- name: PendingOccurrences :many
-- A rule's occurrences scheduled on or after a date that aren't due yet (dated after today), one
-- row per occurrence (a transfer's outgoing leg).
SELECT id, transfer_id FROM transactions
WHERE household_id = @household_id
  AND recurring_rule_id = @rule_id
  AND occurrence_date >= @from_date
  AND occurred_on > @today
  AND kind <> 'transfer_in';

-- name: RulesToTopUp :many
-- Rules whose occurrences stop short of the horizon and that may still have some to create.
SELECT id, household_id FROM recurring_rules
WHERE generated_through < @horizon
  AND (end_date IS NULL OR end_date > generated_through)
ORDER BY id;

-- name: GetRuleSchedule :one
SELECT frequency, end_date FROM recurring_rules WHERE id = @id AND household_id = @household_id;
