-- +goose Up
UPDATE metadata_settings SET keyless_enabled = 0 WHERE tmdb_api_key IS NOT NULL AND keyless_enabled = 1;
-- +goose Down
SELECT 1;
