-- +goose Up
-- Leases shorter than 20 seconds expire before the browser's ten-second
-- heartbeat can renew them. The column CHECK (10-120) stays as a looser bound;
-- the application enforces the new minimum.
UPDATE playback_settings SET lease_seconds = 20 WHERE lease_seconds < 20;

-- +goose Down
SELECT 1;
