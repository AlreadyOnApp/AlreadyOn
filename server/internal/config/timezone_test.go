package config

import (
	"testing"
	"time"
)

func TestApplyTimeZone(t *testing.T) {
	saved := time.Local
	t.Cleanup(func() { time.Local = saved })
	winter := time.Date(2026, 1, 15, 17, 0, 0, 0, time.UTC)
	summer := time.Date(2026, 7, 15, 17, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		tz, want string
		warns    bool
		winter   int
		summer   int
	}{
		{"", "America/New_York", false, 12, 13},
		{"America/Los_Angeles", "America/Los_Angeles", false, 9, 10},
		{"Not/AZone", "America/New_York", true, 12, 13},
	} {
		name, warning := applyTimeZone(tc.tz)
		if name != tc.want || (warning != nil) != tc.warns || time.Local.String() != tc.want {
			t.Errorf("TZ=%q: got %q (%v), Local %q", tc.tz, name, warning, time.Local)
		}
		if winter.In(time.Local).Hour() != tc.winter || summer.In(time.Local).Hour() != tc.summer {
			t.Errorf("TZ=%q: daylight saving not applied", tc.tz)
		}
	}
}
