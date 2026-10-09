-- +goose Up
UPDATE channel_filter_options SET position=CASE id WHEN 'adult' THEN 0 WHEN 'teen' THEN 1 ELSE 2 END WHERE kind='age' AND id IN ('adult','teen','young_child');
UPDATE channel_filter_options SET definition=json_set(definition,'$.name','Teens') WHERE kind='age' AND id='teen' AND json_extract(definition,'$.name')='Teen';
-- +goose Down
SELECT 1;
