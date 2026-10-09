-- +goose Up
CREATE TABLE channel_filter_options (
 kind TEXT NOT NULL CHECK(kind IN ('age','era','collection')),
 id TEXT NOT NULL,
 definition TEXT NOT NULL,
 PRIMARY KEY(kind,id)
);
INSERT INTO channel_filter_options VALUES
 ('age','young_child','{"name":"Kids","ratings":["TV-Y","TV-Y7","TV-G","G"]}'),
 ('age','teen','{"name":"Teen","ratings":["TV-Y","TV-Y7","TV-G","TV-PG","TV-14","G","PG","PG-13"]}'),
 ('age','adult','{"name":"Unrestricted","unrestricted":true,"ratings":[]}'),
 ('era','both','{"name":"All"}'),
 ('era','modern','{"name":"Modern","startYear":2010}'),
 ('era','nostalgia','{"name":"Oldies","endYear":2009}'),
 ('era','2020s','{"name":"2020’s","startYear":2020,"endYear":2029}'),
 ('era','2010s','{"name":"2010’s","startYear":2010,"endYear":2019}'),
 ('era','2000s','{"name":"2000’s","startYear":2000,"endYear":2009}'),
 ('era','1990s','{"name":"1990’s","startYear":1990,"endYear":1999}'),
 ('era','1980s','{"name":"1980’s","startYear":1980,"endYear":1989}'),
 ('era','1970s','{"name":"1970’s","startYear":1970,"endYear":1979}'),
 ('era','1960s','{"name":"1960’s","startYear":1960,"endYear":1969}'),
 ('era','1950s','{"name":"1950’s","startYear":1950,"endYear":1959}');
ALTER TABLE channel_filters ADD COLUMN extra TEXT NOT NULL DEFAULT '{}';
-- +goose Down
ALTER TABLE channel_filters DROP COLUMN extra;
DROP TABLE channel_filter_options;
