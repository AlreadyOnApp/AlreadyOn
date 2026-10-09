-- Adds per-file status tracking to media_files (issue #18), deferred
-- out of 0001_init.sql on purpose so this schema addition stays scoped
-- to the ticket that actually needed it.

-- +goose Up
ALTER TABLE media_files ADD COLUMN status TEXT NOT NULL DEFAULT 'ok' CHECK (status IN ('ok', 'missing', 'unreadable', 'corrupt'));

-- +goose Down
ALTER TABLE media_files DROP COLUMN status;
