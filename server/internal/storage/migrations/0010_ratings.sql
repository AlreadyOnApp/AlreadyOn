-- Adds aggregated ratings (OMDb — see internal/metadata.OMDbClient,
-- which re-exposes IMDb/Rotten Tomatoes/Metacritic in one call, since
-- neither IMDb nor Rotten Tomatoes publishes a public API of its own)
-- to identified_items. ratings_checked_at_unix distinguishes "never
-- looked up" (NULL) from "looked up, nothing came back" (non-NULL with
-- all three rating columns still NULL) — the three rating columns alone
-- can't tell those apart, the same problem genre_tags' NULL-vs-"[]"
-- convention (migration 0008) solves for classification.

-- +goose Up
ALTER TABLE identified_items ADD COLUMN imdb_rating TEXT;
ALTER TABLE identified_items ADD COLUMN rotten_tomatoes_rating TEXT;
ALTER TABLE identified_items ADD COLUMN metacritic_rating TEXT;
ALTER TABLE identified_items ADD COLUMN ratings_checked_at_unix INTEGER;

-- +goose Down
ALTER TABLE identified_items DROP COLUMN ratings_checked_at_unix;
ALTER TABLE identified_items DROP COLUMN metacritic_rating;
ALTER TABLE identified_items DROP COLUMN rotten_tomatoes_rating;
ALTER TABLE identified_items DROP COLUMN imdb_rating;
