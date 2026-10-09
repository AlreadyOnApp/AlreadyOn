package metadata

import "testing"

func TestParseName(t *testing.T) {
	cases := []struct {
		in   string
		want ParsedName
	}{
		{"Love, Death & Robots (2019)", ParsedName{Title: "Love, Death & Robots", Year: 2019}},
		{"Love, Death & Robots", ParsedName{Title: "Love, Death & Robots"}},
		{"1883", ParsedName{Title: "1883"}},
		{"(500) Days of Summer (2009)", ParsedName{Title: "(500) Days of Summer", Year: 2009}},
		{"Show (2019) Extended", ParsedName{Title: "Show (2019) Extended"}},
		{"The Office (US) (2005) {tvdb-73244}", ParsedName{Title: "The Office", Region: "US", Year: 2005, TVDBID: 73244}},
		{"The Office (2005) (US)", ParsedName{Title: "The Office", Region: "US", Year: 2005}},
		{"Batman Begins (2005) {imdb-tt0372784}", ParsedName{Title: "Batman Begins", Year: 2005, IMDbID: "tt0372784"}},
		{"Batman Begins (2005) {tmdb-272}", ParsedName{Title: "Batman Begins", Year: 2005, TMDbID: 272}},
		{"Blade Runner (1982) {edition-Director's Cut}", ParsedName{Title: "Blade Runner", Year: 1982, Edition: "Director's Cut"}},
		{"Movie (2010) [imdbid-tt1375666]", ParsedName{Title: "Movie", Year: 2010, IMDbID: "tt1375666"}},
		{"Movie (2010) {Some-Group}", ParsedName{Title: "Movie", Year: 2010}},
	}
	for _, c := range cases {
		if got := ParseName(c.in); got != c.want {
			t.Errorf("ParseName(%q) = %+v, want %+v", c.in, got, c.want)
		}
	}
}

func TestParsedName_StringRoundTrips(t *testing.T) {
	in := "The Office (US) (2005) {imdb-tt0386676} {tvdb-73244}"
	if got := ParseName(in).String(); got != in {
		t.Errorf("String() = %q, want %q", got, in)
	}
}

func TestSameTitle(t *testing.T) {
	if !SameTitle("Love, Death & Robots (2019)", "Love Death and Robots {imdb-tt9561862}") {
		t.Error("titles differing only by year, tags and punctuation should match")
	}
	if !SameTitle("The Office (US)", "Office") {
		t.Error("a leading 'The' should not matter")
	}
	if SameTitle("Show A", "Show B") || SameTitle("", "") {
		t.Error("different or empty titles should not match")
	}
}

func TestTitleVariants(t *testing.T) {
	got := titleVariants("Love, Death & Robots")
	if len(got) != 2 || got[1] != "Love, Death and Robots" {
		t.Errorf("variants = %v, want the literal title then the 'and' spelling", got)
	}
	got = titleVariants("Star Trek: Deep Space Nine")
	if len(got) != 2 || got[1] != "Star Trek" {
		t.Errorf("variants = %v, want the part before the subtitle as a second try", got)
	}
	got = titleVariants("Star Trek - Deep Space Nine")
	if len(got) < 2 || got[1] != "Star Trek: Deep Space Nine" {
		t.Errorf("variants = %v, want the colon spelling second", got)
	}
}
