-- name: CreateSession :exec
INSERT INTO sessions (id, user_id, token_hash, user_agent, expires_at)
VALUES (@id, @user_id, @token_hash, @user_agent, @expires_at);

-- name: GetSessionByTokenHash :one
-- Joins the membership (as GetActiveMembership picks it), so leaving a household takes effect on
-- the next request. A user with no household gets no row.
SELECT s.id,
       s.user_id,
       s.expires_at,
       s.last_seen_at,
       m.household_id,
       m.role
FROM sessions s
JOIN users u ON u.id = s.user_id
JOIN LATERAL (SELECT hm.household_id, hm.role
              FROM household_members hm
              WHERE hm.user_id = s.user_id
              ORDER BY (hm.household_id = u.active_household_id) IS TRUE DESC, hm.created_at
              LIMIT 1) m ON true
WHERE s.token_hash = $1
  AND s.expires_at > now();

-- name: TouchSession :exec
UPDATE sessions SET last_seen_at = now(), expires_at = @expires_at WHERE id = @id;

-- name: DeleteSessionByTokenHash :exec
DELETE FROM sessions WHERE token_hash = $1;

-- name: DeleteExpiredSessions :execrows
DELETE FROM sessions WHERE expires_at <= now();
