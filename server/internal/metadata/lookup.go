package metadata

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
)

// mediaKind holds what differs between looking up a movie and a TV show.
type mediaKind struct {
	searchPath  string
	yearParam   string
	detailsPath string // "/movie" or "/tv"; the TMDb id is appended
	findKey     string // key of this kind's results in a /find response
}

var (
	kindMovie = mediaKind{"/search/movie", "primary_release_year", "/movie", "movie_results"}
	kindTV    = mediaKind{"/search/tv", "first_air_date_year", "/tv", "tv_results"}
)

// lookup finds the TMDb entry for raw, a title as written in a Plex-style
// folder or file name (see ParseName). It tries, most certain first:
//
//  1. a {tmdb-N} tag, fetched directly;
//  2. a {imdb-ttN} or {tvdb-N} tag, resolved through TMDb's /find;
//  3. a title search using the year, so same-named shows and remakes
//     resolve to the right one, then again without it (folder years are
//     sometimes a year off TMDb's), across a few spellings of the title.
//
// year, when > 0, overrides a year written in raw. No match is (nil, nil).
func (c *Client) lookup(ctx context.Context, kind mediaKind, raw string, year int) (*searchResult, error) {
	p := ParseName(raw)
	if year == 0 {
		year = p.Year
	}
	if p.TMDbID > 0 {
		if r, err := c.byTMDbID(ctx, kind, p.TMDbID); err == nil && r != nil {
			return r, nil
		}
	}
	if p.IMDbID != "" {
		if r, err := c.find(ctx, kind, p.IMDbID, "imdb_id"); err == nil && r != nil {
			return r, nil
		}
	}
	if p.TVDBID > 0 && kind == kindTV {
		if r, err := c.find(ctx, kind, strconv.Itoa(p.TVDBID), "tvdb_id"); err == nil && r != nil {
			return r, nil
		}
	}
	return c.searchLadder(ctx, kind, p.Title, year)
}

func (c *Client) searchLadder(ctx context.Context, kind mediaKind, title string, year int) (*searchResult, error) {
	years := []int{0}
	if year > 0 {
		years = []int{year, 0}
	}
	var titleOnly, anything *searchResult
	for _, variant := range titleVariants(title) {
		for _, y := range years {
			results, err := c.searchAll(ctx, kind, variant, y)
			if err != nil {
				return nil, err
			}
			if len(results) == 0 {
				continue
			}
			best, exact := pickBest(kind, results, title, year)
			if exact {
				return best, nil
			}
			if best != nil && titleOnly == nil {
				titleOnly = best
			}
			if anything == nil {
				anything = &results[0]
			}
		}
	}
	// A title match in the wrong year beats an arbitrary first result.
	if titleOnly != nil {
		return titleOnly, nil
	}
	return anything, nil
}

func (c *Client) searchAll(ctx context.Context, kind mediaKind, query string, year int) ([]searchResult, error) {
	params := url.Values{"query": {query}}
	if year > 0 {
		params.Set(kind.yearParam, strconv.Itoa(year))
	}
	var parsed searchResponse
	if err := c.get(ctx, kind.searchPath+"?"+params.Encode(), &parsed); err != nil {
		return nil, err
	}
	return parsed.Results, nil
}

// pickBest chooses among same-search results. exact is true when the
// result's title equals title and its year is within one of year (or year
// is unknown); best is then that result. Otherwise best is the first
// title-matching result in another year, or nil. Results carrying no
// title at all can't be judged and count as exact.
func pickBest(kind mediaKind, results []searchResult, title string, year int) (best *searchResult, exact bool) {
	want := normalizeForMatch(title)
	var exactAt, titleAt = -1, -1
	exactDiff := 99
	for i, r := range results {
		names := []string{r.Name, r.OriginalName, r.Title, r.OriginalTitle}
		if r.Name == "" && r.OriginalName == "" && r.Title == "" && r.OriginalTitle == "" {
			return &results[i], true
		}
		matched := false
		for _, n := range names {
			if n != "" && normalizeForMatch(n) == want {
				matched = true
				break
			}
		}
		if !matched {
			continue
		}
		if titleAt < 0 {
			titleAt = i
		}
		date := r.FirstAirDate
		if kind == kindMovie {
			date = r.ReleaseDate
		}
		ry := leadingYear(date)
		if year == 0 || ry == 0 {
			if exactDiff > 0 {
				exactAt, exactDiff = i, 0
			}
			continue
		}
		if d := absInt(ry - year); d <= 1 && d < exactDiff {
			exactAt, exactDiff = i, d
		}
	}
	if exactAt >= 0 {
		return &results[exactAt], true
	}
	if titleAt >= 0 {
		return &results[titleAt], false
	}
	return nil, false
}

// byTMDbID fetches one entry by its TMDb id, shaped like a search result.
func (c *Client) byTMDbID(ctx context.Context, kind mediaKind, id int) (*searchResult, error) {
	var d struct {
		searchResult
		Genres []struct {
			ID int `json:"id"`
		} `json:"genres"`
	}
	if err := c.get(ctx, fmt.Sprintf("%s/%d", kind.detailsPath, id), &d); err != nil {
		return nil, err
	}
	r := d.searchResult
	r.ID = id
	for _, g := range d.Genres {
		r.GenreIDs = append(r.GenreIDs, g.ID)
	}
	return &r, nil
}

// find resolves an external id (source is "imdb_id" or "tvdb_id").
func (c *Client) find(ctx context.Context, kind mediaKind, externalID, source string) (*searchResult, error) {
	var out map[string]json.RawMessage
	path := "/find/" + url.PathEscape(externalID) + "?external_source=" + source
	if err := c.get(ctx, path, &out); err != nil {
		return nil, err
	}
	var results []searchResult
	if raw, ok := out[kind.findKey]; ok {
		if err := json.Unmarshal(raw, &results); err != nil {
			return nil, fmt.Errorf("metadata: decode tmdb find: %w", err)
		}
	}
	if len(results) == 0 {
		return nil, nil
	}
	return &results[0], nil
}
