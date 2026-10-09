-- +goose Up
-- Opt-in second (360p) rendition on large conversions; see issue #141.
ALTER TABLE playback_settings ADD COLUMN ladder INTEGER NOT NULL DEFAULT 0 CHECK (ladder IN (0, 1));

-- +goose Down
ALTER TABLE playback_settings DROP COLUMN ladder;
