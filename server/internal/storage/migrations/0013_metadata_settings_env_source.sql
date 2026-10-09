-- Mirrors the ALREADYON_TMDB_API_KEY / ALREADYON_OMDB_API_KEY environment
-- variables into their own columns, separate from the Web-UI-set
-- tmdb_api_key/omdb_api_key columns (migration 0011). Kept in the
-- database (not read from os.Getenv directly by every caller) so any
-- consumer, including HTTP handlers with no access to the process's own
-- environment, resolves the same effective key: the Web UI value if set,
-- else this mirrored env value. See metadata.SyncEnvSettings and
-- Settings.EffectiveTMDbAPIKey/EffectiveOMDbAPIKey.

-- +goose Up
ALTER TABLE metadata_settings ADD COLUMN tmdb_api_key_env TEXT;
ALTER TABLE metadata_settings ADD COLUMN omdb_api_key_env TEXT;

-- +goose Down
ALTER TABLE metadata_settings DROP COLUMN tmdb_api_key_env;
ALTER TABLE metadata_settings DROP COLUMN omdb_api_key_env;
