package metadata

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sort"
	"strings"
	"testing"
)

// newTestClient points a Client at a mock TMDb server that returns body
// for any request, capturing the request path+query it received. The
// request is only made later (when the test calls a Classify* method),
// so the capture is returned as **http.Request — deref it afterward,
// not before.
func newTestClient(t *testing.T, body string) (*Client, **http.Request) {
	t.Helper()
	var captured *http.Request
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured = r
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)

	client := NewClient("test-key")
	client.HTTPClient = server.Client()
	client.BaseURL = server.URL

	return client, &captured
}

func sortedTags(tags []Tag) []string {
	strs := make([]string, len(tags))
	for i, t := range tags {
		strs[i] = string(t)
	}
	sort.Strings(strs)
	return strs
}

func TestClassifyMovie_MapsGenreIDsToTags(t *testing.T) {
	client, reqPtr := newTestClient(t, `{"results":[{"id":550,"genre_ids":[18,53,35]}]}`)

	tags, err := client.ClassifyMovie(context.Background(), "Fight Club", 1999)
	if err != nil {
		t.Fatalf("ClassifyMovie: %v", err)
	}
	if got := sortedTags(tags); !equalStrings(got, []string{"comedy", "drama", "thriller"}) {
		t.Errorf("tags = %v, want [comedy drama thriller]", got)
	}

	req := *reqPtr
	if req == nil || !strings.HasSuffix(req.URL.Path, "/search/movie") {
		t.Fatalf("request path = %v, want .../search/movie", req)
	}
	q, _ := url.ParseQuery(req.URL.RawQuery)
	if q.Get("query") != "Fight Club" || q.Get("primary_release_year") != "1999" {
		t.Errorf("query = %v, want query=Fight Club, primary_release_year=1999", q)
	}
}

func TestClassifyMovie_NoYear_OmitsYearParam(t *testing.T) {
	client, reqPtr := newTestClient(t, `{"results":[]}`)

	_, err := client.ClassifyMovie(context.Background(), "Some Movie", 0)
	if err != nil {
		t.Fatalf("ClassifyMovie: %v", err)
	}
	q, _ := url.ParseQuery((*reqPtr).URL.RawQuery)
	if q.Has("primary_release_year") {
		t.Errorf("query = %v, want no primary_release_year when year<=0", q)
	}
}

func TestClassifyMovie_NoResults_ReturnsNilNil(t *testing.T) {
	client, _ := newTestClient(t, `{"results":[]}`)

	tags, err := client.ClassifyMovie(context.Background(), "Nonexistent Movie", 2020)
	if err != nil || tags != nil {
		t.Errorf("ClassifyMovie = (%v, %v), want (nil, nil)", tags, err)
	}
}

func TestClassifyMovie_DisabledClient_ReturnsNilNilWithoutRequest(t *testing.T) {
	requested := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requested = true
	}))
	defer server.Close()

	client := NewClient("") // no API key: disabled
	tags, err := client.ClassifyMovie(context.Background(), "Anything", 2020)
	if err != nil || tags != nil {
		t.Errorf("ClassifyMovie = (%v, %v), want (nil, nil)", tags, err)
	}
	if requested {
		t.Error("disabled client made an HTTP request, want none")
	}
}

func TestClassifyEpisode_MapsGenreIDsToTags(t *testing.T) {
	client, reqPtr := newTestClient(t, `{"results":[{"id":1,"genre_ids":[18,80],"original_language":"en"}]}`)

	tags, err := client.ClassifyEpisode(context.Background(), "Some Show", 2019)
	if err != nil {
		t.Fatalf("ClassifyEpisode: %v", err)
	}
	if got := sortedTags(tags); !equalStrings(got, []string{"crime", "drama"}) {
		t.Errorf("tags = %v, want [crime drama]", got)
	}
	req := *reqPtr
	if !strings.HasSuffix(req.URL.Path, "/search/tv") {
		t.Errorf("request path = %s, want .../search/tv", req.URL.Path)
	}
	q, _ := url.ParseQuery(req.URL.RawQuery)
	if q.Get("first_air_date_year") != "2019" {
		t.Errorf("query = %v, want first_air_date_year=2019", q)
	}
}

func TestClassifyEpisode_JapaneseAnimation_AddsAnimeTag(t *testing.T) {
	client, _ := newTestClient(t, `{"results":[{"id":1,"genre_ids":[16],"original_language":"ja"}]}`)

	tags, err := client.ClassifyEpisode(context.Background(), "Some Anime", 2020)
	if err != nil {
		t.Fatalf("ClassifyEpisode: %v", err)
	}
	if got := sortedTags(tags); !equalStrings(got, []string{"animation", "anime"}) {
		t.Errorf("tags = %v, want [animation anime]", got)
	}
}

func TestClassifyEpisode_EnglishAnimation_NoAnimeTag(t *testing.T) {
	client, _ := newTestClient(t, `{"results":[{"id":1,"genre_ids":[16],"original_language":"en"}]}`)

	tags, err := client.ClassifyEpisode(context.Background(), "Some Cartoon", 2020)
	if err != nil {
		t.Fatalf("ClassifyEpisode: %v", err)
	}
	if got := sortedTags(tags); !equalStrings(got, []string{"animation"}) {
		t.Errorf("tags = %v, want [animation] (no anime tag for non-Japanese animation)", got)
	}
}

