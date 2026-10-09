-- +goose Up
-- Retire saved appearance choices and remove obsolete rule dimensions while
-- preserving profiles, channels, schedules, and the remaining shared definitions.
ALTER TABLE profiles DROP COLUMN theme;
UPDATE channel_filters SET extra=json_remove(extra,'$.collectionIds');
CREATE TABLE channel_filter_options_replacement (
 kind TEXT NOT NULL CHECK(kind IN ('age','era')),
 id TEXT NOT NULL,
 definition TEXT NOT NULL,
 position INTEGER NOT NULL DEFAULT 1000,
 default_position INTEGER NOT NULL DEFAULT 1000,
 PRIMARY KEY(kind,id)
);
INSERT INTO channel_filter_options_replacement
 SELECT kind,id,json_remove(definition,'$.members'),position,default_position
 FROM channel_filter_options WHERE kind IN ('age','era');
DROP TABLE channel_filter_options;
ALTER TABLE channel_filter_options_replacement RENAME TO channel_filter_options;

-- +goose Down
-- Restores the previous schema only; removed preferences cannot be recovered.
ALTER TABLE profiles ADD COLUMN theme TEXT;
CREATE TABLE channel_filter_options_previous (
 kind TEXT NOT NULL CHECK(kind IN ('age','era','collection')),
 id TEXT NOT NULL,
 definition TEXT NOT NULL,
 position INTEGER NOT NULL DEFAULT 1000,
 default_position INTEGER NOT NULL DEFAULT 1000,
 PRIMARY KEY(kind,id)
);
INSERT INTO channel_filter_options_previous SELECT * FROM channel_filter_options;
DROP TABLE channel_filter_options;
ALTER TABLE channel_filter_options_previous RENAME TO channel_filter_options;
