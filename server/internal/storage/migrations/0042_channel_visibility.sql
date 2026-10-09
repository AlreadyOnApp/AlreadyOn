-- Each channel is Auto (shown when it has content), Show (always) or Hide
-- (never). channels.hidden stays as the effective result the guide reads.
-- Channels an admin or the lineup hid while they had content keep that as Hide.

-- +goose Up
ALTER TABLE channels ADD COLUMN visibility TEXT NOT NULL DEFAULT 'auto' CHECK (visibility IN ('auto', 'show', 'hide'));
UPDATE channels SET visibility = 'hide'
WHERE hidden = 1 AND EXISTS (SELECT 1 FROM channel_items ci WHERE ci.channel_id = channels.id);

-- +goose Down
ALTER TABLE channels DROP COLUMN visibility;
