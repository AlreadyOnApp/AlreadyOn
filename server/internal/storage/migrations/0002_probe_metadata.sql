-- Adds ffprobe result columns to media_files (issue #14), deferred out
-- of 0001_init.sql on purpose so this schema addition stays scoped to
-- the ticket that actually needed it.

-- +goose Up
ALTER TABLE media_files ADD COLUMN duration_seconds REAL;
ALTER TABLE media_files ADD COLUMN container TEXT;
ALTER TABLE media_files ADD COLUMN probe_json TEXT;
ALTER TABLE media_files ADD COLUMN probed_at_unix INTEGER;

-- +goose Down
ALTER TABLE media_files DROP COLUMN probed_at_unix;
ALTER TABLE media_files DROP COLUMN probe_json;
ALTER TABLE media_files DROP COLUMN container;
ALTER TABLE media_files DROP COLUMN duration_seconds;
