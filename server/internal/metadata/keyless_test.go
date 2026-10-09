package metadata

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func testKeylessClient(t *testing.T, handler http.HandlerFunc) *KeylessClient {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	c := NewKeylessClient()
	c.HTTPClient = server.Client()
	c.TVmazeBase = server.URL
	c.WikidataEndpoint = server.URL + "/sparql"
	c.MinInterval = map[string]time.Duration{}
	return c
}

func TestKeyless_ClassifyEpisode_MapsGenresAndPrefersYear(t *testing.T) {
	var gotUA string
	c := testKeylessClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotUA = r.UserAgent()
		fmt.Fprint(w, `[
			{"show":{"name":"The Office","type":"Scripted","language":"English","genres":["Comedy"],"premiered":"2001-07-09"}},
			{"show":{"name":"The Office","type":"Animation","language":"Japanese","genres":["Science-Fiction","Drama"],"premiered":"2005-03-24"}}
		]`)
	})

	tags, err := c.ClassifyEpisode(context.Background(), "The Office", 2005)
	if err != nil {
		t.Fatalf("ClassifyEpisode: %v", err)
	}
	want := []Tag{TagSciFi, TagDrama, TagAnimation, TagAnime}
	if fmt.Sprint(tags) != fmt.Sprint(want) {
		t.Errorf("tags = %v, want %v", tags, want)
	}
	if !strings.Contains(gotUA, "AlreadyOn") {
		t.Errorf("User-Agent = %q, want a descriptive AlreadyOn agent", gotUA)
	}

	tags, _ = c.ClassifyEpisode(context.Background(), "The Office", 0)
	if fmt.Sprint(tags) != fmt.Sprint([]Tag{TagComedy}) {
		t.Errorf("no-year tags = %v, want top result [comedy]", tags)
	}
}

func TestKeyless_ClassifyEpisode_NoResults(t *testing.T) {
	c := testKeylessClient(t, func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, `[]`) })
	tags, err := c.ClassifyEpisode(context.Background(), "nothing", 0)
	if err != nil || tags != nil {
		t.Errorf("got (%v, %v), want (nil, nil)", tags, err)
	}
}

func TestKeyless_ClassifyMovie_YearMustMatch(t *testing.T) {
	var gotQuery string
	c := testKeylessClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.Query().Get("query")
		fmt.Fprint(w, `{"results":{"bindings":[
			{"item":{"value":"Q1"},"year":{"value":"2022"},"labels":{"value":"comedy horror film|slasher film"}},
			{"item":{"value":"Q2"},"year":{"value":"1996"},"labels":{"value":"horror film|mystery film|teen film"}}
		]}}`)
	})

	tags, err := c.ClassifyMovie(context.Background(), `Scream "2" \ x`, 1997)
	if err != nil {
		t.Fatalf("ClassifyMovie: %v", err)
	}
	if fmt.Sprint(tags) != fmt.Sprint([]Tag{TagHorror, TagMystery}) {
		t.Errorf("tags = %v, want [horror mystery] (1996 film within a year of 1997)", tags)
	}
	if !strings.Contains(gotQuery, `mwapi:search "Scream \"2\" \\ x"`) {
		t.Errorf("title not safely escaped in SPARQL query:\n%s", gotQuery)
	}

	if tags, _ := c.ClassifyMovie(context.Background(), "Scream", 1980); tags != nil {
		t.Errorf("tags = %v, want nil when no film matches the year", tags)
	}
}

func TestKeyless_RetriesOnceOn429(t *testing.T) {
	var calls int32
	c := testKeylessClient(t, func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&calls, 1) == 1 {
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		fmt.Fprint(w, `[{"show":{"genres":["Western"]}}]`)
	})
	tags, err := c.ClassifyEpisode(context.Background(), "x", 0)
	if err != nil || fmt.Sprint(tags) != "[western]" {
		t.Errorf("got (%v, %v), want [western] after one retry", tags, err)
	}
	if calls != 2 {
		t.Errorf("calls = %d, want 2", calls)
	}
}

func TestTagsFromLabels(t *testing.T) {
	cases := map[string][]Tag{
		"science fiction film|superhero film|action thriller": {TagSciFi, TagAction, TagThriller},
		"animated feature film|children's film":               {TagAnimation, TagFamily, TagKids},
		"anti-war film":                                       {TagWar},
		"live-action film":                                    nil,
		"Star Wars fan film":                                  nil,
	}
	for in, want := range cases {
		got := tagsFromLabels(strings.Split(in, "|"))
		if fmt.Sprint(got) != fmt.Sprint(want) {
			t.Errorf("tagsFromLabels(%q) = %v, want %v", in, got, want)
		}
	}
}

// Best-effort genre fallback is permitted, but uncertain identity cannot assign
// a release era; provider dates must also be valid calendar dates.
func TestKeylessLookup_DatesRequireConfirmedMatch(t *testing.T) {
	c := testKeylessClient(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `[{"show":{"name":"Different Show","genres":["Comedy"],"premiered":"2009-12-31"}}]`)
	})
	result, err := c.LookupEpisode(context.Background(), "Missing Show", 0)
	if err != nil || result.ReleaseDate != "" || len(result.Tags) != 1 {
		t.Fatalf("fallback=%+v %v", result, err)
	}
	for _, date := range []string{"", "2009", "2010-02-30", "not-a-date"} {
		if validProviderDate(date) != "" {
			t.Fatalf("invalid date accepted: %s", date)
		}
	}
	if validProviderDate("2010-01-01T00:00:00Z") != "2010-01-01" {
		t.Fatal("valid provider timestamp rejected")
	}
}
