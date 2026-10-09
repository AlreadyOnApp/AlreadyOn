-- Admin-configurable content folders and scan exclusions (#114). Each
-- column holds a JSON array of strings, or NULL meaning "not set by an
-- admin, use the built-in default" (see internal/library.FolderSettings).
-- An empty array is an explicit "none". Singleton row, same pattern as
-- library_settings.

-- +goose Up
CREATE TABLE folder_settings (
    id                 INTEGER PRIMARY KEY CHECK (id = 1),
    tv_paths           TEXT,
    movie_paths        TEXT,
    commercial_paths   TEXT,
    exclusion_patterns TEXT,
    updated_at_unix    INTEGER NOT NULL DEFAULT (unixepoch())
);
INSERT INTO folder_settings (id) VALUES (1);

-- +goose Down
DROP TABLE folder_settings;
