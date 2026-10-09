-- +goose Up
CREATE TABLE keyframe_measurements (
 media_file_id INTEGER PRIMARY KEY REFERENCES media_files(id) ON DELETE CASCADE,
 size_bytes INTEGER NOT NULL, modified_unix INTEGER NOT NULL, version INTEGER NOT NULL,
 status TEXT NOT NULL, max_gap_ms INTEGER NOT NULL DEFAULT 0, error TEXT NOT NULL DEFAULT '',
 analyzed_at INTEGER NOT NULL DEFAULT (unixepoch())
);
-- +goose Down
DROP TABLE keyframe_measurements;
