-- +goose Up
-- A friendly identity survives container replacement and is safe to show on the LAN.
ALTER TABLE server_settings ADD COLUMN server_name TEXT NOT NULL DEFAULT 'AlreadyOn';

-- +goose Down
ALTER TABLE server_settings DROP COLUMN server_name;
