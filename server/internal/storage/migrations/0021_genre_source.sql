-- Records which provider produced identified_items.genre_tags (#112):
-- 'tmdb', or 'keyless' (TVmaze/Wikidata fallback when no TMDb key is
-- configured). NULL on existing rows means TMDb, the only provider
-- before this migration. Keyless rows are re-classified once a TMDb key
-- is added.

-- +goose Up
ALTER TABLE identified_items ADD COLUMN genre_source TEXT;

-- +goose Down
ALTER TABLE identified_items DROP COLUMN genre_source;
