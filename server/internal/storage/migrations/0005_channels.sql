-- Initial schema for issue #20: channel CRUD API and data model.

-- +goose Up
CREATE TABLE channels (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    number          INTEGER NOT NULL UNIQUE,
    name            TEXT NOT NULL,
    artwork_ref     TEXT,
    created_at_unix INTEGER NOT NULL DEFAULT (unixepoch()),
    updated_at_unix INTEGER NOT NULL DEFAULT (unixepoch())
);

-- +goose Down
DROP TABLE channels;
