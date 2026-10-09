-- Persist instance TLS identity with the database, never with the image.
-- +goose Up
CREATE TABLE instance_tls (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    root_cert_pem BLOB NOT NULL,
    root_key_pem BLOB NOT NULL,
    server_cert_pem BLOB NOT NULL,
    server_key_pem BLOB NOT NULL
);

-- +goose Down
DROP TABLE instance_tls;
