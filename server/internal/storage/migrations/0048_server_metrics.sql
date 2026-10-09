-- +goose Up
CREATE TABLE server_metrics (
 at_unix INTEGER PRIMARY KEY,
 cpu REAL,
 gpu REAL,
 ram REAL,
 gpu_source TEXT NOT NULL DEFAULT ''
);
-- +goose Down
DROP TABLE server_metrics;
