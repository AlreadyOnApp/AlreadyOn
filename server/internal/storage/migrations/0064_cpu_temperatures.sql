-- +goose Up
-- Sensor names retain per-core identity across dashboard averaging windows.
CREATE TABLE server_cpu_temperatures (
 at_unix INTEGER NOT NULL REFERENCES server_metrics(at_unix) ON DELETE CASCADE,
 name TEXT NOT NULL,
 celsius REAL,
 PRIMARY KEY (at_unix, name)
);
-- +goose Down
DROP TABLE server_cpu_temperatures;
