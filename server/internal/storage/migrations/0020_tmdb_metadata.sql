-- Expands what's persisted from TMDb (see internal/metadata.Client)
-- beyond genre classification alone (migration 0008): each movie's own
-- overview/release date/US certification, plus a first-class tv_shows
-- table so a show's own metadata (era, genre, certification) has one
-- authoritative source instead of being re-derived per episode. See
-- docs/PROJECT_BRIEF.md's metadata section.
--
-- metadata_fetched_at_unix (on both tables) is a fetch-attempted marker,
-- the same NULL-vs-"looked up, nothing came back" convention genre_tags
-- (migration 0008) and ratings_checked_at_unix (migration 0010, now
-- removed) already use — distinct from overview/dates being NULL
-- because nothing was found.
--
-- release_year (existing, untouched) still drives internal/lineup's era
-- rules; release_date is added purely for display/completeness (the
-- 2009-12-31/2010-01-01 era boundary is year-aligned, so the existing
-- integer year is already exact for that comparison).

-- +goose Up
CREATE TABLE tv_shows (
    id                       INTEGER PRIMARY KEY AUTOINCREMENT,
    tmdb_id                  INTEGER NOT NULL UNIQUE,
    title                    TEXT NOT NULL,
    overview                 TEXT,
    first_air_date           TEXT,
    last_air_date            TEXT,
    content_rating           TEXT,
    genre_tags               TEXT,
    metadata_fetched_at_unix INTEGER,
    created_at_unix          INTEGER NOT NULL DEFAULT (unixepoch()),
    updated_at_unix          INTEGER NOT NULL DEFAULT (unixepoch())
);

ALTER TABLE identified_items ADD COLUMN overview TEXT;
ALTER TABLE identified_items ADD COLUMN release_date TEXT;
ALTER TABLE identified_items ADD COLUMN episode_air_date TEXT;
ALTER TABLE identified_items ADD COLUMN content_rating TEXT;
ALTER TABLE identified_items ADD COLUMN tmdb_id INTEGER;
ALTER TABLE identified_items ADD COLUMN show_id INTEGER REFERENCES tv_shows(id);
ALTER TABLE identified_items ADD COLUMN metadata_fetched_at_unix INTEGER;

-- +goose Down
ALTER TABLE identified_items DROP COLUMN metadata_fetched_at_unix;
ALTER TABLE identified_items DROP COLUMN show_id;
ALTER TABLE identified_items DROP COLUMN tmdb_id;
ALTER TABLE identified_items DROP COLUMN content_rating;
ALTER TABLE identified_items DROP COLUMN episode_air_date;
ALTER TABLE identified_items DROP COLUMN release_date;
ALTER TABLE identified_items DROP COLUMN overview;
DROP TABLE tv_shows;
