-- +goose Up
-- A per-instance salt makes PIN lookup expensive without hashing every account.
CREATE TABLE tv_pairing_salt (id INTEGER PRIMARY KEY CHECK (id = 1), salt BLOB NOT NULL);
INSERT INTO tv_pairing_salt VALUES (1, randomblob(16));
CREATE TABLE tv_pairing_pins (
    user_id INTEGER PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    pin_digest BLOB NOT NULL UNIQUE
);

-- +goose Down
DROP TABLE tv_pairing_pins;
DROP TABLE tv_pairing_salt;
