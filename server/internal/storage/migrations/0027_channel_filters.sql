-- Filter rule for a custom channel (#123): which library content it airs.
-- A channel with no row here keeps its hand-managed lineup. genres is a
-- JSON array of genre tags ('[]' = any genre). Built-in default channels
-- (100-199) use the rules in internal/lineup/categories.go instead.

-- +goose Up
CREATE TABLE channel_filters (
    channel_id      INTEGER PRIMARY KEY REFERENCES channels (id) ON DELETE CASCADE,
    audience_tier   TEXT NOT NULL CHECK (audience_tier IN ('young_child', 'teen', 'adult')),
    genres          TEXT NOT NULL DEFAULT '[]',
    media_type      TEXT NOT NULL DEFAULT 'both' CHECK (media_type IN ('tv', 'movie', 'both')),
    era             TEXT NOT NULL DEFAULT 'both' CHECK (era IN ('modern', 'nostalgia', 'both')),
    title_contains  TEXT NOT NULL DEFAULT '',
    updated_at_unix INTEGER NOT NULL DEFAULT (unixepoch())
);

-- +goose Down
DROP TABLE channel_filters;
