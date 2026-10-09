-- When the library was last scanned and the schedules last regenerated, by any
-- trigger (manual button, startup, polling, hourly maintenance). Singleton row.

-- +goose Up
CREATE TABLE refresh_times (
    id                 INTEGER PRIMARY KEY CHECK (id = 1),
    media_at_unix      INTEGER NOT NULL DEFAULT 0,
    schedule_at_unix   INTEGER NOT NULL DEFAULT 0
);
INSERT INTO refresh_times (id) VALUES (1);

-- +goose Down
DROP TABLE refresh_times;
