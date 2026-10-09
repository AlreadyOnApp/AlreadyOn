-- Shows in "Name (2019)" folders never matched TMDb because the search used
-- the literal title. The lookup now strips the year, so retry those rows
-- (manual genres are kept).

-- +goose Up
UPDATE identified_items
SET metadata_fetched_at_unix = NULL
WHERE media_type = 'episode' AND show_id IS NULL
  AND series_title GLOB '*([0-9][0-9][0-9][0-9])';

UPDATE identified_items
SET genre_tags = NULL, genre_source = NULL
WHERE media_type = 'episode' AND genre_tags = '[]'
  AND COALESCE(genre_source, '') != 'manual'
  AND series_title GLOB '*([0-9][0-9][0-9][0-9])';

-- +goose Down
SELECT 1;
