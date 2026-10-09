-- Initial schema (Epic 11 / issue #12): media files and identified items.
--
-- Scope is deliberately minimal. Columns needed by later tasks (ffprobe
-- results in #14, scan status in #18, manual-correction tracking in #17)
-- land in their own migrations instead of being guessed at here.

-- +goose Up
CREATE TABLE media_files (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    root_kind       TEXT NOT NULL CHECK (root_kind IN ('media', 'commercial')),
    path            TEXT NOT NULL UNIQUE,
    size_bytes      INTEGER NOT NULL,
    modified_unix   INTEGER NOT NULL,
    first_seen_unix INTEGER NOT NULL,
    last_seen_unix  INTEGER NOT NULL,
    created_at_unix INTEGER NOT NULL DEFAULT (unixepoch()),
    updated_at_unix INTEGER NOT NULL DEFAULT (unixepoch())
);

CREATE INDEX idx_media_files_root_kind ON media_files (root_kind);

CREATE TABLE identified_items (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    media_file_id   INTEGER NOT NULL UNIQUE REFERENCES media_files (id) ON DELETE CASCADE,
    media_type      TEXT NOT NULL CHECK (media_type IN ('movie', 'episode')),
    title           TEXT NOT NULL,
    series_title    TEXT,
    season_number   INTEGER,
    episode_number  INTEGER,
    release_year    INTEGER,
    created_at_unix INTEGER NOT NULL DEFAULT (unixepoch()),
    updated_at_unix INTEGER NOT NULL DEFAULT (unixepoch())
);

-- +goose Down
DROP TABLE identified_items;
DROP TABLE media_files;
