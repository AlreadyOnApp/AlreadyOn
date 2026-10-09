-- Initial schema for issue #21: channel content selection and episode
-- ordering. A channel's lineup is just an ordered list of media files;
-- looping/repeating it is a schedule-generation concern (#22), not
-- something modeled by duplicate rows here.

-- +goose Up
CREATE TABLE channel_items (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    channel_id      INTEGER NOT NULL REFERENCES channels (id) ON DELETE CASCADE,
    media_file_id   INTEGER NOT NULL REFERENCES media_files (id) ON DELETE CASCADE,
    position        INTEGER NOT NULL,
    created_at_unix INTEGER NOT NULL DEFAULT (unixepoch()),
    UNIQUE (channel_id, position),
    UNIQUE (channel_id, media_file_id)
);

CREATE INDEX idx_channel_items_channel_id ON channel_items (channel_id);

-- +goose Down
DROP TABLE channel_items;
