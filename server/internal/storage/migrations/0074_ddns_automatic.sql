-- +goose Up
-- Configured providers now sync automatically; selecting None disables them.
UPDATE ddns_configs SET enabled=1 WHERE secret<>'' AND hostname<>'' AND (provider NOT IN ('dyndns','no-ip') OR username<>'');

-- +goose Down
-- Preserve administrator configuration when rolling back this UI policy.
SELECT 1;
