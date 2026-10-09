-- Adds manual-correction tracking to identified_items (issue #17),
-- deferred out of 0001_init.sql on purpose so this schema addition
-- stays scoped to the ticket that actually needed it.

-- +goose Up
ALTER TABLE identified_items ADD COLUMN is_manual_correction INTEGER NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE identified_items DROP COLUMN is_manual_correction;
