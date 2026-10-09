-- A channel identity follows the profile across clients; deletion clears it.
-- +goose Up
ALTER TABLE profiles ADD COLUMN last_watched_channel_id INTEGER REFERENCES channels(id) ON DELETE SET NULL;

-- +goose Down
ALTER TABLE profiles DROP COLUMN last_watched_channel_id;
