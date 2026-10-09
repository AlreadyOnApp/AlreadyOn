-- Adds a self-referencing alias column to media_files, used only by
-- the dev/test content multiplier (internal/library.GenerateTestAliases,
-- gated by ALREADYON_TEST_CONTENT_MULTIPLIER): a small real library can
-- be "multiplied" into several distinct, randomly-titled shows/movies
-- that all actually play the same underlying file, so channel lineup
-- population has enough distinct series to interleave/test against.
-- NULL (the normal case) means "a real, scanned file"; non-NULL points
-- back at the real file whose bytes this alias actually plays.

-- +goose Up
ALTER TABLE media_files ADD COLUMN source_media_file_id INTEGER REFERENCES media_files (id) ON DELETE CASCADE;

-- +goose Down
ALTER TABLE media_files DROP COLUMN source_media_file_id;
