-- +goose Up
CREATE TYPE recurring_kind AS ENUM ('expense', 'income', 'transfer');

CREATE TYPE recurring_frequency AS ENUM (
    'daily', 'every_2_days', 'weekdays', 'weekly', 'every_2_weeks', 'every_4_weeks',
    'monthly', 'every_2_months', 'every_3_months', 'every_6_months', 'yearly'
);

-- A repeating expense, income or transfer: a template plus a schedule whose first occurrence is
-- start_date. Occurrences are pre-created as ordinary (future-dated) transactions up to
-- generated_through. That date only moves forward, so an occurrence deleted on its own is never
-- created again.
CREATE TABLE recurring_rules (
    id                uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    household_id      uuid                NOT NULL REFERENCES households (id) ON DELETE CASCADE,
    kind              recurring_kind      NOT NULL,
    -- The expense/income wallet, or the transfer's source wallet.
    wallet_id         uuid                NOT NULL REFERENCES wallets (id) ON DELETE CASCADE,
    to_wallet_id      uuid                REFERENCES wallets (id) ON DELETE CASCADE,
    category_id       uuid                REFERENCES categories (id),
    -- The expense/income amount, or what the transfer sends.
    amount_minor      bigint              NOT NULL CHECK (amount_minor > 0),
    to_amount_minor   bigint              CHECK (to_amount_minor > 0),
    rate              numeric(20, 10)     CHECK (rate > 0),
    note              text                NOT NULL DEFAULT '' CHECK (char_length(note) <= 500),
    frequency         recurring_frequency NOT NULL,
    start_date        date                NOT NULL,
    -- Last date an occurrence may fall on; null repeats forever.
    end_date          date,
    generated_through date                NOT NULL,
    created_by        uuid                REFERENCES users (id) ON DELETE SET NULL,
    created_at        timestamptz         NOT NULL DEFAULT now(),
    updated_at        timestamptz         NOT NULL DEFAULT now(),
    CHECK (end_date IS NULL OR end_date >= start_date),
    CHECK (
        (kind IN ('expense', 'income') AND category_id IS NOT NULL AND to_wallet_id IS NULL
            AND to_amount_minor IS NULL AND rate IS NULL)
        OR (kind = 'transfer' AND category_id IS NULL AND to_wallet_id IS NOT NULL
            AND to_amount_minor IS NOT NULL AND rate IS NOT NULL AND to_wallet_id <> wallet_id)
    )
);
CREATE INDEX recurring_rules_household_idx ON recurring_rules (household_id);
CREATE INDEX recurring_rules_generated_idx ON recurring_rules (generated_through);
CREATE INDEX recurring_rules_category_idx ON recurring_rules (category_id);

CREATE TABLE recurring_rule_labels (
    rule_id  uuid NOT NULL REFERENCES recurring_rules (id) ON DELETE CASCADE,
    label_id uuid NOT NULL REFERENCES labels (id) ON DELETE CASCADE,
    PRIMARY KEY (rule_id, label_id)
);
CREATE INDEX recurring_rule_labels_label_idx ON recurring_rule_labels (label_id);

-- An occurrence keeps its scheduled date in occurrence_date even if occurred_on is edited.
ALTER TABLE transactions
    ADD COLUMN recurring_rule_id uuid REFERENCES recurring_rules (id) ON DELETE SET NULL,
    ADD COLUMN occurrence_date   date;

-- One occurrence per rule and scheduled date; a transfer's outgoing leg stands for the transfer.
-- This is what makes topping up idempotent.
CREATE UNIQUE INDEX transactions_occurrence_key ON transactions (recurring_rule_id, occurrence_date)
    WHERE recurring_rule_id IS NOT NULL AND kind <> 'transfer_in';
-- Upcoming rows across the household.
CREATE INDEX transactions_household_date_idx ON transactions (household_id, occurred_on);

-- +goose Down
DROP INDEX transactions_household_date_idx;
DROP INDEX transactions_occurrence_key;
ALTER TABLE transactions
    DROP COLUMN occurrence_date,
    DROP COLUMN recurring_rule_id;
DROP TABLE recurring_rule_labels;
DROP TABLE recurring_rules;
DROP TYPE recurring_frequency;
DROP TYPE recurring_kind;
