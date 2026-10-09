-- +goose Up
-- Keep applied migration history intact; discard the retired fields on upgrade.
ALTER TABLE channels DROP COLUMN artwork_ref;
ALTER TABLE profiles DROP COLUMN guide_logos;
UPDATE profiles SET icon = '/profile-icons/' || substr(icon, length('/channel-artwork/profile-icons/') + 1)
WHERE substr(icon, 1, length('/channel-artwork/profile-icons/')) = '/channel-artwork/profile-icons/';

-- +goose Down
-- Removed image references and preferences cannot be recovered.
ALTER TABLE channels ADD COLUMN artwork_ref TEXT;
ALTER TABLE profiles ADD COLUMN guide_logos INTEGER CHECK (guide_logos IN (0, 1));
