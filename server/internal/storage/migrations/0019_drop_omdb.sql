-- Drops OMDb-sourced columns (see migrations 0010/0011/0013): the
-- project has moved to TMDb as its sole metadata provider (see
-- docs/PROJECT_BRIEF.md), so the separate OMDb ratings integration and
-- its credential columns are removed entirely.

-- +goose Up
ALTER TABLE identified_items DROP COLUMN imdb_rating;
ALTER TABLE identified_items DROP COLUMN rotten_tomatoes_rating;
ALTER TABLE identified_items DROP COLUMN metacritic_rating;
ALTER TABLE identified_items DROP COLUMN ratings_checked_at_unix;
ALTER TABLE metadata_settings DROP COLUMN omdb_api_key;
ALTER TABLE metadata_settings DROP COLUMN omdb_api_key_env;

-- +goose Down
ALTER TABLE identified_items ADD COLUMN imdb_rating TEXT;
ALTER TABLE identified_items ADD COLUMN rotten_tomatoes_rating TEXT;
ALTER TABLE identified_items ADD COLUMN metacritic_rating TEXT;
ALTER TABLE identified_items ADD COLUMN ratings_checked_at_unix INTEGER;
ALTER TABLE metadata_settings ADD COLUMN omdb_api_key TEXT;
ALTER TABLE metadata_settings ADD COLUMN omdb_api_key_env TEXT;
