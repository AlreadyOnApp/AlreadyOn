-- Replaces the old 4-way Frequency (fill-end-of-slot/every-15/every-30/
-- every-60) with two modes and per-media-type break sizes. Grid
-- alignment (the old fill-end-of-slot behavior) is retired entirely —
-- see internal/schedule/expand.go's rewrite. A settings singleton, same
-- pattern as before; dropping and recreating rather than ALTERing is
-- fine here (alpha phase, no real user data in a config-only table).

-- +goose Up
DROP TABLE commercial_settings;
CREATE TABLE commercial_settings (
    id                          INTEGER PRIMARY KEY CHECK (id = 1),
    enabled                     INTEGER NOT NULL DEFAULT 0,
    mode                        TEXT NOT NULL DEFAULT 'between-only'
                                    CHECK (mode IN ('between-only', 'interval')),
    interval_minutes            INTEGER NOT NULL DEFAULT 30
                                    CHECK (interval_minutes IN (15, 30, 60)),
    tv_commercials_per_break    INTEGER NOT NULL DEFAULT 2
                                    CHECK (tv_commercials_per_break BETWEEN 1 AND 5),
    movie_commercials_per_break INTEGER NOT NULL DEFAULT 5
                                    CHECK (movie_commercials_per_break BETWEEN 1 AND 5),
    updated_at_unix             INTEGER NOT NULL DEFAULT (unixepoch())
);
INSERT INTO commercial_settings (id) VALUES (1);

-- +goose Down
DROP TABLE commercial_settings;
CREATE TABLE commercial_settings (
    id              INTEGER PRIMARY KEY CHECK (id = 1),
    enabled         INTEGER NOT NULL DEFAULT 0,
    frequency       TEXT NOT NULL DEFAULT 'fill-end-of-slot'
                        CHECK (frequency IN ('fill-end-of-slot', 'every-15', 'every-30', 'every-60')),
    updated_at_unix INTEGER NOT NULL DEFAULT (unixepoch())
);
INSERT INTO commercial_settings (id, enabled, frequency) VALUES (1, 0, 'fill-end-of-slot');
