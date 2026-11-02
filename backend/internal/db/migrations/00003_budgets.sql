-- +goose Up
CREATE TYPE budget_repeat AS ENUM ('once', 'daily', 'weekly', 'biweekly', 'monthly', 'yearly');

-- A spending limit over repeating periods that start on start_date. "once" budgets cover
-- start_date..end_date; every other repeat leaves end_date null and runs indefinitely.
CREATE TABLE budgets (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    household_id uuid          NOT NULL REFERENCES households (id) ON DELETE CASCADE,
    name         text          NOT NULL CHECK (char_length(name) BETWEEN 1 AND 60),
    amount_minor bigint        NOT NULL CHECK (amount_minor > 0),
    currency     char(3)       NOT NULL REFERENCES currencies (code),
    repeat       budget_repeat NOT NULL,
    start_date   date          NOT NULL,
    end_date     date,
    created_at   timestamptz   NOT NULL DEFAULT now(),
    updated_at   timestamptz   NOT NULL DEFAULT now(),
    CHECK ((repeat = 'once') = (end_date IS NOT NULL)),
    CHECK (end_date IS NULL OR end_date >= start_date)
);
CREATE INDEX budgets_household_idx ON budgets (household_id);

-- No rows means every wallet / every expense category counts.
CREATE TABLE budget_wallets (
    budget_id uuid NOT NULL REFERENCES budgets (id) ON DELETE CASCADE,
    wallet_id uuid NOT NULL REFERENCES wallets (id) ON DELETE CASCADE,
    PRIMARY KEY (budget_id, wallet_id)
);

CREATE TABLE budget_categories (
    budget_id   uuid NOT NULL REFERENCES budgets (id) ON DELETE CASCADE,
    category_id uuid NOT NULL REFERENCES categories (id) ON DELETE CASCADE,
    PRIMARY KEY (budget_id, category_id)
);

-- +goose Down
DROP TABLE budget_categories;
DROP TABLE budget_wallets;
DROP TABLE budgets;
DROP TYPE budget_repeat;
