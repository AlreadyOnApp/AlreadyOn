-- Viewer profiles and their per-profile preferences (#117), previously
-- kept only in each browser's localStorage. A NULL preference means "use
-- the client default". pin_hash is a bcrypt hash (#118) and is never
-- returned by the API. A fresh install starts with an Admin and a Kids
-- profile, neither with a PIN.

-- +goose Up
CREATE TABLE profiles (
    id                    INTEGER PRIMARY KEY AUTOINCREMENT,
    name                  TEXT NOT NULL UNIQUE COLLATE NOCASE,
    role                  TEXT NOT NULL CHECK (role IN ('admin', 'viewer')),
    audience_tier         TEXT NOT NULL CHECK (audience_tier IN ('young_child', 'teen', 'adult')),
    pin_hash              TEXT,
    pin_failures          INTEGER NOT NULL DEFAULT 0,
    pin_locked_until_unix INTEGER NOT NULL DEFAULT 0,
    icon                  TEXT,
    theme                 TEXT,
    font_level            INTEGER CHECK (font_level BETWEEN 0 AND 3),
    brightness_level      INTEGER CHECK (brightness_level BETWEEN 0 AND 3),
    guide_span_hours      INTEGER CHECK (guide_span_hours BETWEEN 2 AND 12),
    guide_time_format     TEXT CHECK (guide_time_format IN ('12h-padded', '12h', '24h')),
    created_at_unix       INTEGER NOT NULL DEFAULT (unixepoch()),
    updated_at_unix       INTEGER NOT NULL DEFAULT (unixepoch())
);
INSERT INTO profiles (name, role, audience_tier) VALUES
    ('Admin', 'admin', 'adult'),
    ('Kids', 'viewer', 'young_child');

-- +goose Down
DROP TABLE profiles;
