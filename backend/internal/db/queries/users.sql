-- name: GetUser :one
SELECT * FROM users WHERE id = $1;

-- name: GetUserByGoogleSub :one
SELECT * FROM users WHERE google_sub = $1;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE lower(email) = lower(@email::text);

-- name: CreateUser :one
INSERT INTO users (id, google_sub, email, name, avatar_url)
VALUES (@id, @google_sub, @email, @name, @avatar_url)
RETURNING *;

-- name: UpdateUserFromGoogle :one
UPDATE users
SET email = @email, name = @name, avatar_url = @avatar_url, updated_at = now()
WHERE id = @id
RETURNING *;

-- name: UpdateUserProfile :one
UPDATE users
SET name       = coalesce(sqlc.narg(name), name),
    updated_at = now()
WHERE id = @id
RETURNING *;

-- name: CreateHousehold :one
INSERT INTO households (id, name, base_currency) VALUES (@id, @name, @base_currency) RETURNING *;

-- name: AddHouseholdMember :exec
INSERT INTO household_members (household_id, user_id, role) VALUES (@household_id, @user_id, @role);

-- name: GetActiveMembership :one
-- The household a user works in: their active one, or their oldest membership when that is unset
-- or they left it. No row means they belong to no household.
SELECT hm.household_id, hm.role
FROM users u
JOIN household_members hm ON hm.user_id = u.id
WHERE u.id = $1
ORDER BY (hm.household_id = u.active_household_id) IS TRUE DESC, hm.created_at
LIMIT 1;

-- name: ListCurrencies :many
SELECT * FROM currencies ORDER BY code;

-- name: GetCurrency :one
SELECT * FROM currencies WHERE code = $1;
