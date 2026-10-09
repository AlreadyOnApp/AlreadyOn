-- TMDb key and the no-key metadata fallback become admin-only settings
-- (Epic #113): the ALREADYON_TMDB_API_KEY environment mirror is dropped and
-- the keyless fallback (formerly ALREADYON_KEYLESS_METADATA) becomes a
-- database flag, off by default.

-- +goose Up
-- Keep a key that was only ever supplied through the environment.
UPDATE metadata_settings SET tmdb_api_key = tmdb_api_key_env
    WHERE tmdb_api_key IS NULL AND tmdb_api_key_env IS NOT NULL;
ALTER TABLE metadata_settings DROP COLUMN tmdb_api_key_env;
ALTER TABLE metadata_settings ADD COLUMN keyless_enabled INTEGER NOT NULL DEFAULT 0 CHECK (keyless_enabled IN (0, 1));

-- +goose Down
ALTER TABLE metadata_settings DROP COLUMN keyless_enabled;
ALTER TABLE metadata_settings ADD COLUMN tmdb_api_key_env TEXT;
