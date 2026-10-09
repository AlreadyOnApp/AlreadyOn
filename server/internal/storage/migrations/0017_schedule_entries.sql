-- Persisted, committed schedule: replaces computing "what's playing
-- now" on the fly from an endlessly-looping lineup (the old epoch/
-- loopPosition model) with real rows spanning roughly 1 hour in the
-- past through 48 hours ahead (see internal/schedule.Regenerate). Once
-- written, an entry never silently changes underneath a viewer already
-- partway through it — only entries that haven't started yet are ever
-- replaced by a regeneration.

-- +goose Up
CREATE TABLE schedule_entries (
    id                    INTEGER PRIMARY KEY AUTOINCREMENT,
    channel_id            INTEGER NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
    media_file_id         INTEGER NOT NULL,
    kind                  TEXT NOT NULL CHECK (kind IN ('program', 'commercial')),
    source_offset_seconds REAL NOT NULL DEFAULT 0,
    duration_seconds      REAL NOT NULL,
    starts_at_unix        INTEGER NOT NULL,
    ends_at_unix          INTEGER NOT NULL
);
CREATE INDEX idx_schedule_entries_channel_starts ON schedule_entries(channel_id, starts_at_unix);

-- +goose Down
DROP TABLE schedule_entries;
