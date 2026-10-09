-- Channel overrides inherit global enablement and all folders until saved.
-- Folder paths reference the configured list; removed paths fall back at read time.
-- +goose Up
CREATE TABLE channel_commercial_settings (
    channel_id INTEGER PRIMARY KEY REFERENCES channels(id) ON DELETE CASCADE,
    enabled INTEGER CHECK (enabled IN (0, 1)),
    folders_json TEXT NOT NULL DEFAULT '[]'
);
-- +goose Down
DROP TABLE channel_commercial_settings;
