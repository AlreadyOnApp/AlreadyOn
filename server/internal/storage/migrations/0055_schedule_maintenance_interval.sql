-- +goose Up
ALTER TABLE server_settings ADD COLUMN schedule_maintenance_minutes INTEGER NOT NULL DEFAULT 60;
-- +goose Down
ALTER TABLE server_settings DROP COLUMN schedule_maintenance_minutes;
