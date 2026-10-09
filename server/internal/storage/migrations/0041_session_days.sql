-- How many days a browser stays signed in after it signs in; 0 never expires.
-- Changing it affects new sign-ins only.

-- +goose Up
ALTER TABLE server_settings ADD COLUMN session_days INTEGER NOT NULL DEFAULT 30 CHECK (session_days BETWEEN 0 AND 3650);

-- +goose Down
ALTER TABLE server_settings DROP COLUMN session_days;
