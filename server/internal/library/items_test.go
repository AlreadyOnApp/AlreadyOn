package library

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"testing"
)

func mustPatch(t *testing.T, body string) ItemPatch {
	t.Helper()
	var p ItemPatch
	if err := json.Unmarshal([]byte(body), &p); err != nil {
		t.Fatalf("unmarshal %s: %v", body, err)
	}
	return p
}

type itemState struct {
	mediaType, title             string
	series, genres, source, cert sql.NullString
	year, season, episode        sql.NullInt64
	manual, certManual           int
	fetchedAt                    sql.NullInt64
	showID                       sql.NullInt64
}

func loadItem(t *testing.T, db *sql.DB, id int64) itemState {
	t.Helper()
	var s itemState
	if err := db.QueryRow(`
		SELECT media_type, title, series_title, genre_tags, genre_source, content_rating, release_year, season_number, episode_number,
		       is_manual_correction, content_rating_manual, metadata_fetched_at_unix, show_id
		FROM identified_items WHERE id = ?`, id).
		Scan(&s.mediaType, &s.title, &s.series, &s.genres, &s.source, &s.cert, &s.year, &s.season, &s.episode, &s.manual, &s.certManual, &s.fetchedAt, &s.showID); err != nil {
		t.Fatalf("load item %d: %v", id, err)
	}
	return s
}

func TestUpdateItems_CorrectionPersistsAndRescansDoNotOverwriteIt(t *testing.T) {
	db := openClassifyTestDB(t)
	ctx := context.Background()
	id := insertUnclassified(t, db, "movie", "Bad Title", "", 2000)

	n, err := UpdateItems(ctx, db, []int64{id}, mustPatch(t, `{"title":"  Good Title ","year":2010}`))
	if err != nil || n != 1 {
		t.Fatalf("UpdateItems = %d, %v", n, err)
	}
	s := loadItem(t, db, id)
	if s.title != "Good Title" || s.year.Int64 != 2010 || s.manual != 1 {
		t.Fatalf("state = %+v, want the corrected, protected item", s)
	}

	var fileID int64
	db.QueryRow(`SELECT media_file_id FROM identified_items WHERE id = ?`, id).Scan(&fileID)
	if err := SaveIdentification(ctx, db, fileID, Identification{Title: "Rescan Guess", Year: 1999}); err != nil {
		t.Fatalf("SaveIdentification: %v", err)
	}
	if got := loadItem(t, db, id); got.title != "Good Title" {
		t.Errorf("title after rescan = %q, want the manual correction kept", got.title)
	}
}

func TestUpdateItems_ChangingIdentityQueuesAutomaticLookupsAgain(t *testing.T) {
	db := openClassifyTestDB(t)
	ctx := context.Background()
	auto := insertUnclassified(t, db, "movie", "Misparsed", "", 2000)
	manualGenres := insertUnclassified(t, db, "movie", "Hand Tagged", "", 2000)
	db.Exec(`UPDATE identified_items SET genre_tags = '[]', genre_source = 'tmdb', metadata_fetched_at_unix = 100 WHERE id = ?`, auto)
	db.Exec(`UPDATE identified_items SET genre_tags = '["drama"]', genre_source = 'manual', metadata_fetched_at_unix = 100 WHERE id = ?`, manualGenres)

	if _, err := UpdateItems(ctx, db, []int64{auto, manualGenres}, mustPatch(t, `{"title":"Fixed"}`)); err != nil {
		t.Fatalf("UpdateItems: %v", err)
	}
	a := loadItem(t, db, auto)
	if a.genres.Valid || a.source.Valid || a.fetchedAt.Valid {
		t.Errorf("auto item = %+v, want genres and metadata cleared for a retry", a)
	}
	m := loadItem(t, db, manualGenres)
	if m.genres.String != `["drama"]` || m.source.String != "manual" {
		t.Errorf("manually tagged item lost its genres: %+v", m)
	}
}

func TestUpdateItems_ManualGenresSurviveClassification(t *testing.T) {
	db := openClassifyTestDB(t)
	ctx := context.Background()
	id := insertUnclassified(t, db, "movie", "Some Movie", "", 2000)
	if _, err := UpdateItems(ctx, db, []int64{id}, mustPatch(t, `{"genres":["comedy","comedy","romance"]}`)); err != nil {
		t.Fatalf("UpdateItems: %v", err)
	}
	s := loadItem(t, db, id)
	if s.genres.String != `["comedy","romance"]` || s.source.String != "manual" || s.manual != 0 {
		t.Fatalf("state = %+v, want de-duplicated manual genres and no identity flag", s)
	}

	client := mockTMDbClient(t, `{"results":[{"id":1,"genre_ids":[28]}]}`)
	if err := ClassifyGenres(ctx, db, client); err != nil {
		t.Fatalf("ClassifyGenres: %v", err)
	}
	if got := loadItem(t, db, id); got.genres.String != `["comedy","romance"]` {
		t.Errorf("classification overwrote manual genres: %q", got.genres.String)
	}
}

