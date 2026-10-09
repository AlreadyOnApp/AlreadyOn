-- Give every database its own browser-cookie namespace. Browser cookies are
-- shared by hostname rather than port, so this keeps separate AlreadyOn
-- instances on the same host signed in independently.

-- +goose Up
ALTER TABLE server_settings ADD COLUMN session_cookie_name TEXT NOT NULL DEFAULT '';
UPDATE server_settings
SET session_cookie_name = 'alreadyon_' || lower(hex(randomblob(16)))
WHERE id = 1;
-- Existing browsers only hold the former fixed-name cookie and cannot present
-- these tokens after the rename, so do not leave unreachable sessions listed.
DELETE FROM browser_sessions;

-- +goose Down
ALTER TABLE server_settings DROP COLUMN session_cookie_name;
