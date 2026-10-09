package playback

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Preserve the muxer's first date before HLS drops old playlist entries. Clients
// can then map any later sliding-window fragment onto the persisted schedule,
// including a worker prepared before its entry starts.
func captureHLSOrigin(dir string) (bool, error) {
	// A ladder's master playlist carries no dates; its low rung is the first to
	// have a segment, so read that.
	raw, err := os.ReadFile(filepath.Join(dir, "stream-0.m3u8"))
	if errors.Is(err, os.ErrNotExist) {
		raw, err = os.ReadFile(filepath.Join(dir, "index.m3u8"))
	}
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	for _, line := range strings.Split(string(raw), "\n") {
		if value, ok := strings.CutPrefix(line, "#EXT-X-PROGRAM-DATE-TIME:"); ok {
			date, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(value))
			if err != nil {
				date, err = time.Parse("2006-01-02T15:04:05.999999999Z0700", strings.TrimSpace(value))
			}
			if err != nil {
				return false, err
			}
			return true, os.WriteFile(filepath.Join(dir, "origin"), []byte(date.Format(time.RFC3339Nano)), 0600)
		}
	}
	return false, nil
}

func HLSOrigin(dir string) (time.Time, error) {
	raw, err := os.ReadFile(filepath.Join(dir, "origin"))
	if err != nil {
		return time.Time{}, err
	}
	return time.Parse(time.RFC3339Nano, string(raw))
}
