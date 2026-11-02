-- name: CreateImportBatch :exec
INSERT INTO import_batches (id, household_id, wallet_id, file_name, rows, created_by, expires_at)
VALUES (@id, @household_id, @wallet_id, @file_name, @rows, @created_by, @expires_at);

-- name: GetImportBatch :one
SELECT id, wallet_id, file_name, rows FROM import_batches
WHERE id = @id AND household_id = @household_id AND expires_at > now();

-- name: DeleteImportBatch :exec
DELETE FROM import_batches WHERE id = @id AND household_id = @household_id;

-- name: DeleteExpiredImportBatches :execrows
DELETE FROM import_batches WHERE expires_at <= @now;

-- name: GetImportProfile :one
SELECT mapping, headers FROM import_profiles WHERE wallet_id = @wallet_id AND household_id = @household_id;

-- name: SaveImportProfile :exec
INSERT INTO import_profiles (wallet_id, household_id, mapping, headers)
VALUES (@wallet_id, @household_id, @mapping, @headers)
ON CONFLICT (wallet_id) DO UPDATE SET mapping = excluded.mapping, headers = excluded.headers, updated_at = now();

-- name: ImportCandidates :many
-- A wallet's expenses and income in a date range, to match imported rows against: signed amounts,
-- the current note, and the bank descriptions of the rows it was imported from.
SELECT t.id,
       t.kind,
       t.occurred_on,
       (CASE WHEN t.kind = 'income' THEN t.amount_minor ELSE -t.amount_minor END)::bigint AS signed_minor,
       t.note,
       coalesce(array_agg(r.note ORDER BY r.created_at) FILTER (WHERE r.note IS NOT NULL), '{}')::text[] AS source_notes
FROM transactions t
LEFT JOIN import_refs r ON r.transaction_id = t.id
WHERE t.household_id = @household_id AND t.wallet_id = @wallet_id
  AND t.kind IN ('expense', 'income')
  AND t.occurred_on BETWEEN @from_date AND @to_date
GROUP BY t.id;

-- name: ImportRefsByFingerprint :many
SELECT r.fingerprint, r.transaction_id, t.occurred_on, t.note
FROM import_refs r
JOIN transactions t ON t.id = r.transaction_id
WHERE r.household_id = @household_id AND r.wallet_id = @wallet_id
  AND r.fingerprint = ANY(@fingerprints::text[]);

-- name: InsertImportRefs :exec
-- The source rows of imported transactions (the arrays line up row by row).
INSERT INTO import_refs (wallet_id, household_id, fingerprint, transaction_id, occurred_on, amount_minor, note)
SELECT @wallet_id, @household_id, unnest(@fingerprints::text[]), unnest(@transaction_ids::uuid[]),
       unnest(@dates::date[]), unnest(@amounts::bigint[]), unnest(@notes::text[])
ON CONFLICT (wallet_id, fingerprint) DO NOTHING;

-- name: ApplyImportDiscount :execrows
-- Lowers an expense by a discount imported later (e.g. Uruguay's REDIVA 19210 VAT refund).
UPDATE transactions
SET amount_minor = amount_minor - @discount_minor, updated_at = now()
WHERE id = @id AND household_id = @household_id AND wallet_id = @wallet_id
  AND kind = 'expense' AND amount_minor > @discount_minor;

-- name: ListImportCategoryRules :many
SELECT id, pattern, category_id FROM import_category_rules
WHERE household_id = $1
ORDER BY lower(pattern);

-- name: CreateImportCategoryRule :one
INSERT INTO import_category_rules (id, household_id, pattern, category_id)
VALUES (@id, @household_id, @pattern, @category_id)
RETURNING id, pattern, category_id;

-- name: DeleteImportCategoryRule :execrows
DELETE FROM import_category_rules WHERE id = @id AND household_id = @household_id;

-- name: ReassignImportRuleCategories :exec
UPDATE import_category_rules
SET category_id = @to_id
WHERE household_id = @household_id AND category_id = ANY(@from_ids::uuid[]);

-- name: InsertImportedTransactions :execrows
-- Expenses and income from an import, all at once (the arrays line up row by row).
INSERT INTO transactions (id, household_id, wallet_id, kind, category_id, amount_minor, occurred_on, note, created_by)
SELECT unnest(@ids::uuid[]), @household_id, @wallet_id, unnest(@kinds::text[])::transaction_kind,
       unnest(@category_ids::uuid[]), unnest(@amounts::bigint[]), unnest(@dates::date[]), unnest(@notes::text[]),
       @created_by;
