-- The background scan now defaults to hourly instead of every 15 minutes.
-- SQLite can't change a column default in place, so the table is rebuilt.
-- A stored value of 15 can't be told apart from the old default, so it is
-- moved to 60 too; pick 15 again on the Media tab if you want it.

-- +goose Up
CREATE TABLE library_settings_new (
    id                    INTEGER PRIMARY KEY CHECK (id = 1),
    poll_interval_minutes INTEGER NOT NULL DEFAULT 60
                              CHECK (poll_interval_minutes IN (15, 30, 60, 120, 240, 480, 720, 1440)),
    updated_at_unix       INTEGER NOT NULL DEFAULT (unixepoch())
);
INSERT INTO library_settings_new (id, poll_interval_minutes, updated_at_unix)
    SELECT id, CASE WHEN poll_interval_minutes = 15 THEN 60 ELSE poll_interval_minutes END, updated_at_unix
    FROM library_settings;
DROP TABLE library_settings;
ALTER TABLE library_settings_new RENAME TO library_settings;

-- +goose Down
CREATE TABLE library_settings_old (
    id                    INTEGER PRIMARY KEY CHECK (id = 1),
    poll_interval_minutes INTEGER NOT NULL DEFAULT 15
                              CHECK (poll_interval_minutes IN (15, 30, 60, 120, 240, 480, 720, 1440)),
    updated_at_unix       INTEGER NOT NULL DEFAULT (unixepoch())
);
INSERT INTO library_settings_old SELECT id, poll_interval_minutes, updated_at_unix FROM library_settings;
DROP TABLE library_settings;
ALTER TABLE library_settings_old RENAME TO library_settings;
