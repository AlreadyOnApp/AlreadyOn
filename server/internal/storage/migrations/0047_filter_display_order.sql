-- +goose Up
ALTER TABLE channel_filter_options ADD COLUMN position INTEGER NOT NULL DEFAULT 1000;
UPDATE channel_filter_options SET position=CASE id
 WHEN 'young_child' THEN 0 WHEN 'teen' THEN 1 WHEN 'adult' THEN 2
 WHEN 'both' THEN 0 WHEN 'modern' THEN 1 WHEN 'nostalgia' THEN 2
 WHEN '2020s' THEN 3 WHEN '2010s' THEN 4 WHEN '2000s' THEN 5 WHEN '1990s' THEN 6
 WHEN '1980s' THEN 7 WHEN '1970s' THEN 8 WHEN '1960s' THEN 9 WHEN '1950s' THEN 10 ELSE 1000 END;
UPDATE channel_filter_options SET definition=json_set(definition,'$.name','Classics')
 WHERE kind='era' AND id='nostalgia' AND json_extract(definition,'$.name')='Oldies';
-- +goose Down
ALTER TABLE channel_filter_options DROP COLUMN position;
