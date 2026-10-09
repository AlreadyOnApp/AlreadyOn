package library

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestLookupName(t *testing.T) {
	cases := []struct {
		name, display, path, want string
	}{
		{
			name:    "Sonarr layout: id tag only in the folder",
			display: "Love, Death & Robots (2019)",
			path:    "/tv/Love, Death & Robots (2019) {imdb-tt9561862}/Season 4/Love, Death & Robots (2019) - S04E01 - 400 Boys.mkv",
			want:    "Love, Death & Robots (2019) {imdb-tt9561862}",
		},
		{
			name:    "year only in the folder",
			display: "Band of Brothers",
			path:    "/tv/Band of Brothers (2001)/Season 01/Band of Brothers - s01e01.mkv",
			want:    "Band of Brothers (2001)",
		},
		{
			name:    "region and tvdb id from the folder",
			display: "The Office",
			path:    "/tv/The Office (US) (2005) {tvdb-73244}/Season 01/The Office - s01e01 - Pilot.mkv",
			want:    "The Office (US) (2005) {tvdb-73244}",
		},
		{
			name:    "flat layout must not adopt the library folder's name",
			display: "Show",
			path:    "/TV Shows (2010) {tvdb-1}/Show - S01E01.mkv",
			want:    "Show",
		},
		{
			name:    "movie year and id from folder and file name",
			display: "Batman Begins",
			path:    "/movies/Batman Begins (2005)/Batman Begins (2005) {tmdb-272}.mp4",
			want:    "Batman Begins (2005) {tmdb-272}",
		},
		{
			name:    "nothing to add",
			display: "Some Movie",
			path:    "/movies/Some Movie/Some Movie.mkv",
			want:    "Some Movie",
		},
	}
	for _, c := range cases {
		if got := lookupName(c.display, c.path); got != c.want {
			t.Errorf("%s: lookupName(%q) = %q, want %q", c.name, c.display, got, c.want)
		}
	}
}

// A folder's {imdb-…} tag reaches TMDb even though the file name (and so the
// stored series title) lacks it, and beats a title search.
func TestEnrichMetadata_FolderIDTag_ResolvesShowThroughFind(t *testing.T) {
	db := openClassifyTestDB(t)
	ctx := context.Background()
	now := time.Now().Unix()
	var fileID int64
	if err := db.QueryRowContext(ctx, `
		INSERT INTO media_files (root_kind, path, size_bytes, modified_unix, first_seen_unix, last_seen_unix)
		VALUES ('media', ?, 1, ?, ?, ?) RETURNING id`,
		"/tv/Odd Name {imdb-tt9561862}/Season 1/Odd Name - S01E01.mkv", now, now, now).Scan(&fileID); err != nil {
		t.Fatalf("insert media_files: %v", err)
	}
	var id int64
	if err := db.QueryRowContext(ctx, `
		INSERT INTO identified_items (media_file_id, media_type, title, series_title, season_number, episode_number)
		VALUES (?, 'episode', 'Pilot', 'Odd Name', 1, 1) RETURNING id`, fileID).Scan(&id); err != nil {
		t.Fatalf("insert identified_items: %v", err)
	}

	client, _ := mockTMDbPathRoutedClient(t, map[string]string{
		"/find/tt9561862":              `{"tv_results":[{"id":86831,"name":"Something Else Entirely"}]}`,
		"/tv/86831":                    `{"overview":"Show overview","first_air_date":"2019-03-15"}`,
		"/tv/86831/content_ratings":    `{"results":[]}`,
		"/tv/86831/season/1/episode/1": `{"overview":"Episode overview","air_date":"2019-03-15"}`,
	})
	if err := EnrichMetadata(ctx, db, client); err != nil {
		t.Fatalf("EnrichMetadata: %v", err)
	}
	var overview, showID any
	if err := db.QueryRow(`SELECT overview, show_id FROM identified_items WHERE id = ?`, id).Scan(&overview, &showID); err != nil {
		t.Fatal(err)
	}
	if s, _ := overview.(string); !strings.Contains(s, "Episode overview") || showID == nil {
		t.Errorf("overview = %v, show_id = %v; want the episode matched via the folder's imdb tag", overview, showID)
	}
}
