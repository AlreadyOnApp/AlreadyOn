package library

// Correction overrides an automatic Identification for one media item
// and must survive rescans. TODO: implement (Epic 2: manual metadata
// correction API and persistence across rescans).
type Correction struct {
	MediaID        string
	Identification Identification
}
