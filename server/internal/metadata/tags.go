package metadata

// TagInfo is a Tag with its display name, for UI pickers.
type TagInfo struct {
	ID    Tag
	Label string
}

// tagList is the full vocabulary in display order. Every Tag constant in
// tmdb.go must appear here.
var tagList = []TagInfo{
	{TagAction, "Action"},
	{TagAdventure, "Adventure"},
	{TagAnimation, "Animation"},
	{TagAnime, "Anime"},
	{TagComedy, "Comedy"},
	{TagCrime, "Crime"},
	{TagDocumentary, "Documentary"},
	{TagDrama, "Drama"},
	{TagFamily, "Family"},
	{TagFantasy, "Fantasy"},
	{TagHistory, "History"},
	{TagHorror, "Horror"},
	{TagKids, "Kids"},
	{TagMusic, "Music"},
	{TagMystery, "Mystery"},
	{TagReality, "Reality"},
	{TagRomance, "Romance"},
	{TagSciFi, "Sci-Fi"},
	{TagThriller, "Thriller"},
	{TagWar, "War"},
	{TagWestern, "Western"},
}

// Tags returns the genre vocabulary in display order.
func Tags() []TagInfo {
	return append([]TagInfo(nil), tagList...)
}

// ValidTag reports whether s is one of the known genre tags.
func ValidTag(s string) bool {
	for _, t := range tagList {
		if string(t.ID) == s {
			return true
		}
	}
	return false
}
