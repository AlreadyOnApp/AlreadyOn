-- +goose Up
-- Only extend the original default; never widen an administrator's custom policy.
UPDATE channel_filter_options
SET definition = json_set(definition, '$.ratings[#]', 'TV-Y7-FV')
WHERE kind = 'age' AND id = 'teen'
  AND json_extract(definition, '$.ratings') = '["TV-Y","TV-Y7","TV-G","TV-PG","TV-14","G","PG","PG-13"]';

-- +goose Down
-- Retain explicit rating choices on rollback rather than remove user policy.
SELECT 1;
