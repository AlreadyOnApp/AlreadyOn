-- Flip the commercial-breaks singleton to enabled by default — most
-- environments exercising this feature already have commercial media
-- configured, so leaving it off just adds a needless extra step.

-- +goose Up
UPDATE commercial_settings SET enabled = 1 WHERE id = 1;

-- +goose Down
UPDATE commercial_settings SET enabled = 0 WHERE id = 1;
