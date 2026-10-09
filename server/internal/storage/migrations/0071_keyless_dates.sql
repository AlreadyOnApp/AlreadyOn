-- Free-source dates have separate provenance from TMDb/manual enrichment.
-- A series premiere is stored per episode without inventing a TMDb show ID.
-- +goose Up
ALTER TABLE identified_items ADD COLUMN keyless_release_date TEXT;
ALTER TABLE identified_items ADD COLUMN keyless_provider TEXT;
ALTER TABLE identified_items ADD COLUMN keyless_fetched_at_unix INTEGER;

-- +goose Down
ALTER TABLE identified_items DROP COLUMN keyless_fetched_at_unix;
ALTER TABLE identified_items DROP COLUMN keyless_provider;
ALTER TABLE identified_items DROP COLUMN keyless_release_date;
