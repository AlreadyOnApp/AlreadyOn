-- Per-profile choice to show the channel picture in the guide's left column.
-- NULL means the default: shown.

-- +goose Up
ALTER TABLE profiles ADD COLUMN guide_logos INTEGER CHECK (guide_logos IN (0, 1));

-- +goose Down
ALTER TABLE profiles DROP COLUMN guide_logos;
