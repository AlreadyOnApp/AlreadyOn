-- Hardware decoding (with scaling and tone mapping) is a separate opt-in that
-- builds on a hardware encoder (issue #47); the server refuses it while the
-- encoder is software.

-- +goose Up
ALTER TABLE server_settings ADD COLUMN hardware_decoding INTEGER NOT NULL DEFAULT 0 CHECK (hardware_decoding IN (0, 1));

-- +goose Down
ALTER TABLE server_settings DROP COLUMN hardware_decoding;
