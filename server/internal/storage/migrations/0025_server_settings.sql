-- Server tab choices (#120): transcoding, TLS, ACME and DDNS. Choices
-- only. DDNS/TLS credentials are not stored anywhere yet; when those
-- features exist they must be write-only like the TMDb key. Singleton
-- row, same pattern as folder_settings.

-- +goose Up
CREATE TABLE server_settings (
    id              INTEGER PRIMARY KEY CHECK (id = 1),
    transcoding     TEXT NOT NULL DEFAULT 'software',
    tls_mode        TEXT NOT NULL DEFAULT 'self-signed',
    acme_provider   TEXT NOT NULL DEFAULT 'letsencrypt',
    acme_domain     TEXT NOT NULL DEFAULT '',
    ddns_provider   TEXT NOT NULL DEFAULT 'none',
    updated_at_unix INTEGER NOT NULL DEFAULT (unixepoch())
);
INSERT INTO server_settings (id) VALUES (1);

-- +goose Down
DROP TABLE server_settings;
