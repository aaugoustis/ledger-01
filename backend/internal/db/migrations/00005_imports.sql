-- +goose Up
-- An uploaded bank file's rows (text cells, header included), kept between the import steps so
-- each step can reach a different API instance. cmd/jobs deletes expired batches.
CREATE TABLE import_batches (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    household_id uuid        NOT NULL REFERENCES households (id) ON DELETE CASCADE,
    wallet_id    uuid        NOT NULL REFERENCES wallets (id) ON DELETE CASCADE,
    file_name    text        NOT NULL,
    rows         jsonb       NOT NULL,
    created_by   uuid        REFERENCES users (id) ON DELETE SET NULL,
    created_at   timestamptz NOT NULL DEFAULT now(),
    expires_at   timestamptz NOT NULL
);
CREATE INDEX import_batches_household_idx ON import_batches (household_id);
CREATE INDEX import_batches_expires_idx ON import_batches (expires_at);

-- The column mapping last used to import into a wallet (its "bank profile"), with the header
-- row it was saved for, so the next import of the same export is one click.
CREATE TABLE import_profiles (
    wallet_id    uuid PRIMARY KEY REFERENCES wallets (id) ON DELETE CASCADE,
    household_id uuid        NOT NULL REFERENCES households (id) ON DELETE CASCADE,
    mapping      jsonb       NOT NULL,
    headers      jsonb       NOT NULL DEFAULT '[]',
    updated_at   timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX import_profiles_household_idx ON import_profiles (household_id);

-- "Note contains pattern -> category" for imported rows.
CREATE TABLE import_category_rules (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    household_id uuid        NOT NULL REFERENCES households (id) ON DELETE CASCADE,
    pattern      text        NOT NULL CHECK (char_length(pattern) BETWEEN 1 AND 100),
    category_id  uuid        NOT NULL REFERENCES categories (id) ON DELETE CASCADE,
    created_at   timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX import_category_rules_pattern_key ON import_category_rules (household_id, lower(pattern));
CREATE INDEX import_category_rules_category_idx ON import_category_rules (category_id);

-- +goose Down
DROP TABLE import_category_rules;
DROP TABLE import_profiles;
DROP TABLE import_batches;
