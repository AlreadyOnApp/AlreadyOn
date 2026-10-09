-- How often the background library scanner re-checks configured media
-- roots for new content (see internal/library.GetPollSettings and
-- main.go's polling loop) — separate from the one-time startup scan.
-- Singleton row, same pattern as commercial_settings/metadata_settings.

-- +goose Up
CREATE TABLE library_settings (
    id                    INTEGER PRIMARY KEY CHECK (id = 1),
    poll_interval_minutes INTEGER NOT NULL DEFAULT 15
                              CHECK (poll_interval_minutes IN (15, 30, 60, 120, 240, 480, 720, 1440)),
    updated_at_unix       INTEGER NOT NULL DEFAULT (unixepoch())
);
INSERT INTO library_settings (id) VALUES (1);

-- +goose Down
DROP TABLE library_settings;
