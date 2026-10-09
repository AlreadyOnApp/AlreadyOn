-- +goose Up
-- Replace the legacy defaults once, preserving custom channels and all media/accounts.
CREATE TEMP TABLE approved_lineup (number INTEGER PRIMARY KEY, name TEXT, audience TEXT, genres TEXT, media TEXT, era TEXT, extra TEXT);
INSERT INTO approved_lineup VALUES (101,'Kids Toon Classic','young_child','["animation", "anime"]','tv','nostalgia','{"requireKnownYear": true}');
INSERT INTO approved_lineup VALUES (102,'Kids Toon Modern','young_child','["animation", "anime"]','tv','modern','{"requireKnownYear": true}');
INSERT INTO approved_lineup VALUES (103,'Kids Live Classic','young_child','[]','tv','nostalgia','{"requireKnownYear": true, "excludedGenres": ["animation", "anime"]}');
INSERT INTO approved_lineup VALUES (104,'Kids Live Modern','young_child','[]','tv','modern','{"requireKnownYear": true, "excludedGenres": ["animation", "anime"]}');
INSERT INTO approved_lineup VALUES (105,'Kids Movie Classic','young_child','[]','movie','nostalgia','{"requireKnownYear": true}');
INSERT INTO approved_lineup VALUES (106,'Kids Movie Modern','young_child','[]','movie','modern','{"requireKnownYear": true}');
INSERT INTO approved_lineup VALUES (107,'Teen Toon Classic','teen','["animation"]','tv','nostalgia','{"requireKnownYear": true, "excludedGenres": ["anime"]}');
INSERT INTO approved_lineup VALUES (108,'Teen Toon Modern','teen','["animation"]','tv','modern','{"requireKnownYear": true, "excludedGenres": ["anime"]}');
INSERT INTO approved_lineup VALUES (109,'Teen Anime One','teen','["anime"]','tv','both','{"requireKnownYear": true}');
INSERT INTO approved_lineup VALUES (110,'Teen Anime Two','teen','["anime"]','tv','both','{"requireKnownYear": true}');
INSERT INTO approved_lineup VALUES (111,'Teen Real Classic','teen','["reality"]','tv','nostalgia','{"requireKnownYear": true}');
INSERT INTO approved_lineup VALUES (112,'Teen Real Modern','teen','["reality"]','tv','modern','{"requireKnownYear": true}');
INSERT INTO approved_lineup VALUES (113,'Teen Drama Classic','teen','["drama"]','tv','nostalgia','{"requireKnownYear": true}');
INSERT INTO approved_lineup VALUES (114,'Teen Drama Modern','teen','["drama"]','tv','modern','{"requireKnownYear": true}');
INSERT INTO approved_lineup VALUES (115,'Teen Worlds Classic','teen','["fantasy", "scifi"]','tv','nostalgia','{"requireKnownYear": true}');
INSERT INTO approved_lineup VALUES (116,'Teen Worlds Modern','teen','["fantasy", "scifi"]','tv','modern','{"requireKnownYear": true}');
INSERT INTO approved_lineup VALUES (117,'Teen Action Classic','teen','["action", "adventure"]','tv','nostalgia','{"requireKnownYear": true}');
INSERT INTO approved_lineup VALUES (118,'Teen Action Modern','teen','["action", "adventure"]','tv','modern','{"requireKnownYear": true}');
INSERT INTO approved_lineup VALUES (119,'Teen Movie Classic','teen','[]','movie','nostalgia','{"requireKnownYear": true}');
INSERT INTO approved_lineup VALUES (120,'Teen Movie Modern','teen','[]','movie','modern','{"requireKnownYear": true}');
INSERT INTO approved_lineup VALUES (121,'Toon One','adult','["animation"]','tv','both','{"requireKnownYear": false, "excludedGenres": ["anime"]}');
INSERT INTO approved_lineup VALUES (122,'Toon Two','adult','["animation"]','tv','both','{"requireKnownYear": false, "excludedGenres": ["anime"]}');
INSERT INTO approved_lineup VALUES (123,'Anime One','adult','["anime"]','tv','both','{"requireKnownYear": false}');
INSERT INTO approved_lineup VALUES (124,'Anime Two','adult','["anime"]','tv','both','{"requireKnownYear": false}');
INSERT INTO approved_lineup VALUES (125,'Reality One','adult','["reality"]','tv','both','{"requireKnownYear": false}');
INSERT INTO approved_lineup VALUES (126,'Reality Two','adult','["reality"]','tv','both','{"requireKnownYear": false}');
INSERT INTO approved_lineup VALUES (127,'Drama One','adult','["drama"]','tv','both','{"requireKnownYear": false}');
INSERT INTO approved_lineup VALUES (128,'Drama Two','adult','["drama"]','tv','both','{"requireKnownYear": false}');
INSERT INTO approved_lineup VALUES (129,'Worlds One','adult','["fantasy", "scifi"]','tv','both','{"requireKnownYear": false}');
INSERT INTO approved_lineup VALUES (130,'Worlds Two','adult','["fantasy", "scifi"]','tv','both','{"requireKnownYear": false}');
INSERT INTO approved_lineup VALUES (131,'Action One','adult','["action", "adventure"]','tv','both','{"requireKnownYear": false}');
INSERT INTO approved_lineup VALUES (132,'Action Two','adult','["action", "adventure"]','tv','both','{"requireKnownYear": false}');
INSERT INTO approved_lineup VALUES (133,'Cinema One','adult','[]','movie','both','{"requireKnownYear": false}');
INSERT INTO approved_lineup VALUES (134,'Cinema Two','adult','[]','movie','both','{"requireKnownYear": false}');
INSERT INTO approved_lineup VALUES (135,'Action Cinema','adult','["action", "adventure", "crime", "thriller"]','movie','both','{"requireKnownYear": false}');
INSERT INTO approved_lineup VALUES (136,'Beyond Cinema','adult','["fantasy", "scifi", "horror"]','movie','both','{"requireKnownYear": false}');
CREATE TEMP TABLE legacy_lineup_present AS SELECT count(*) AS n FROM channels WHERE number BETWEEN 100 AND 152;
DELETE FROM channels WHERE number=100 OR number BETWEEN 137 AND 152;
UPDATE channels SET name=(SELECT name FROM approved_lineup WHERE number=channels.number), visibility='auto', hidden=0, updated_at_unix=unixepoch()
 WHERE number BETWEEN 101 AND 136 AND (SELECT n FROM legacy_lineup_present)>0;
INSERT INTO channels(number,name) SELECT number,name FROM approved_lineup WHERE (SELECT n FROM legacy_lineup_present)>0 AND number NOT IN (SELECT number FROM channels);
INSERT OR REPLACE INTO channel_filters(channel_id,audience_tier,genres,media_type,era,title_contains,extra)
 SELECT c.id,a.audience,a.genres,a.media,a.era,'',a.extra FROM channels c JOIN approved_lineup a ON a.number=c.number WHERE (SELECT n FROM legacy_lineup_present)>0;
DELETE FROM schedule_entries WHERE channel_id IN (SELECT id FROM channels WHERE number BETWEEN 101 AND 136) AND (SELECT n FROM legacy_lineup_present)>0;
DELETE FROM channel_items WHERE channel_id IN (SELECT id FROM channels WHERE number BETWEEN 101 AND 136) AND (SELECT n FROM legacy_lineup_present)>0;
DROP TABLE approved_lineup;
DROP TABLE legacy_lineup_present;

-- +goose Down
-- Library/accounts are unchanged. Previous generated defaults are not restored.
