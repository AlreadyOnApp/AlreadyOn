-- Multiple user accounts (#278, #279). The single-row server_account
-- becomes user 1, an owner. Sessions, API keys and household profiles now
-- belong to one user each. Columns default to user 1 so existing rows keep
-- belonging to the original account.
--
-- These columns deliberately have no foreign keys. On a fresh install the
-- seeded profiles belong to user 1 before the setup wizard creates that
-- user, and SQLite cannot add a referencing column with a non-NULL
-- default. Deleting a user removes its rows in the same transaction instead
-- (see auth.DeleteUser).

-- +goose Up
CREATE TABLE users (
    id                   INTEGER PRIMARY KEY AUTOINCREMENT,
    username             TEXT NOT NULL UNIQUE COLLATE NOCASE,
    password_hash        TEXT NOT NULL,
    role                 TEXT NOT NULL CHECK (role IN ('owner', 'viewer')),
    disabled             INTEGER NOT NULL DEFAULT 0 CHECK (disabled IN (0, 1)),
    must_change_password INTEGER NOT NULL DEFAULT 0 CHECK (must_change_password IN (0, 1)),
    password_locked      INTEGER NOT NULL DEFAULT 0 CHECK (password_locked IN (0, 1)),
    -- 0 means no per-user limit; the server-wide limits still apply.
    max_streams          INTEGER NOT NULL DEFAULT 0 CHECK (max_streams BETWEEN 0 AND 32),
    created_at_unix      INTEGER NOT NULL DEFAULT (unixepoch()),
    updated_at_unix      INTEGER NOT NULL DEFAULT (unixepoch())
);
INSERT INTO users (id, username, password_hash, role, created_at_unix)
    SELECT id, username, password_hash, 'owner', created_at_unix FROM server_account;

-- A fresh install also gets the Teen default; existing households keep
-- exactly the profiles they already chose.
INSERT INTO profiles (name, role, audience_tier)
    SELECT 'Teen', 'viewer', 'teen'
    WHERE NOT EXISTS (SELECT 1 FROM server_account)
      AND NOT EXISTS (SELECT 1 FROM profiles WHERE name = 'Teen');

DROP TABLE server_account;

ALTER TABLE browser_sessions ADD COLUMN user_id INTEGER NOT NULL DEFAULT 1;
CREATE INDEX browser_sessions_user ON browser_sessions (user_id);
ALTER TABLE api_keys ADD COLUMN user_id INTEGER NOT NULL DEFAULT 1;
CREATE INDEX api_keys_user ON api_keys (user_id);

-- Profile names are unique per user rather than server-wide, which needs
-- a table rebuild. Nothing references profiles, so ids are kept as-is.
CREATE TABLE profiles_new (
    id                      INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id                 INTEGER NOT NULL DEFAULT 1,
    name                    TEXT NOT NULL COLLATE NOCASE,
    role                    TEXT NOT NULL CHECK (role IN ('admin', 'viewer')),
    audience_tier           TEXT NOT NULL CHECK (audience_tier IN ('young_child', 'teen', 'adult')),
    pin_hash                TEXT,
    pin_failures            INTEGER NOT NULL DEFAULT 0,
    pin_locked_until_unix   INTEGER NOT NULL DEFAULT 0,
    icon                    TEXT,
    theme                   TEXT,
    font_level              INTEGER CHECK (font_level BETWEEN 0 AND 3),
    brightness_level        INTEGER CHECK (brightness_level BETWEEN 0 AND 3),
    guide_time_format       TEXT CHECK (guide_time_format IN ('12h-padded', '12h', '24h')),
    created_at_unix         INTEGER NOT NULL DEFAULT (unixepoch()),
    updated_at_unix         INTEGER NOT NULL DEFAULT (unixepoch()),
    subtitles_enabled       INTEGER CHECK (subtitles_enabled IN (0, 1)),
    subtitle_language       TEXT,
    audio_language          TEXT,
    last_watched_channel_id INTEGER REFERENCES channels(id) ON DELETE SET NULL,
    UNIQUE (user_id, name)
);
INSERT INTO profiles_new (id, user_id, name, role, audience_tier, pin_hash, pin_failures, pin_locked_until_unix,
        icon, theme, font_level, brightness_level, guide_time_format, created_at_unix, updated_at_unix,
        subtitles_enabled, subtitle_language, audio_language, last_watched_channel_id)
    SELECT id, 1, name, role, audience_tier, pin_hash, pin_failures, pin_locked_until_unix,
        icon, theme, font_level, brightness_level, guide_time_format, created_at_unix, updated_at_unix,
        subtitles_enabled, subtitle_language, audio_language, last_watched_channel_id
    FROM profiles;
DROP TABLE profiles;
ALTER TABLE profiles_new RENAME TO profiles;

-- +goose Down
-- Only user 1 and its profiles survive a downgrade; other accounts' rows
-- would otherwise silently gain the original owner's access.
DELETE FROM browser_sessions WHERE user_id <> 1;
DELETE FROM api_keys WHERE user_id <> 1;
DELETE FROM profiles WHERE user_id <> 1;
CREATE TABLE server_account (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    username TEXT NOT NULL,
    password_hash TEXT NOT NULL,
    created_at_unix INTEGER NOT NULL DEFAULT (unixepoch())
);
INSERT INTO server_account (id, username, password_hash, created_at_unix)
    SELECT id, username, password_hash, created_at_unix FROM users WHERE id = 1;
CREATE TABLE profiles_old (
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
    guide_time_format     TEXT CHECK (guide_time_format IN ('12h-padded', '12h', '24h')),
    created_at_unix       INTEGER NOT NULL DEFAULT (unixepoch()),
    updated_at_unix       INTEGER NOT NULL DEFAULT (unixepoch()),
    subtitles_enabled     INTEGER CHECK (subtitles_enabled IN (0, 1)),
    subtitle_language     TEXT,
    audio_language        TEXT,
    last_watched_channel_id INTEGER REFERENCES channels(id) ON DELETE SET NULL
);
INSERT INTO profiles_old (id, name, role, audience_tier, pin_hash, pin_failures, pin_locked_until_unix,
        icon, theme, font_level, brightness_level, guide_time_format, created_at_unix, updated_at_unix,
        subtitles_enabled, subtitle_language, audio_language, last_watched_channel_id)
    SELECT id, name, role, audience_tier, pin_hash, pin_failures, pin_locked_until_unix,
        icon, theme, font_level, brightness_level, guide_time_format, created_at_unix, updated_at_unix,
        subtitles_enabled, subtitle_language, audio_language, last_watched_channel_id
    FROM profiles;
DROP TABLE profiles;
ALTER TABLE profiles_old RENAME TO profiles;
DROP INDEX api_keys_user;
ALTER TABLE api_keys DROP COLUMN user_id;
DROP INDEX browser_sessions_user;
ALTER TABLE browser_sessions DROP COLUMN user_id;
DROP TABLE users;
