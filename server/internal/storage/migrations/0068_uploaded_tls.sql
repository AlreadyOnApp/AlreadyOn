-- Keep the generated identity intact when an administrator activates an upload.
-- +goose Up
CREATE TABLE uploaded_tls (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    certificate_pem BLOB NOT NULL,
    private_key_pem BLOB NOT NULL,
    ca_pem BLOB NOT NULL,
    active INTEGER NOT NULL CHECK (active IN (0, 1))
);
-- +goose Down
DROP TABLE uploaded_tls;
