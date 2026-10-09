-- +goose Up
ALTER TABLE commercial_settings ADD COLUMN volume_offset_db REAL NOT NULL DEFAULT 0 CHECK(volume_offset_db BETWEEN -30 AND 12);
ALTER TABLE profiles ADD COLUMN audio_language TEXT;

-- +goose Down
ALTER TABLE profiles DROP COLUMN audio_language;
ALTER TABLE commercial_settings DROP COLUMN volume_offset_db;
