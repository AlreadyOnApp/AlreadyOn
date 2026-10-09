-- +goose Up
-- Keep provider credentials separate from readable settings and retain each
-- provider's setup when the administrator switches providers for testing.
CREATE TABLE ddns_configs (
 provider TEXT PRIMARY KEY CHECK(provider IN ('cloudflare','dyndns','no-ip','duckdns')),
 hostname TEXT NOT NULL DEFAULT '', zone_id TEXT NOT NULL DEFAULT '',
 username TEXT NOT NULL DEFAULT '', secret TEXT NOT NULL DEFAULT '',
 enabled INTEGER NOT NULL DEFAULT 0,
 last_attempt INTEGER NOT NULL DEFAULT 0, last_update INTEGER NOT NULL DEFAULT 0,
 last_verified INTEGER NOT NULL DEFAULT 0, last_ip TEXT NOT NULL DEFAULT '',
 last_error TEXT NOT NULL DEFAULT '', failures INTEGER NOT NULL DEFAULT 0,
 next_attempt INTEGER NOT NULL DEFAULT 0, blocked INTEGER NOT NULL DEFAULT 0
);
UPDATE server_settings SET ddns_provider='none' WHERE ddns_provider='custom';
-- +goose Down
DROP TABLE ddns_configs;
