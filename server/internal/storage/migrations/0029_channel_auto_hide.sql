-- Whether channel visibility follows content automatically (#hide-empty).
-- Set by the Channels tab's Hide Empty button and cleared by any manual
-- show/hide (Enable All, Disable All, or a single channel's toggle). While
-- on, every refresh hides channels with no shows or movies and reveals ones
-- that have gained some. Singleton row.

-- +goose Up
CREATE TABLE channel_settings (
    id              INTEGER PRIMARY KEY CHECK (id = 1),
    auto_hide_empty INTEGER NOT NULL DEFAULT 0 CHECK (auto_hide_empty IN (0, 1)),
    updated_at_unix INTEGER NOT NULL DEFAULT (unixepoch())
);
INSERT INTO channel_settings (id) VALUES (1);

-- +goose Down
DROP TABLE channel_settings;
