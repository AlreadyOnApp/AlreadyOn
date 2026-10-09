-- +goose Up
ALTER TABLE browser_sessions ADD COLUMN last_active_at_unix INTEGER;
-- +goose Down
ALTER TABLE browser_sessions DROP COLUMN last_active_at_unix;
