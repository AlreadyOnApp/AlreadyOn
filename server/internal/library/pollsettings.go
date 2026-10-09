package library

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// PollSettings controls how often the background scanner (see main.go's
// polling loop) re-checks configured media roots for new content, on
// top of the one-time startup scan.
type PollSettings struct {
	// IntervalMinutes is one of 15, 30, 60, 120, 240, 480, 720, or 1440.
	IntervalMinutes int
}

var validPollIntervalMinutes = map[int]bool{
	15: true, 30: true, 60: true, 120: true, 240: true, 480: true, 720: true, 1440: true,
}

var ErrInvalidPollInterval = errors.New("library: invalid poll interval minutes")

// GetPollSettings returns the current settings. There's always exactly
// one row (seeded by migration 0016), so a missing row is a schema bug,
// not a valid "unconfigured" state.
func GetPollSettings(ctx context.Context, db *sql.DB) (PollSettings, error) {
	var s PollSettings
	err := db.QueryRowContext(ctx, `SELECT poll_interval_minutes FROM library_settings WHERE id = 1`).
		Scan(&s.IntervalMinutes)
	if err != nil {
		return PollSettings{}, fmt.Errorf("library: get poll settings: %w", err)
	}
	return s, nil
}

// SavePollSettings validates and persists s, replacing the single
// settings row.
func SavePollSettings(ctx context.Context, db *sql.DB, s PollSettings) error {
	if !validPollIntervalMinutes[s.IntervalMinutes] {
		return ErrInvalidPollInterval
	}
	_, err := db.ExecContext(ctx, `
		UPDATE library_settings SET poll_interval_minutes = ?, updated_at_unix = unixepoch() WHERE id = 1`,
		s.IntervalMinutes)
	if err != nil {
		return fmt.Errorf("library: save poll settings: %w", err)
	}
	return nil
}
