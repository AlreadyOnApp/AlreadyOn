-- +goose Up
CREATE TABLE playback_delivery (
    id INTEGER PRIMARY KEY CHECK (id=1),
    media_url TEXT NOT NULL DEFAULT '',
    allowed_origins TEXT NOT NULL DEFAULT '[]'
);
INSERT INTO playback_delivery (id) VALUES (1);

-- +goose Down
DROP TABLE playback_delivery;
