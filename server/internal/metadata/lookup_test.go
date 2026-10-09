package metadata

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// lookupServer answers TMDb requests from handlers keyed by URL path and
// records every request it saw.
func lookupServer(t *testing.T, handlers map[string]func(q url.Values) string) (*Client, *[]string) {
	t.Helper()
	var seen []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = append(seen, r.URL.String())
		h, ok := handlers[r.URL.Path]
		if !ok {
			http.NotFound(w, r)
			return
		}
		w.Write([]byte(h(r.URL.Query())))
	}))
	t.Cleanup(server.Close)
	return &Client{APIKey: "k", BaseURL: server.URL, HTTPClient: server.Client()}, &seen
}

func TestFindTVID_YearSuffixedTitle_SearchesWithYearThenWithout(t *testing.T) {
	c, seen := lookupServer(t, map[string]func(url.Values) string{
		"/search/tv": func(q url.Values) string {
			if q.Get("query") == "Love, Death & Robots" && q.Get("first_air_date_year") == "" {
				return `{"results":[{"id":86831,"name":"Love, Death & Robots","first_air_date":"2019-03-15"}]}`
			}
			return `{"results":[]}`
		},
	})
	id, err := c.FindTVID(context.Background(), "Love, Death & Robots (2019)", 0)
	if err != nil || id != 86831 {
		t.Fatalf("FindTVID = %d, %v; want 86831", id, err)
	}
	if len(*seen) != 2 || !strings.Contains((*seen)[0], "first_air_date_year=2019") {
		t.Errorf("requests = %v, want one filtered to 2019 then one without a year", *seen)
	}
}

func TestFindTVID_SameNamedShows_YearPicksTheRightOne(t *testing.T) {
	// TMDb ignores the year filter here, as when a folder year is off by a
	// few years: ranking by year must still pick the 2005 remake.
	c, _ := lookupServer(t, map[string]func(url.Values) string{
		"/search/tv": func(url.Values) string {
			return `{"results":[
				{"id":2996,"name":"The Office","first_air_date":"2001-07-09"},
				{"id":2316,"name":"The Office","first_air_date":"2005-03-24"}]}`
		},
	})
	id, err := c.FindTVID(context.Background(), "The Office (US) (2005)", 0)
	if err != nil || id != 2316 {
		t.Fatalf("FindTVID = %d, %v; want 2316 (the 2005 show)", id, err)
	}
}

func TestFindTVID_WrongYearInFolder_StillTakesTheTitleMatch(t *testing.T) {
	c, _ := lookupServer(t, map[string]func(url.Values) string{
		"/search/tv": func(q url.Values) string {
			return `{"results":[{"id":7,"name":"Some Show","first_air_date":"2015-01-01"}]}`
		},
	})
	id, err := c.FindTVID(context.Background(), "Some Show (2019)", 0)
	if err != nil || id != 7 {
		t.Fatalf("FindTVID = %d, %v; want 7 (title matches even though the year does not)", id, err)
	}
}

func TestFindTVID_IMDbTag_ResolvesThroughFindWithoutSearching(t *testing.T) {
	c, seen := lookupServer(t, map[string]func(url.Values) string{
		"/find/tt9561862": func(q url.Values) string {
			if q.Get("external_source") != "imdb_id" {
				t.Errorf("external_source = %q", q.Get("external_source"))
			}
			return `{"movie_results":[],"tv_results":[{"id":86831,"name":"Love, Death & Robots"}]}`
		},
	})
	id, err := c.FindTVID(context.Background(), "Some Other Name (2019) {imdb-tt9561862}", 0)
	if err != nil || id != 86831 {
		t.Fatalf("FindTVID = %d, %v; want 86831", id, err)
	}
	if len(*seen) != 1 {
		t.Errorf("requests = %v, want only the /find call", *seen)
	}
}

func TestFindTVID_TVDBTag_ResolvesThroughFind(t *testing.T) {
	c, _ := lookupServer(t, map[string]func(url.Values) string{
		"/find/73244": func(q url.Values) string {
			if q.Get("external_source") != "tvdb_id" {
				t.Errorf("external_source = %q", q.Get("external_source"))
			}
			return `{"tv_results":[{"id":2316}]}`
		},
	})
	id, err := c.FindTVID(context.Background(), "The Office (US) (2005) {tvdb-73244}", 0)
	if err != nil || id != 2316 {
		t.Fatalf("FindTVID = %d, %v; want 2316", id, err)
	}
}

func TestFindMovieID_TMDbTag_UsedDirectlyAndClassifiedFromItsGenres(t *testing.T) {
	c, seen := lookupServer(t, map[string]func(url.Values) string{
		"/movie/272": func(url.Values) string { return `{"id":272,"title":"Batman Begins","genres":[{"id":28}]}` },
	})
	tags, err := c.ClassifyMovie(context.Background(), "Batman Begins (2005) {tmdb-272}", 2005)
	if err != nil || len(tags) != 1 || tags[0] != TagAction {
		t.Fatalf("ClassifyMovie = %v, %v; want [action]", tags, err)
	}
	if len(*seen) != 1 {
		t.Errorf("requests = %v, want only the direct /movie/272 fetch", *seen)
	}
}

func TestFindMovieID_BadIDTag_FallsBackToSearch(t *testing.T) {
	c, _ := lookupServer(t, map[string]func(url.Values) string{
		"/search/movie": func(url.Values) string {
			return `{"results":[{"id":272,"title":"Batman Begins","release_date":"2005-06-10"}]}`
		},
	})
	id, err := c.FindMovieID(context.Background(), "Batman Begins (2005) {tmdb-999999999}", 0)
	if err != nil || id != 272 {
		t.Fatalf("FindMovieID = %d, %v; want 272 via search", id, err)
	}
}

func TestFindMovieID_YearOffByOne_Accepted(t *testing.T) {
	c, _ := lookupServer(t, map[string]func(url.Values) string{
		"/search/movie": func(q url.Values) string {
			if q.Get("primary_release_year") != "" {
				return `{"results":[]}`
			}
			return `{"results":[
				{"id":1,"title":"Alien","release_date":"1979-05-25"},
				{"id":2,"title":"Alien","release_date":"2003-01-01"}]}`
		},
	})
	id, err := c.FindMovieID(context.Background(), "Alien", 1980)
	if err != nil || id != 1 {
		t.Fatalf("FindMovieID = %d, %v; want 1 (1979 is within a year of 1980)", id, err)
	}
}

func TestFindTVID_AmpersandSpelling_RetriedAsAnd(t *testing.T) {
	c, _ := lookupServer(t, map[string]func(url.Values) string{
		"/search/tv": func(q url.Values) string {
			if q.Get("query") == "Beauty and the Beast" {
				return `{"results":[{"id":5,"name":"Beauty and the Beast"}]}`
			}
			return `{"results":[]}`
		},
	})
	id, err := c.FindTVID(context.Background(), "Beauty & the Beast", 0)
	if err != nil || id != 5 {
		t.Fatalf("FindTVID = %d, %v; want 5", id, err)
	}
}
