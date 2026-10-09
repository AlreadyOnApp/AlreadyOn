-- A manually chosen certification (#103) must survive metadata refreshes,
-- which would otherwise overwrite content_rating with TMDb's value. For an
-- episode, a non-NULL content_rating overrides its show's certification.

-- +goose Up
ALTER TABLE identified_items ADD COLUMN content_rating_manual INTEGER NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE identified_items DROP COLUMN content_rating_manual;
