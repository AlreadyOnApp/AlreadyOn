// Package refreshlog remembers when the library was last scanned and the
// schedules last regenerated, however that was triggered.
package refreshlog

import (
	"context"
	"database/sql"
	"time"
)

type Kind string

const (
	Media    Kind = "media_at_unix"
	Schedule Kind = "schedule_at_unix"
)

// Record notes that a refresh of kind finished now. A failure to record is not
// worth failing the refresh for, so it is returned only for the caller to log.
func Record(ctx context.Context, db *sql.DB, kind Kind, now time.Time) error {
	column := string(kind)
	if kind != Media && kind != Schedule {
		return nil
	}
	_, err := db.ExecContext(ctx, `UPDATE refresh_times SET `+column+` = ? WHERE id = 1`, now.Unix())
	return err
}

// Times are when each kind last finished; the zero time means never.
type Times struct {
	Media    time.Time
	Schedule time.Time
}

func Get(ctx context.Context, db *sql.DB) (Times, error) {
	var media, schedule int64
	if err := db.QueryRowContext(ctx, `SELECT media_at_unix, schedule_at_unix FROM refresh_times WHERE id = 1`).Scan(&media, &schedule); err != nil {
		return Times{}, err
	}
	var t Times
	if media > 0 {
		t.Media = time.Unix(media, 0)
	}
	if schedule > 0 {
		t.Schedule = time.Unix(schedule, 0)
	}
	return t, nil
}
