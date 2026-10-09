-- +goose Up
CREATE TABLE loudness_settings (
 id INTEGER PRIMARY KEY CHECK(id=1), enabled INTEGER NOT NULL DEFAULT 0,
 generation INTEGER NOT NULL DEFAULT 0, running INTEGER NOT NULL DEFAULT 0,
 current_file TEXT NOT NULL DEFAULT '', heartbeat INTEGER NOT NULL DEFAULT 0
);
INSERT INTO loudness_settings(id) VALUES(1);
CREATE TABLE loudness_measurements (
 media_file_id INTEGER PRIMARY KEY REFERENCES media_files(id) ON DELETE CASCADE,
 size_bytes INTEGER NOT NULL, modified_unix INTEGER NOT NULL, modified_ns INTEGER NOT NULL,
 stream_index INTEGER NOT NULL, version INTEGER NOT NULL, status TEXT NOT NULL,
 measurement TEXT NOT NULL DEFAULT '{}', error TEXT NOT NULL DEFAULT '',
 analyzed_at INTEGER NOT NULL DEFAULT (unixepoch()), generation INTEGER NOT NULL
);
-- +goose Down
DROP TABLE loudness_measurements;
DROP TABLE loudness_settings;
