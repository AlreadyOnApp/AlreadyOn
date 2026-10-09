-- +goose Up
-- Account keys and issued private keys stay in the existing private database.
CREATE TABLE acme_tls (
 id INTEGER PRIMARY KEY CHECK (id=1),
 active_hostname TEXT NOT NULL DEFAULT '',
 hostname TEXT NOT NULL DEFAULT '', email TEXT NOT NULL DEFAULT '',
 enabled INTEGER NOT NULL DEFAULT 0,
 account_key BLOB NOT NULL DEFAULT '', staging_key BLOB NOT NULL DEFAULT '',
 certificate_pem BLOB NOT NULL DEFAULT '', private_key_pem BLOB NOT NULL DEFAULT '',
 last_attempt INTEGER NOT NULL DEFAULT 0, next_attempt INTEGER NOT NULL DEFAULT 0,
 last_success INTEGER NOT NULL DEFAULT 0, message TEXT NOT NULL DEFAULT '',
 staging_hostname TEXT NOT NULL DEFAULT ''
);
INSERT INTO acme_tls(id) VALUES(1);
UPDATE server_settings SET acme_provider='letsencrypt';
-- +goose Down
DROP TABLE acme_tls;
