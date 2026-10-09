-- The guide always shows 12 hours ahead, so the per-profile span goes away.
-- Movie/TV files of five minutes or less are ignored from now on (the scanner
-- drops them); remove the ones already recorded. Their schedule entries are
-- rebuilt at startup.

-- +goose Up
ALTER TABLE profiles DROP COLUMN guide_span_hours;
DELETE FROM media_files WHERE root_kind = 'media' AND duration_seconds > 0 AND duration_seconds <= 300;

-- +goose Down
ALTER TABLE profiles ADD COLUMN guide_span_hours INTEGER CHECK (guide_span_hours BETWEEN 2 AND 12);
