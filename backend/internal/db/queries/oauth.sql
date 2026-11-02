-- name: CreateOAuthClient :one
INSERT INTO oauth_clients (id, name, redirect_uris)
VALUES (@id, @name, @redirect_uris::text[])
RETURNING *;

-- name: GetOAuthClient :one
SELECT * FROM oauth_clients WHERE id = $1;

-- name: CreateOAuthRequest :exec
INSERT INTO oauth_requests (id, client_id, redirect_uri, code_challenge, scope, resource, state, expires_at)
VALUES (@id, @client_id, @redirect_uri, @code_challenge, @scope, @resource, @state, @expires_at);

-- name: GetPendingOAuthRequest :one
-- A request still waiting for the user's decision.
SELECT r.*, c.name AS client_name
FROM oauth_requests r
JOIN oauth_clients c ON c.id = r.client_id
WHERE r.id = $1
  AND r.code_hash IS NULL
  AND r.expires_at > now();

-- name: ApproveOAuthRequest :one
UPDATE oauth_requests
SET user_id = @user_id, code_hash = @code_hash, expires_at = @expires_at
WHERE id = @id
  AND code_hash IS NULL
  AND expires_at > now()
RETURNING *;

-- name: DeleteOAuthRequest :one
DELETE FROM oauth_requests
WHERE id = $1
  AND code_hash IS NULL
  AND expires_at > now()
RETURNING *;

-- name: ConsumeOAuthCode :one
-- Codes are single use: the exchange deletes the request.
DELETE FROM oauth_requests
WHERE code_hash = $1
  AND expires_at > now()
RETURNING *;

-- name: CreateOAuthGrant :exec
INSERT INTO oauth_grants (id, client_id, user_id, scope, resource)
VALUES (@id, @client_id, @user_id, @scope, @resource);

-- name: CreateOAuthToken :exec
INSERT INTO oauth_tokens (token_hash, grant_id, kind, expires_at)
VALUES (@token_hash, @grant_id, @kind, @expires_at);

-- name: GetOAuthAccess :one
-- Resolves an access token to its grant and the user's household (as sessions do).
SELECT g.id AS grant_id,
       g.client_id,
       g.user_id,
       g.scope,
       g.resource,
       g.last_used_at,
       m.household_id,
       m.role
FROM oauth_tokens t
JOIN oauth_grants g ON g.id = t.grant_id
JOIN users u ON u.id = g.user_id
JOIN LATERAL (SELECT hm.household_id, hm.role
              FROM household_members hm
              WHERE hm.user_id = g.user_id
              ORDER BY (hm.household_id = u.active_household_id) IS TRUE DESC, hm.created_at
              LIMIT 1) m ON true
WHERE t.token_hash = $1
  AND t.kind = 'access'
  AND t.expires_at > now()
  AND g.revoked_at IS NULL;

-- name: UseOAuthRefreshToken :one
-- Marks a refresh token used. No row back means it doesn't exist, expired, was revoked or was
-- already used (see GetOAuthRefreshGrant).
UPDATE oauth_tokens t
SET used_at = now()
FROM oauth_grants g
WHERE t.token_hash = $1
  AND t.kind = 'refresh'
  AND t.used_at IS NULL
  AND t.expires_at > now()
  AND g.id = t.grant_id
  AND g.revoked_at IS NULL
RETURNING g.id AS grant_id, g.client_id, g.scope, g.resource;

-- name: GetOAuthRefreshGrant :one
-- The grant behind any refresh token, used or not, to revoke it on reuse.
SELECT grant_id FROM oauth_tokens WHERE token_hash = $1 AND kind = 'refresh';

-- name: TouchOAuthGrant :exec
UPDATE oauth_grants SET last_used_at = now() WHERE id = $1;

-- name: RevokeOAuthGrant :execrows
UPDATE oauth_grants SET revoked_at = now() WHERE id = @id AND revoked_at IS NULL;

-- name: RevokeOAuthGrantForUser :execrows
UPDATE oauth_grants SET revoked_at = now()
WHERE id = @id AND user_id = @user_id AND revoked_at IS NULL;

-- name: GetOAuthTokenGrant :one
-- The grant and client behind any token, for revocation.
SELECT g.id AS grant_id, g.client_id
FROM oauth_tokens t
JOIN oauth_grants g ON g.id = t.grant_id
WHERE t.token_hash = $1;

-- name: ListOAuthGrants :many
SELECT g.id, g.scope, g.created_at, g.last_used_at, c.name AS client_name, c.redirect_uris
FROM oauth_grants g
JOIN oauth_clients c ON c.id = g.client_id
WHERE g.user_id = $1
  AND g.revoked_at IS NULL
ORDER BY g.created_at DESC;

-- name: DeleteExpiredOAuth :one
WITH requests AS (
    DELETE FROM oauth_requests WHERE expires_at <= now() RETURNING 1
), tokens AS (
    DELETE FROM oauth_tokens WHERE expires_at <= now() RETURNING 1
), grants AS (
    -- Revoked grants, and grants whose tokens have all expired, are of no further use.
    DELETE FROM oauth_grants g
    WHERE g.revoked_at IS NOT NULL
       OR NOT EXISTS (SELECT 1 FROM oauth_tokens t WHERE t.grant_id = g.id AND t.expires_at > now())
          AND g.created_at < now() - interval '1 day'
    RETURNING 1
), clients AS (
    -- Registrations that never led to a grant.
    DELETE FROM oauth_clients c
    WHERE c.created_at < now() - interval '30 days'
      AND NOT EXISTS (SELECT 1 FROM oauth_grants g WHERE g.client_id = c.id)
      AND NOT EXISTS (SELECT 1 FROM oauth_requests r WHERE r.client_id = c.id AND r.expires_at > now())
    RETURNING 1
)
SELECT ((SELECT count(*) FROM requests) + (SELECT count(*) FROM tokens) + (SELECT count(*) FROM grants)
    + (SELECT count(*) FROM clients))::bigint AS deleted;
