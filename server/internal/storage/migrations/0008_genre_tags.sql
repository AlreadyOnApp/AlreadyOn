-- Adds genre classification to identified_items (issue #79), so
-- default channel lineup population (see internal/lineup) can match
-- library content to a channel's theme. NULL means "not classified
-- yet" (no TMDb key configured, no match found, or not attempted) —
-- distinct from an empty JSON array "[]", which means "classified, no
-- genres matched our taxonomy".

-- +goose Up
ALTER TABLE identified_items ADD COLUMN genre_tags TEXT;

-- +goose Down
ALTER TABLE identified_items DROP COLUMN genre_tags;
