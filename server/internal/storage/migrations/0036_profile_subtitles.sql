-- Per-profile subtitle defaults. NULL means "use the client default": off,
-- and English when the viewer switches them on.

-- +goose Up
ALTER TABLE profiles ADD COLUMN subtitles_enabled INTEGER CHECK (subtitles_enabled IN (0, 1));
ALTER TABLE profiles ADD COLUMN subtitle_language TEXT;

-- +goose Down
ALTER TABLE profiles DROP COLUMN subtitle_language;
ALTER TABLE profiles DROP COLUMN subtitles_enabled;
