-- Admin-editable metadata provider credentials (issue #77), as an
-- alternative to the ALREADYON_TMDB_API_KEY / ALREADYON_OMDB_API_KEY
-- environment variables — main.go prefers whichever of these is set,
-- falling back to the env var. Singleton row, same pattern as
-- commercial_settings (migration 0007).

-- +goose Up
CREATE TABLE metadata_settings (
    id              INTEGER PRIMARY KEY CHECK (id = 1),
    tmdb_api_key    TEXT,
    omdb_api_key    TEXT,
    updated_at_unix INTEGER NOT NULL DEFAULT (unixepoch())
);

INSERT INTO metadata_settings (id) VALUES (1);

-- +goose Down
DROP TABLE metadata_settings;
