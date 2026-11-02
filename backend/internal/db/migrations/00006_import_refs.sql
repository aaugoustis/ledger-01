-- +goose Up
-- Where an imported transaction came from: a fingerprint of the bank row (its date, amount and
-- description as they were in the file) plus those values, so a later import of the same row is
-- recognised even after the transaction was renamed or repriced. A transaction can carry several
-- rows (a purchase and the discount merged into it).
CREATE TABLE import_refs (
    wallet_id      uuid        NOT NULL REFERENCES wallets (id) ON DELETE CASCADE,
    household_id   uuid        NOT NULL REFERENCES households (id) ON DELETE CASCADE,
    fingerprint    text        NOT NULL CHECK (char_length(fingerprint) = 16),
    transaction_id uuid        NOT NULL REFERENCES transactions (id) ON DELETE CASCADE,
    occurred_on    date        NOT NULL,
    -- Signed, in the wallet's minor units, as in the file.
    amount_minor   bigint      NOT NULL,
    note           text        NOT NULL,
    created_at     timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (wallet_id, fingerprint)
);
CREATE INDEX import_refs_transaction_idx ON import_refs (transaction_id);
CREATE INDEX import_refs_household_idx ON import_refs (household_id);

-- +goose Down
DROP TABLE import_refs;
