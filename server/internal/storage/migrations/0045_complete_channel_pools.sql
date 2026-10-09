-- +goose Up
-- Existing installations get the new default; empty databases are seeded by the app.
INSERT INTO channels (number, name, artwork_ref)
SELECT 152, 'Western and War', '/channel-icons/152.svg'
WHERE EXISTS (SELECT 1 FROM channels)
AND NOT EXISTS (SELECT 1 FROM channels WHERE number = 152);

-- Preserve administrator additions while expanding Golden Age Cinema.
UPDATE channel_filters SET genres = (
 SELECT json_group_array(value) FROM (
  SELECT value FROM json_each(channel_filters.genres)
  UNION SELECT 'war' UNION SELECT 'western'
 )
)
WHERE channel_id IN (SELECT id FROM channels WHERE number = 145);

-- Candidate pools are derived data. Rebuild using complete movie pools and
-- show premiere dates. Current airings/checkpoints remain intact.
DELETE FROM channel_items WHERE channel_id IN (SELECT channel_id FROM channel_filters);
