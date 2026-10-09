-- +goose Up
-- What this server measured about its own encoding speed, shared by every viewer.
CREATE TABLE server_capacity (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    measured_at_unix INTEGER NOT NULL DEFAULT 0,
    encoder TEXT NOT NULL DEFAULT '',
    samples TEXT NOT NULL DEFAULT '[]',
    max_height INTEGER NOT NULL DEFAULT 0,
    max_fps REAL NOT NULL DEFAULT 0,
    processes INTEGER NOT NULL DEFAULT 0,
    processes_at_unix INTEGER NOT NULL DEFAULT 0
);
INSERT INTO server_capacity (id) VALUES (1);
-- Parallel conversions follow the measured capacity unless an administrator picks a number.
ALTER TABLE playback_settings ADD COLUMN auto_processes INTEGER NOT NULL DEFAULT 1 CHECK (auto_processes IN (0, 1));
-- +goose Down
ALTER TABLE playback_settings DROP COLUMN auto_processes;
DROP TABLE server_capacity;