func TestSearch_NonOKStatus_ReturnsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	client := NewClient("bad-key")
	client.HTTPClient = server.Client()
	client.BaseURL = server.URL

	_, err := client.ClassifyMovie(context.Background(), "Anything", 2020)
	if err == nil {
		t.Error("err = nil, want an error for a non-200 response")
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// Sanity check that our fixture JSON actually parses the way the real
// TMDb response shape does, independent of this package's own decoding.
func TestSearchResponse_DecodesGenreIDs(t *testing.T) {
	var resp searchResponse
	if err := json.Unmarshal([]byte(`{"results":[{"id":1,"genre_ids":[1,2,3]}]}`), &resp); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if len(resp.Results) != 1 || len(resp.Results[0].GenreIDs) != 3 {
		t.Errorf("resp = %+v", resp)
	}
}

func TestFindMovieID_ReturnsMatchedID(t *testing.T) {
	client, _ := newTestClient(t, `{"results":[{"id":550,"genre_ids":[18]}]}`)
	id, err := client.FindMovieID(context.Background(), "Fight Club", 1999)
	if err != nil {
		t.Fatalf("FindMovieID: %v", err)
	}
	if id != 550 {
		t.Errorf("id = %d, want 550", id)
	}
}

func TestFindMovieID_NoResults_ReturnsZero(t *testing.T) {
	client, _ := newTestClient(t, `{"results":[]}`)
	id, err := client.FindMovieID(context.Background(), "Nonexistent", 2020)
	if err != nil || id != 0 {
		t.Errorf("FindMovieID = (%d, %v), want (0, nil)", id, err)
	}
}

func TestFindTVID_ReturnsMatchedID(t *testing.T) {
	client, _ := newTestClient(t, `{"results":[{"id":1399,"genre_ids":[18]}]}`)
	id, err := client.FindTVID(context.Background(), "Game of Thrones", 2011)
	if err != nil {
		t.Fatalf("FindTVID: %v", err)
	}
	if id != 1399 {
		t.Errorf("id = %d, want 1399", id)
	}
}

// pathRoutedServer serves a different body depending on the request
// path — MovieDetails/TVDetails each make two requests (details, then
// release_dates/content_ratings), unlike the single-request Classify*
// methods newTestClient covers.
func pathRoutedServer(t *testing.T, byPath map[string]string) *Client {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, ok := byPath[r.URL.Path]
		if !ok {
			t.Errorf("unexpected request path %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)
	client := NewClient("test-key")
	client.HTTPClient = server.Client()
	client.BaseURL = server.URL
	return client
}

func TestMovieDetails_ParsesOverviewReleaseDateAndUSCertification(t *testing.T) {
	client := pathRoutedServer(t, map[string]string{
		"/movie/550":               `{"overview":"An insomniac office worker...","release_date":"1999-10-15"}`,
		"/movie/550/release_dates": `{"results":[{"iso_3166_1":"US","release_dates":[{"certification":"","type":1},{"certification":"R","type":3}]}]}`,
	})
	d, err := client.MovieDetails(context.Background(), 550)
	if err != nil {
		t.Fatalf("MovieDetails: %v", err)
	}
	if d.Overview != "An insomniac office worker..." || d.ReleaseDate != "1999-10-15" || d.Certification != "R" {
		t.Errorf("d = %+v", d)
	}
}

func TestMovieDetails_NoUSTheatricalCertification_FallsBackToAnyType(t *testing.T) {
	client := pathRoutedServer(t, map[string]string{
		"/movie/1":               `{"overview":"x","release_date":"2001-01-01"}`,
		"/movie/1/release_dates": `{"results":[{"iso_3166_1":"US","release_dates":[{"certification":"PG-13","type":4}]}]}`,
	})
	d, err := client.MovieDetails(context.Background(), 1)
	if err != nil {
		t.Fatalf("MovieDetails: %v", err)
	}
	if d.Certification != "PG-13" {
		t.Errorf("Certification = %q, want PG-13 (fallback to non-theatrical release)", d.Certification)
	}
}

func TestMovieDetails_DisabledClientOrNoID_ReturnsZeroValue(t *testing.T) {
	client := NewClient("")
	d, err := client.MovieDetails(context.Background(), 550)
	if err != nil || d != (MovieDetails{}) {
		t.Errorf("MovieDetails = (%+v, %v), want zero value, nil", d, err)
	}

	client = NewClient("test-key")
	d, err = client.MovieDetails(context.Background(), 0)
	if err != nil || d != (MovieDetails{}) {
		t.Errorf("MovieDetails(id=0) = (%+v, %v), want zero value, nil", d, err)
	}
}

func TestTVDetails_ParsesOverviewDatesAndUSRating(t *testing.T) {
	client := pathRoutedServer(t, map[string]string{
		"/tv/1399":                 `{"overview":"Nine noble families...","first_air_date":"2011-04-17","last_air_date":"2019-05-19"}`,
		"/tv/1399/content_ratings": `{"results":[{"iso_3166_1":"GB","rating":"15"},{"iso_3166_1":"US","rating":"TV-MA"}]}`,
	})
	d, err := client.TVDetails(context.Background(), 1399)
	if err != nil {
		t.Fatalf("TVDetails: %v", err)
	}
	if d.Overview != "Nine noble families..." || d.FirstAirDate != "2011-04-17" || d.LastAirDate != "2019-05-19" || d.Certification != "TV-MA" {
		t.Errorf("d = %+v", d)
	}
}

func TestEpisodeDetails_ParsesOverviewAndAirDate(t *testing.T) {
	client := pathRoutedServer(t, map[string]string{
		"/tv/1399/season/1/episode/1": `{"overview":"Lord Eddard Stark...","air_date":"2011-04-17"}`,
	})
	d, err := client.EpisodeDetails(context.Background(), 1399, 1, 1)
	if err != nil {
		t.Fatalf("EpisodeDetails: %v", err)
	}
	if d.Overview != "Lord Eddard Stark..." || d.AirDate != "2011-04-17" {
		t.Errorf("d = %+v", d)
	}
}
