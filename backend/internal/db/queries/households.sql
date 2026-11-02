-- Household settings, members and invites (ledger.Household*). Owner-only writes are checked in
-- ledger before these run.

-- name: GetHousehold :one
SELECT * FROM households WHERE id = $1;

-- name: UpdateHousehold :one
UPDATE households
SET name          = coalesce(sqlc.narg(name), name),
    base_currency = coalesce(sqlc.narg(base_currency), base_currency)
WHERE id = @id
RETURNING *;

-- name: ListHouseholdMembers :many
SELECT u.id, u.email, u.name, u.avatar_url, hm.role, hm.created_at
FROM household_members hm
JOIN users u ON u.id = hm.user_id
WHERE hm.household_id = $1
ORDER BY hm.role = 'owner' DESC, hm.created_at;

-- name: ListFormerCreators :many
-- People who entered transactions here but are no longer members (for the by-person filter).
SELECT u.id, u.email, u.name, u.avatar_url
FROM users u
WHERE u.id IN (SELECT DISTINCT t.created_by FROM transactions t
               WHERE t.household_id = @household_id AND t.created_by IS NOT NULL)
  AND NOT EXISTS (SELECT 1 FROM household_members hm
                  WHERE hm.household_id = @household_id AND hm.user_id = u.id)
ORDER BY u.name, u.email;

-- name: ListUserHouseholds :many
SELECT h.id, h.name, hm.role,
       (SELECT count(*) FROM household_members x WHERE x.household_id = h.id)::bigint AS member_count
FROM household_members hm
JOIN households h ON h.id = hm.household_id
WHERE hm.user_id = $1
ORDER BY hm.created_at;

-- name: IsHouseholdMember :one
SELECT EXISTS (SELECT 1 FROM household_members WHERE household_id = @household_id AND user_id = @user_id);

-- name: IsHouseholdMemberEmail :one
SELECT EXISTS (SELECT 1 FROM household_members hm JOIN users u ON u.id = hm.user_id
               WHERE hm.household_id = @household_id AND lower(u.email) = lower(@email::text));

-- name: RemoveHouseholdMember :execrows
-- Owners can't be removed (one owner per household; ownership transfer is out of scope).
DELETE FROM household_members
WHERE household_id = @household_id AND user_id = @user_id AND role = 'member';

-- name: SetActiveHousehold :exec
UPDATE users SET active_household_id = sqlc.narg(household_id), updated_at = now() WHERE id = @id;

-- name: CreateHouseholdInvite :one
INSERT INTO household_invites (household_id, email, invited_by)
VALUES (@household_id, lower(@email::text), @invited_by)
RETURNING *;

-- name: ListHouseholdInvites :many
SELECT * FROM household_invites WHERE household_id = $1 ORDER BY created_at;

-- name: DeleteHouseholdInvite :execrows
DELETE FROM household_invites WHERE household_id = @household_id AND lower(email) = lower(@email::text);

-- name: AcceptHouseholdInvites :many
-- Turns every pending invite for an email into a membership, oldest first. Returns the households
-- joined.
WITH accepted AS (
    DELETE FROM household_invites WHERE lower(email) = lower(@email::text)
    RETURNING household_id, created_at
)
INSERT INTO household_members (household_id, user_id, role, created_at)
SELECT a.household_id, @user_id, 'member', now() + (row_number() OVER (ORDER BY a.created_at)) * interval '1 microsecond'
FROM accepted a
ON CONFLICT (household_id, user_id) DO NOTHING
RETURNING household_id;
