-- +goose Up
-- OAuth 2.1 authorization server for MCP clients (such as Claude). Clients register themselves
-- (dynamic client registration), the user approves a request on the consent page, and the client
-- exchanges the one-time code for tokens. Tokens are random; only their sha256 is stored.
CREATE TABLE oauth_clients (
    id            uuid        PRIMARY KEY,
    -- As the client named itself; shown with the redirect host on the consent page.
    name          text        NOT NULL CHECK (char_length(name) BETWEEN 1 AND 100),
    redirect_uris text[]      NOT NULL CHECK (cardinality(redirect_uris) BETWEEN 1 AND 10),
    created_at    timestamptz NOT NULL DEFAULT now()
);

-- A pending authorization: created by /oauth/authorize, approved on the consent page (user_id and
-- code_hash set), consumed by the token exchange (deleted).
CREATE TABLE oauth_requests (
    id             uuid        PRIMARY KEY,
    client_id      uuid        NOT NULL REFERENCES oauth_clients (id) ON DELETE CASCADE,
    redirect_uri   text        NOT NULL,
    code_challenge text        NOT NULL,
    scope          text        NOT NULL,
    resource       text        NOT NULL,
    state          text        NOT NULL,
    user_id        uuid        REFERENCES users (id) ON DELETE CASCADE,
    code_hash      bytea       UNIQUE,
    created_at     timestamptz NOT NULL DEFAULT now(),
    expires_at     timestamptz NOT NULL
);
CREATE INDEX oauth_requests_expires_idx ON oauth_requests (expires_at);

-- A user's consent for one client. Revoking it kills every token issued under it.
CREATE TABLE oauth_grants (
    id           uuid        PRIMARY KEY,
    client_id    uuid        NOT NULL REFERENCES oauth_clients (id) ON DELETE CASCADE,
    user_id      uuid        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    scope        text        NOT NULL,
    resource     text        NOT NULL,
    created_at   timestamptz NOT NULL DEFAULT now(),
    last_used_at timestamptz,
    revoked_at   timestamptz
);
CREATE INDEX oauth_grants_user_idx ON oauth_grants (user_id);

CREATE TABLE oauth_tokens (
    token_hash bytea       PRIMARY KEY,
    grant_id   uuid        NOT NULL REFERENCES oauth_grants (id) ON DELETE CASCADE,
    kind       text        NOT NULL CHECK (kind IN ('access', 'refresh')),
    created_at timestamptz NOT NULL DEFAULT now(),
    expires_at timestamptz NOT NULL,
    -- Refresh tokens rotate: set when exchanged. A second use means the token leaked.
    used_at    timestamptz
);
CREATE INDEX oauth_tokens_grant_idx ON oauth_tokens (grant_id);
CREATE INDEX oauth_tokens_expires_idx ON oauth_tokens (expires_at);

-- +goose Down
DROP TABLE oauth_tokens;
DROP TABLE oauth_grants;
DROP TABLE oauth_requests;
DROP TABLE oauth_clients;
