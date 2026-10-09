-- +goose Up
ALTER TABLE channel_filter_options ADD COLUMN default_position INTEGER NOT NULL DEFAULT 0;
UPDATE channel_filter_options SET default_position=CASE
 WHEN kind='age' AND id='adult' THEN 0
 WHEN kind='age' AND id='teen' THEN 1
 WHEN kind='age' AND id='young_child' THEN 2
 WHEN kind='era' AND id='both' THEN 0
 WHEN kind='era' AND id='modern' THEN 1
 WHEN kind='era' AND id='nostalgia' THEN 2
 WHEN kind='era' AND id='2020s' THEN 3
 WHEN kind='era' AND id='2010s' THEN 4
 WHEN kind='era' AND id='2000s' THEN 5
 WHEN kind='era' AND id='1990s' THEN 6
 WHEN kind='era' AND id='1980s' THEN 7
 WHEN kind='era' AND id='1970s' THEN 8
 WHEN kind='era' AND id='1960s' THEN 9
 WHEN kind='era' AND id='1950s' THEN 10
 ELSE 1000+rowid END;
-- +goose Down
ALTER TABLE channel_filter_options DROP COLUMN default_position;