func TestUpdateItems_NullGenresHandsBackToAutomatic(t *testing.T) {
	db := openClassifyTestDB(t)
	ctx := context.Background()
	id := insertUnclassified(t, db, "movie", "Some Movie", "", 2000)
	UpdateItems(ctx, db, []int64{id}, mustPatch(t, `{"genres":["drama"]}`))
	if _, err := UpdateItems(ctx, db, []int64{id}, mustPatch(t, `{"genres":null}`)); err != nil {
		t.Fatalf("UpdateItems: %v", err)
	}
	if s := loadItem(t, db, id); s.genres.Valid || s.source.Valid {
		t.Errorf("state = %+v, want genres cleared so classification retries", s)
	}
}

func TestUpdateItems_ManualCertificationSurvivesEnrichment(t *testing.T) {
	db := openClassifyTestDB(t)
	ctx := context.Background()
	id := insertUnclassified(t, db, "movie", "Fight Club", "", 1999)
	if _, err := UpdateItems(ctx, db, []int64{id}, mustPatch(t, `{"certification":"PG"}`)); err != nil {
		t.Fatalf("UpdateItems: %v", err)
	}
	client, _ := mockTMDbPathRoutedClient(t, map[string]string{
		"/search/movie":            `{"results":[{"id":550,"genre_ids":[18]}]}`,
		"/movie/550":               `{"overview":"An insomniac...","release_date":"1999-10-15"}`,
		"/movie/550/release_dates": `{"results":[{"iso_3166_1":"US","release_dates":[{"certification":"R","type":3}]}]}`,
	})
	if err := EnrichMetadata(ctx, db, client); err != nil {
		t.Fatalf("EnrichMetadata: %v", err)
	}
	s := loadItem(t, db, id)
	if s.cert.String != "PG" || s.certManual != 1 {
		t.Errorf("cert = %q (manual %d), want the manual PG kept over TMDb's R", s.cert.String, s.certManual)
	}

	// Clearing it hands the item back to TMDb on the next enrichment.
	if _, err := UpdateItems(ctx, db, []int64{id}, mustPatch(t, `{"certification":null}`)); err != nil {
		t.Fatalf("clear: %v", err)
	}
	if err := EnrichMetadata(ctx, db, client); err != nil {
		t.Fatalf("EnrichMetadata: %v", err)
	}
	if s := loadItem(t, db, id); s.cert.String != "R" || s.certManual != 0 {
		t.Errorf("after clearing: cert = %q (manual %d), want TMDb's R", s.cert.String, s.certManual)
	}
}

func TestUpdateItems_Validation(t *testing.T) {
	db := openClassifyTestDB(t)
	ctx := context.Background()
	movie := insertUnclassified(t, db, "movie", "A Movie", "", 2000)
	ep := insertUnclassified(t, db, "episode", "Ep", "Some Show", 0)
	long := make([]byte, 201)
	for i := range long {
		long[i] = 'a'
	}

	cases := []struct {
		name  string
		id    int64
		patch string
	}{
		{"empty title", movie, `{"title":"  "}`},
		{"long title", movie, `{"title":"` + string(long) + `"}`},
		{"bad media type", movie, `{"mediaType":"song"}`},
		{"unknown genre", movie, `{"genres":["comedy","nope"]}`},
		{"TV rating on a movie", movie, `{"certification":"TV-14"}`},
		{"movie rating on an episode", ep, `{"certification":"PG"}`},
		{"year too small", movie, `{"year":1200}`},
		{"year too large", movie, `{"year":3000}`},
		{"negative season", ep, `{"season":-1}`},
		{"episode without a series", ep, `{"seriesTitle":null}`},
		{"movie to episode without a series", movie, `{"mediaType":"episode"}`},
		{"switching type strands the rating", ep, `{"mediaType":"movie","certification":"TV-14"}`},
		{"nothing to change", movie, `{}`},
	}
	for _, c := range cases {
		before := loadItem(t, db, c.id)
		if _, err := UpdateItems(ctx, db, []int64{c.id}, mustPatch(t, c.patch)); !errors.Is(err, ErrInvalidItem) {
			t.Errorf("%s: err = %v, want ErrInvalidItem", c.name, err)
		}
		if after := loadItem(t, db, c.id); after != before {
			t.Errorf("%s: a rejected patch wrote to the row", c.name)
		}
	}
	if _, err := UpdateItems(ctx, db, nil, mustPatch(t, `{"title":"x"}`)); !errors.Is(err, ErrInvalidItem) {
		t.Errorf("no ids: err = %v, want ErrInvalidItem", err)
	}
}

