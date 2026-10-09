package metadata

// movieGenreTags and tvGenreTags map TMDb's own (stable, rarely-changed)
// numeric genre IDs onto this package's Tag taxonomy — verified against
// GET /genre/movie/list and GET /genre/tv/list. IDs with no entry here
// (TV Movie, News, Soap, Talk) aren't useful for channel matching and
// are simply dropped by tagsFromGenreIDs.
var movieGenreTags = map[int][]Tag{
	28:    {TagAction},
	12:    {TagAdventure},
	16:    {TagAnimation},
	35:    {TagComedy},
	80:    {TagCrime},
	99:    {TagDocumentary},
	18:    {TagDrama},
	10751: {TagFamily},
	14:    {TagFantasy},
	36:    {TagHistory},
	27:    {TagHorror},
	10402: {TagMusic},
	9648:  {TagMystery},
	10749: {TagRomance},
	878:   {TagSciFi},
	53:    {TagThriller},
	10752: {TagWar},
	37:    {TagWestern},
}

var tvGenreTags = map[int][]Tag{
	10759: {TagAction, TagAdventure},
	16:    {TagAnimation},
	35:    {TagComedy},
	80:    {TagCrime},
	99:    {TagDocumentary},
	18:    {TagDrama},
	10751: {TagFamily},
	10762: {TagKids},
	9648:  {TagMystery},
	10764: {TagReality},
	10765: {TagSciFi, TagFantasy},
	10768: {TagWar},
	37:    {TagWestern},
}
