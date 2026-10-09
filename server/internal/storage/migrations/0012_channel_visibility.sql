-- Adds a persisted, admin-controlled show/hide flag to channels. The
-- automatic ">=2 distinct shows" rule (previously live-recomputed on
-- every guide request — see #90) now only supplies the *default* value
-- at population time (internal/lineup) — after that, this column is the
-- sole authority, so an admin's explicit choice always sticks until
-- they change it again (including via the "Hide Empty" bulk action,
-- which just re-applies that same rule on demand).

-- +goose Up
ALTER TABLE channels ADD COLUMN hidden INTEGER NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE channels DROP COLUMN hidden;
