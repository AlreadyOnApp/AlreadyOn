-- +goose Up
CREATE TABLE server_disk_metrics (
 at_unix INTEGER NOT NULL REFERENCES server_metrics(at_unix) ON DELETE CASCADE,
 name TEXT NOT NULL,
 scope TEXT NOT NULL,
 utilization REAL,
 PRIMARY KEY (at_unix, name, scope)
);
-- +goose Down
DROP TABLE server_disk_metrics;
