package schedule

import "time"

// Clock is injected so schedule generation and current-offset
// calculations stay deterministic and testable.
// See docs/PROJECT_BRIEF.md section 11 (inject clocks and random seeds).
type Clock interface {
	Now() time.Time
}

// SystemClock implements Clock using the real wall clock, in UTC.
type SystemClock struct{}

// Now returns the current time in UTC.
func (SystemClock) Now() time.Time { return time.Now().UTC() }

// FixedClock implements Clock by always returning the same instant —
// used in tests for deterministic schedule generation, and to simulate
// "time passing" between two reads by constructing a second FixedClock
// with a later Time (e.g. #23's restart-continuity test).
type FixedClock struct {
	Time time.Time
}

// Now returns c.Time, in UTC.
func (c FixedClock) Now() time.Time { return c.Time.UTC() }
