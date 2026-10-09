-- +goose Up
-- Issue #226: per-core CPU, load/pressure, container memory, disk I/O and
-- capacity replace per-disk busy time and CPU temperatures on the dashboard.
DROP TABLE server_disk_metrics;
DROP TABLE server_cpu_temperatures;
ALTER TABLE server_metrics ADD COLUMN cpu_load REAL;
ALTER TABLE server_metrics ADD COLUMN cpu_pressure REAL;
ALTER TABLE server_metrics ADD COLUMN memory_used_bytes INTEGER;
ALTER TABLE server_metrics ADD COLUMN memory_limit_bytes INTEGER;
ALTER TABLE server_metrics ADD COLUMN memory_limited INTEGER NOT NULL DEFAULT 0;
ALTER TABLE server_metrics ADD COLUMN disk_read_mbps REAL;
ALTER TABLE server_metrics ADD COLUMN disk_write_mbps REAL;
ALTER TABLE server_metrics ADD COLUMN io_pressure REAL;
ALTER TABLE server_metrics ADD COLUMN disk_capacity REAL;
ALTER TABLE server_metrics ADD COLUMN disk_used_bytes INTEGER;
ALTER TABLE server_metrics ADD COLUMN disk_total_bytes INTEGER;
ALTER TABLE server_metrics ADD COLUMN disk_path TEXT NOT NULL DEFAULT '';
ALTER TABLE server_metrics ADD COLUMN disk_scope TEXT NOT NULL DEFAULT '';
CREATE TABLE server_cpu_core_metrics (
 at_unix INTEGER NOT NULL REFERENCES server_metrics(at_unix) ON DELETE CASCADE,
 core INTEGER NOT NULL,
 utilization REAL,
 PRIMARY KEY (at_unix, core)
);
-- +goose Down
DROP TABLE server_cpu_core_metrics;
ALTER TABLE server_metrics DROP COLUMN disk_scope;
ALTER TABLE server_metrics DROP COLUMN disk_path;
ALTER TABLE server_metrics DROP COLUMN disk_total_bytes;
ALTER TABLE server_metrics DROP COLUMN disk_used_bytes;
ALTER TABLE server_metrics DROP COLUMN disk_capacity;
ALTER TABLE server_metrics DROP COLUMN io_pressure;
ALTER TABLE server_metrics DROP COLUMN disk_write_mbps;
ALTER TABLE server_metrics DROP COLUMN disk_read_mbps;
ALTER TABLE server_metrics DROP COLUMN memory_limited;
ALTER TABLE server_metrics DROP COLUMN memory_limit_bytes;
ALTER TABLE server_metrics DROP COLUMN memory_used_bytes;
ALTER TABLE server_metrics DROP COLUMN cpu_pressure;
ALTER TABLE server_metrics DROP COLUMN cpu_load;
CREATE TABLE server_cpu_temperatures (
 at_unix INTEGER NOT NULL REFERENCES server_metrics(at_unix) ON DELETE CASCADE,
 name TEXT NOT NULL,
 celsius REAL,
 PRIMARY KEY (at_unix, name)
);
CREATE TABLE server_disk_metrics (
 at_unix INTEGER NOT NULL REFERENCES server_metrics(at_unix) ON DELETE CASCADE,
 name TEXT NOT NULL,
 scope TEXT NOT NULL,
 utilization REAL,
 PRIMARY KEY (at_unix, name, scope)
);
