package schedule

// CommercialRule configures how often commercial breaks are inserted
// into a channel's schedule. TODO: implement (Epic 3: commercial break
// insertion); avoid hardcoding assumptions about commercial duration.
type CommercialRule struct {
	ChannelID      string
	EveryNPrograms int
}
