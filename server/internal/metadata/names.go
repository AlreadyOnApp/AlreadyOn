package metadata

import (
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

// ParsedName is a title as Plex, Sonarr and Radarr write it into folder and
// file names: "The Office (US) (2005) {tvdb-73244}". Providers only match
// the bare title, but the year, region and IDs are what tell apart shows
// that share one.
type ParsedName struct {
	Title   string
	Year    int    // 0 when absent
	Region  string // "US" in "The Office (US)"; "" when absent
	TMDbID  int
	IMDbID  string // "tt0372784"
	TVDBID  int
	Edition string // Radarr/Plex "{edition-Director's Cut}"; never used for lookups
}

var (
	// {imdb-tt123}, {tmdb-123}, {tvdb-123}, {edition-Name}; also the Jellyfin
	// style [imdbid-tt123]. Sonarr/Radarr write the braces form.
	idTagRe = regexp.MustCompile(`(?i)[{\[]\s*(imdb|tmdb|tvdb|edition)(?:id)?\s*[-=]\s*([^}\]]+?)\s*[}\]]`)
	// Any other leftover {...} tag, e.g. Radarr's {Release Group}.
	otherTagRe = regexp.MustCompile(`\{[^}]*\}`)

	trailingYearRe   = regexp.MustCompile(`\s*[(\[](\d{4})[)\]]\s*$`)
	trailingRegionRe = regexp.MustCompile(`\s*\(([A-Z]{2,3})\)\s*$`)
)

// ParseName splits raw into a title plus the year, region and provider IDs
// found in it. A name with none of them comes back as just Title.
func ParseName(raw string) ParsedName {
	var p ParsedName
	s := idTagRe.ReplaceAllStringFunc(raw, func(tag string) string {
		m := idTagRe.FindStringSubmatch(tag)
		kind, value := strings.ToLower(m[1]), strings.TrimSpace(m[2])
		switch kind {
		case "imdb":
			p.IMDbID = strings.ToLower(value)
		case "tmdb":
			p.TMDbID, _ = strconv.Atoi(value)
		case "tvdb":
			p.TVDBID, _ = strconv.Atoi(value)
		case "edition":
			p.Edition = value
		}
		return ""
	})
	s = strings.TrimSpace(otherTagRe.ReplaceAllString(s, ""))

	// Year and region may come in either order: "Show (2005) (US)" or
	// "Show (US) (2005)".
	for range 2 {
		if m := trailingYearRe.FindStringSubmatch(s); m != nil && p.Year == 0 && len(m[0]) < len(s) {
			p.Year, _ = strconv.Atoi(m[1])
			s = strings.TrimSpace(s[:len(s)-len(m[0])])
			continue
		}
		if m := trailingRegionRe.FindStringSubmatch(s); m != nil && p.Region == "" && len(m[0]) < len(s) {
			p.Region = m[1]
			s = strings.TrimSpace(s[:len(s)-len(m[0])])
		}
	}
	p.Title = s
	return p
}

// String writes p back in the same convention ParseName reads.
func (p ParsedName) String() string {
	parts := []string{p.Title}
	if p.Region != "" {
		parts = append(parts, "("+p.Region+")")
	}
	if p.Year > 0 {
		parts = append(parts, "("+strconv.Itoa(p.Year)+")")
	}
	if p.TMDbID > 0 {
		parts = append(parts, "{tmdb-"+strconv.Itoa(p.TMDbID)+"}")
	}
	if p.IMDbID != "" {
		parts = append(parts, "{imdb-"+p.IMDbID+"}")
	}
	if p.TVDBID > 0 {
		parts = append(parts, "{tvdb-"+strconv.Itoa(p.TVDBID)+"}")
	}
	return strings.Join(parts, " ")
}

// SameTitle reports whether two names are the same title once year, region,
// IDs, case and punctuation are ignored.
func SameTitle(a, b string) bool {
	na, nb := normalizeForMatch(ParseName(a).Title), normalizeForMatch(ParseName(b).Title)
	return na != "" && na == nb
}

// normalizeForMatch reduces a title to lowercase letters and digits, with
// "&" read as "and" and a leading "the" dropped, so "Love, Death & Robots"
// equals "Love Death and Robots" and "The Office" equals "Office".
func normalizeForMatch(s string) string {
	s = strings.ToLower(strings.ReplaceAll(s, "&", " and "))
	var b strings.Builder
	for _, r := range s {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(r)
		case unicode.IsSpace(r):
			b.WriteByte(' ')
		}
	}
	out := strings.Join(strings.Fields(b.String()), " ")
	out = strings.TrimPrefix(out, "the ")
	return strings.ReplaceAll(out, " ", "")
}

// titleVariants lists spellings of title worth searching, most literal
// first: as written, "&" and "and" swapped, " - " read as the ":" Sonarr
// and Radarr replace in file names, and the part before a subtitle
// separator ("Show: The Subtitle" -> "Show").
func titleVariants(title string) []string {
	variants := []string{title}
	add := func(v string) {
		v = strings.TrimSpace(v)
		if len(v) < 3 {
			return
		}
		for _, existing := range variants {
			if strings.EqualFold(existing, v) {
				return
			}
		}
		variants = append(variants, v)
	}
	switch {
	case strings.Contains(title, "&"):
		add(strings.ReplaceAll(title, "&", "and"))
	case strings.Contains(strings.ToLower(title), " and "):
		add(regexp.MustCompile(`(?i)\s+and\s+`).ReplaceAllString(title, " & "))
	}
	if strings.Contains(title, " - ") {
		add(strings.ReplaceAll(title, " - ", ": "))
	}
	for _, sep := range []string{":", " - "} {
		if before, _, ok := strings.Cut(title, sep); ok {
			add(before)
		}
	}
	return variants
}
