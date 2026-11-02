-- +goose Up
CREATE TABLE currencies (
    code        char(3) PRIMARY KEY,
    name        text     NOT NULL,
    symbol      text     NOT NULL,
    minor_units smallint NOT NULL CHECK (minor_units BETWEEN 0 AND 4)
);

CREATE TABLE users (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    google_sub    text        NOT NULL UNIQUE,
    email         text        NOT NULL,
    name          text        NOT NULL DEFAULT '',
    avatar_url    text        NOT NULL DEFAULT '',
    base_currency char(3)     NOT NULL DEFAULT 'USD' REFERENCES currencies (code),
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX users_email_key ON users (lower(email));

-- The tenant. Every ledger row belongs to exactly one household.
CREATE TABLE households (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name       text        NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE household_members (
    household_id uuid        NOT NULL REFERENCES households (id) ON DELETE CASCADE,
    user_id      uuid        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    role         text        NOT NULL CHECK (role IN ('owner', 'member')),
    created_at   timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (household_id, user_id)
);
CREATE INDEX household_members_user_idx ON household_members (user_id);

-- Only a SHA-256 of the session token is stored.
CREATE TABLE sessions (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      uuid        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    token_hash   bytea       NOT NULL UNIQUE,
    user_agent   text        NOT NULL DEFAULT '',
    created_at   timestamptz NOT NULL DEFAULT now(),
    last_seen_at timestamptz NOT NULL DEFAULT now(),
    expires_at   timestamptz NOT NULL
);
CREATE INDEX sessions_user_idx ON sessions (user_id);
CREATE INDEX sessions_expires_idx ON sessions (expires_at);

CREATE TABLE wallets (
    id                    uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    household_id          uuid        NOT NULL REFERENCES households (id) ON DELETE CASCADE,
    name                  text        NOT NULL CHECK (char_length(name) BETWEEN 1 AND 60),
    currency              char(3)     NOT NULL REFERENCES currencies (code),
    initial_balance_minor bigint      NOT NULL DEFAULT 0,
    hidden                boolean     NOT NULL DEFAULT false,
    position              integer     NOT NULL DEFAULT 0,
    archived_at           timestamptz,
    created_at            timestamptz NOT NULL DEFAULT now(),
    updated_at            timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX wallets_household_idx ON wallets (household_id);

CREATE TYPE category_kind AS ENUM ('expense', 'income');

CREATE TABLE categories (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    household_id uuid          NOT NULL REFERENCES households (id) ON DELETE CASCADE,
    kind         category_kind NOT NULL,
    name         text          NOT NULL CHECK (char_length(name) BETWEEN 1 AND 40),
    icon         text          NOT NULL,
    color        text          NOT NULL,
    position     integer       NOT NULL DEFAULT 0,
    archived_at  timestamptz,
    created_at   timestamptz   NOT NULL DEFAULT now(),
    updated_at   timestamptz   NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX categories_name_key ON categories (household_id, kind, lower(name));

-- A category hidden from one wallet's pickers. Existing transactions keep it.
CREATE TABLE category_wallet_hidden (
    wallet_id   uuid NOT NULL REFERENCES wallets (id) ON DELETE CASCADE,
    category_id uuid NOT NULL REFERENCES categories (id) ON DELETE CASCADE,
    PRIMARY KEY (wallet_id, category_id)
);
CREATE INDEX category_wallet_hidden_category_idx ON category_wallet_hidden (category_id);

CREATE TABLE labels (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    household_id uuid        NOT NULL REFERENCES households (id) ON DELETE CASCADE,
    name         text        NOT NULL CHECK (char_length(name) BETWEEN 1 AND 30),
    color        text        NOT NULL,
    created_at   timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX labels_name_key ON labels (household_id, lower(name));

-- A transfer owns two transaction legs (transfer_out + transfer_in).
CREATE TABLE transfers (
    id                uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    household_id      uuid           NOT NULL REFERENCES households (id) ON DELETE CASCADE,
    from_wallet_id    uuid           NOT NULL REFERENCES wallets (id) ON DELETE CASCADE,
    to_wallet_id      uuid           NOT NULL REFERENCES wallets (id) ON DELETE CASCADE,
    from_amount_minor bigint         NOT NULL CHECK (from_amount_minor > 0),
    to_amount_minor   bigint         NOT NULL CHECK (to_amount_minor > 0),
    -- Target units per 1 source unit.
    rate              numeric(20, 10) NOT NULL CHECK (rate > 0),
    occurred_on       date           NOT NULL,
    note              text           NOT NULL DEFAULT '',
    created_by        uuid           REFERENCES users (id) ON DELETE SET NULL,
    created_at        timestamptz    NOT NULL DEFAULT now(),
    updated_at        timestamptz    NOT NULL DEFAULT now(),
    CHECK (from_wallet_id <> to_wallet_id)
);
CREATE INDEX transfers_household_idx ON transfers (household_id);

CREATE TYPE transaction_kind AS ENUM ('expense', 'income', 'transfer_out', 'transfer_in');

-- Amounts are positive minor units in the wallet's currency; the kind gives the direction.
CREATE TABLE transactions (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    household_id uuid             NOT NULL REFERENCES households (id) ON DELETE CASCADE,
    wallet_id    uuid             NOT NULL REFERENCES wallets (id) ON DELETE CASCADE,
    kind         transaction_kind NOT NULL,
    category_id  uuid             REFERENCES categories (id),
    transfer_id  uuid             REFERENCES transfers (id) ON DELETE CASCADE,
    amount_minor bigint           NOT NULL CHECK (amount_minor > 0),
    occurred_on  date             NOT NULL,
    note         text             NOT NULL DEFAULT '' CHECK (char_length(note) <= 500),
    created_by   uuid             REFERENCES users (id) ON DELETE SET NULL,
    created_at   timestamptz      NOT NULL DEFAULT now(),
    updated_at   timestamptz      NOT NULL DEFAULT now(),
    CHECK (
        (kind IN ('expense', 'income') AND category_id IS NOT NULL AND transfer_id IS NULL)
        OR (kind IN ('transfer_out', 'transfer_in') AND category_id IS NULL AND transfer_id IS NOT NULL)
    )
);
CREATE INDEX transactions_wallet_date_idx ON transactions (wallet_id, occurred_on DESC, created_at DESC, id DESC);
CREATE INDEX transactions_household_idx ON transactions (household_id);
CREATE INDEX transactions_category_idx ON transactions (category_id);
CREATE INDEX transactions_transfer_idx ON transactions (transfer_id);

CREATE TABLE transaction_labels (
    transaction_id uuid NOT NULL REFERENCES transactions (id) ON DELETE CASCADE,
    label_id       uuid NOT NULL REFERENCES labels (id) ON DELETE CASCADE,
    PRIMARY KEY (transaction_id, label_id)
);
CREATE INDEX transaction_labels_label_idx ON transaction_labels (label_id);

-- +goose Down
DROP TABLE transaction_labels;
DROP TABLE transactions;
DROP TYPE transaction_kind;
DROP TABLE transfers;
DROP TABLE labels;
DROP TABLE category_wallet_hidden;
DROP TABLE categories;
DROP TYPE category_kind;
DROP TABLE wallets;
DROP TABLE sessions;
DROP TABLE household_members;
DROP TABLE households;
DROP TABLE users;
DROP TABLE currencies;
