-- The transcode thread count found by the server's own benchmark (#143 follow-up),
-- kept so a restart starts with it instead of the default until it is measured again.

-- +goose Up
ALTER TABLE server_settings ADD COLUMN tuned_threads INTEGER NOT NULL DEFAULT 0 CHECK (tuned_threads BETWEEN 0 AND 64);
ALTER TABLE server_settings ADD COLUMN tuned_at_unix INTEGER NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE server_settings DROP COLUMN tuned_at_unix;
ALTER TABLE server_settings DROP COLUMN tuned_threads;