func TestUpdateItems_MovieClearsEpisodeFields(t *testing.T) {
	db := openClassifyTestDB(t)
	ep := insertUnclassified(t, db, "episode", "Ep", "Some Show", 0)
	db.Exec(`UPDATE identified_items SET season_number = 2, episode_number = 3 WHERE id = ?`, ep)
	if _, err := UpdateItems(context.Background(), db, []int64{ep}, mustPatch(t, `{"mediaType":"movie","title":"Actually A Movie","year":2001}`)); err != nil {
		t.Fatalf("UpdateItems: %v", err)
	}
	s := loadItem(t, db, ep)
	if s.mediaType != "movie" || s.series.Valid || s.season.Valid || s.episode.Valid {
		t.Errorf("state = %+v, want series/season/episode cleared for a movie", s)
	}
}

func TestUpdateItems_BulkAppliesToEveryItemAllOrNothing(t *testing.T) {
	db := openClassifyTestDB(t)
	ctx := context.Background()
	a := insertUnclassified(t, db, "movie", "File A", "", 0)
	b := insertUnclassified(t, db, "movie", "File B", "", 0)
	c := insertUnclassified(t, db, "movie", "File C", "", 0)

	n, err := UpdateItems(ctx, db, []int64{a, b, c, a}, mustPatch(t, `{"mediaType":"episode","seriesTitle":"My Show","season":1,"genres":["drama"],"certification":"TV-14"}`))
	if err != nil || n != 3 {
		t.Fatalf("bulk = %d, %v; want 3 distinct items", n, err)
	}
	for _, id := range []int64{a, b, c} {
		s := loadItem(t, db, id)
		if s.mediaType != "episode" || s.series.String != "My Show" || s.season.Int64 != 1 || s.genres.String != `["drama"]` || s.cert.String != "TV-14" {
			t.Errorf("item %d = %+v", id, s)
		}
		if s.title == "" {
			t.Errorf("item %d lost its own title", id)
		}
	}

	// One unknown id rejects the whole request, leaving the others untouched.
	before := loadItem(t, db, a)
	if _, err := UpdateItems(ctx, db, []int64{a, 99999}, mustPatch(t, `{"seriesTitle":"Other Show"}`)); !errors.Is(err, ErrItemNotFound) {
		t.Errorf("err = %v, want ErrItemNotFound", err)
	}
	if after := loadItem(t, db, a); after != before {
		t.Error("a failed bulk request partially applied")
	}
}

// A show that TMDb couldn't find under its misparsed name is looked up again
// once the name is corrected, and picks up TMDb's genres, show and rating.
func TestUpdateItems_CorrectedShowIsLookedUpAgain(t *testing.T) {
	db := openClassifyTestDB(t)
	ctx := context.Background()
	id := insertUnclassified(t, db, "episode", "Pilot", "Foo (2019)", 0)

	miss, _ := mockTMDbPathRoutedClient(t, map[string]string{"/search/tv": `{"results":[]}`})
	if err := ClassifyGenres(ctx, db, miss); err != nil {
		t.Fatalf("first ClassifyGenres: %v", err)
	}
	if err := EnrichMetadata(ctx, db, miss); err != nil {
		t.Fatalf("first EnrichMetadata: %v", err)
	}
	s := loadItem(t, db, id)
	if s.genres.String != "[]" || s.showID.Valid || !s.fetchedAt.Valid {
		t.Fatalf("after the failed lookups: %+v, want no genres, no show, and the attempt recorded", s)
	}

	if _, err := UpdateItems(ctx, db, []int64{id}, mustPatch(t, `{"seriesTitle":"Foo"}`)); err != nil {
		t.Fatalf("UpdateItems: %v", err)
	}

	hit, _ := mockTMDbPathRoutedClient(t, map[string]string{
		"/search/tv":                `{"results":[{"id":77,"genre_ids":[16]}]}`,
		"/tv/77":                    `{"overview":"A show.","first_air_date":"2019-03-15"}`,
		"/tv/77/content_ratings":    `{"results":[{"iso_3166_1":"US","rating":"TV-MA"}]}`,
		"/tv/77/season/0/episode/0": `{"overview":"The pilot.","air_date":"2019-03-15"}`,
	})
	if err := ClassifyGenres(ctx, db, hit); err != nil {
		t.Fatalf("second ClassifyGenres: %v", err)
	}
	if err := EnrichMetadata(ctx, db, hit); err != nil {
		t.Fatalf("second EnrichMetadata: %v", err)
	}
	s = loadItem(t, db, id)
	if s.genres.String != `["animation"]` || s.source.String != "tmdb" || !s.showID.Valid || s.series.String != "Foo" || s.manual != 1 {
		t.Errorf("after the corrected lookups: %+v, want TMDb's genres and show under the corrected name, still protected from rescans", s)
	}
	var rating string
	db.QueryRow(`SELECT content_rating FROM tv_shows WHERE id = ?`, s.showID.Int64).Scan(&rating)
	if rating != "TV-MA" {
		t.Errorf("show rating = %q, want TMDb's TV-MA", rating)
	}
}
