-- +goose Up
-- One server-wide sign-in (profiles are not accounts). The CHECK keeps it a
-- single row, so account creation is a create-if-absent that cannot race.
CREATE TABLE server_account (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    username TEXT NOT NULL,
    password_hash TEXT NOT NULL,
    created_at_unix INTEGER NOT NULL DEFAULT (unixepoch())
);

-- Only a hash of each session token is stored.
CREATE TABLE browser_sessions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    token_hash TEXT NOT NULL UNIQUE,
    created_at_unix INTEGER NOT NULL,
    expires_at_unix INTEGER NOT NULL,
    user_agent TEXT NOT NULL DEFAULT ''
);
CREATE INDEX browser_sessions_expires ON browser_sessions (expires_at_unix);

-- +goose Down
DROP TABLE browser_sessions;
DROP TABLE server_account;
