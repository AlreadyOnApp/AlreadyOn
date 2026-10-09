package config

import (
	"os"
	"time"
	_ "time/tzdata" // zone data travels with the binary, so TZ works on minimal hosts
)

// DefaultTimeZone applies when TZ is unset or not a known zone name.
const DefaultTimeZone = "America/New_York"

// ApplyTimeZone makes the TZ environment variable (or DefaultTimeZone) the
// zone for every time the server formats, daylight saving included. Go alone
// would silently fall back to UTC for a misspelt TZ.
func ApplyTimeZone() (name string, warning error) {
	return applyTimeZone(os.Getenv("TZ"))
}

func applyTimeZone(tz string) (string, error) {
	var warning error
	if tz == "" {
		tz = DefaultTimeZone
	}
	loc, err := time.LoadLocation(tz)
	if err != nil {
		warning = err
		tz = DefaultTimeZone
		loc, err = time.LoadLocation(tz)
		if err != nil {
			return "UTC", err
		}
	}
	time.Local = loc
	return tz, warning
}
