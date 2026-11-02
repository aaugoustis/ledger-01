-- +goose Up
-- Household members (REQ-households): the base currency moves to the household, each user has an
-- active household, and owners invite people by email.
ALTER TABLE households
    ADD COLUMN base_currency char(3) NOT NULL DEFAULT 'USD' REFERENCES currencies (code);

UPDATE households h
SET base_currency = u.base_currency
FROM household_members m
JOIN users u ON u.id = m.user_id
WHERE m.household_id = h.id AND m.role = 'owner';

-- users.base_currency is no longer read. It stays until the next migration so the previous
-- release keeps working while a deploy rolls out.
ALTER TABLE users ALTER COLUMN base_currency DROP NOT NULL;

-- The household a user's sessions and OAuth tokens work in. NULL or a household they left falls
-- back to their oldest membership.
ALTER TABLE users
    ADD COLUMN active_household_id uuid REFERENCES households (id) ON DELETE SET NULL;

UPDATE users u
SET active_household_id = (SELECT m.household_id FROM household_members m
                           WHERE m.user_id = u.id ORDER BY m.created_at LIMIT 1);

CREATE UNIQUE INDEX household_members_one_owner ON household_members (household_id) WHERE role = 'owner';

-- A pending invite, accepted on that email's next sign-in. The email must also be allowlisted.
CREATE TABLE household_invites (
    household_id uuid        NOT NULL REFERENCES households (id) ON DELETE CASCADE,
    email        text        NOT NULL CHECK (char_length(email) BETWEEN 3 AND 254),
    invited_by   uuid        REFERENCES users (id) ON DELETE SET NULL,
    created_at   timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX household_invites_key ON household_invites (household_id, lower(email));
CREATE INDEX household_invites_email_idx ON household_invites (lower(email));

-- The by-person filter.
CREATE INDEX transactions_created_by_idx ON transactions (household_id, created_by);

-- +goose Down
DROP INDEX transactions_created_by_idx;
DROP TABLE household_invites;
DROP INDEX household_members_one_owner;
ALTER TABLE users DROP COLUMN active_household_id;
UPDATE users u
SET base_currency = h.base_currency
FROM household_members m
JOIN households h ON h.id = m.household_id
WHERE m.user_id = u.id AND m.role = 'owner';
UPDATE users SET base_currency = 'USD' WHERE base_currency IS NULL;
ALTER TABLE users ALTER COLUMN base_currency SET NOT NULL;
ALTER TABLE households DROP COLUMN base_currency;
