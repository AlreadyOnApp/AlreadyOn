-- Save the generator state after each scheduled segment so rebuilding the
-- future resumes every show's episode position, even across restarts.
-- +goose Up
ALTER TABLE schedule_entries ADD COLUMN continuation_json TEXT;

-- Generated TV pools used to contain only the first ~14 hours. Rebuild those
-- pools on startup so every season can air. Keep manual lineups and the
-- currently-playing schedule entries intact.
DELETE FROM channel_items WHERE channel_id IN (
 SELECT DISTINCT ci.channel_id FROM channel_items ci
 JOIN channel_filters cf ON cf.channel_id = ci.channel_id
 JOIN identified_items ii ON ii.media_file_id = ci.media_file_id
 WHERE ii.media_type = 'episode'
);

-- +goose Down
ALTER TABLE schedule_entries DROP COLUMN continuation_json;
