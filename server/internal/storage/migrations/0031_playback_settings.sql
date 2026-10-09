-- +goose Up
CREATE TABLE playback_settings (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    max_processes INTEGER NOT NULL DEFAULT 2 CHECK (max_processes BETWEEN 1 AND 16),
    max_renditions INTEGER NOT NULL DEFAULT 8 CHECK (max_renditions BETWEEN 1 AND 32),
    max_viewers INTEGER NOT NULL DEFAULT 256 CHECK (max_viewers BETWEEN 1 AND 1024),
    lease_seconds INTEGER NOT NULL DEFAULT 30 CHECK (lease_seconds BETWEEN 10 AND 120)
);
INSERT INTO playback_settings (id) VALUES (1);

-- +goose Down
DROP TABLE playback_settings;
