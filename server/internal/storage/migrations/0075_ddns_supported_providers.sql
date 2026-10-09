-- +goose Up
-- Retain retired setup privately, but prevent it from being selected or updated.
UPDATE server_settings SET ddns_provider='none' WHERE ddns_provider IN ('dyndns','duckdns');
UPDATE ddns_configs SET enabled=0,blocked=1 WHERE provider IN ('dyndns','duckdns');

-- +goose Down
-- Do not reactivate removed providers or change the administrator's selection.
SELECT 1;
