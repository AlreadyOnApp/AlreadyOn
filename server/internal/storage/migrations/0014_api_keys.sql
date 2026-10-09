-- Generated credentials for authenticating directly to the server's
-- HTTP API (Server Settings > API tab, see internal/auth.GenerateAPIKey)
-- — independent of the local admin account/session system
-- (internal/auth's Account/Session, Epic 5, not yet implemented). Only
-- a SHA-256 hash and a short display prefix are stored; the raw key is
-- shown to the admin exactly once, at creation, and never persisted or
-- retrievable again — same "never round-trip a secret" convention as
-- migration 0011's metadata_settings.
--
-- Note: nothing enforces these yet — internal/auth.RequireAuth is
-- still a no-op passthrough (see its own TODO). This table only
-- supports generating/managing keys for now, not authenticating with
-- them.

-- +goose Up
CREATE TABLE api_keys (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    name            TEXT NOT NULL,
    key_hash        TEXT NOT NULL UNIQUE,
    key_prefix      TEXT NOT NULL,
    created_at_unix INTEGER NOT NULL DEFAULT (unixepoch())
);

-- +goose Down
DROP TABLE api_keys;
