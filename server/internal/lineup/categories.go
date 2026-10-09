package lineup

// Zero bounds leave that side unbounded; unknown-year handling belongs to filters.
type eraBound struct {
	MinYear int
	MaxYear int
}

var (
	modernEra    = eraBound{MinYear: 2010}
	nostalgiaEra = eraBound{MaxYear: 2009}
)
