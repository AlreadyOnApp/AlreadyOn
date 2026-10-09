-- Initial schema for issue #24: commercial break insertion. A single
-- settings row (id is fixed at 1, enforced by the CHECK) — there's only
-- ever one commercial-break configuration for the whole server, not
-- one per channel.

-- +goose Up
CREATE TABLE commercial_settings (
    id              INTEGER PRIMARY KEY CHECK (id = 1),
    enabled         INTEGER NOT NULL DEFAULT 0,
    frequency       TEXT NOT NULL DEFAULT 'fill-end-of-slot'
                        CHECK (frequency IN ('fill-end-of-slot', 'every-15', 'every-30', 'every-60')),
    updated_at_unix INTEGER NOT NULL DEFAULT (unixepoch())
);

INSERT INTO commercial_settings (id, enabled, frequency) VALUES (1, 0, 'fill-end-of-slot');

-- +goose Down
DROP TABLE commercial_settings;
