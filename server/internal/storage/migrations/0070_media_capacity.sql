-- +goose Up
-- Media folder capacity, summed over distinct filesystems, beside /config.
ALTER TABLE server_metrics ADD COLUMN media_used_bytes INTEGER;
ALTER TABLE server_metrics ADD COLUMN media_total_bytes INTEGER;
-- +goose Down
ALTER TABLE server_metrics DROP COLUMN media_total_bytes;
ALTER TABLE server_metrics DROP COLUMN media_used_bytes;
