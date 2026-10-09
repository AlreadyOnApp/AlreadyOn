package library

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// SaveProbeResult persists a ProbeResult onto the media_files row it was
// probed from. The row itself is created by the library scanner (#15);
// this only updates the probe-related columns of an existing row.
func SaveProbeResult(ctx context.Context, db *sql.DB, mediaFileID int64, probe ProbeResult) error {
	payload, err := json.Marshal(probe)
	if err != nil {
		return fmt.Errorf("library: marshal probe result for media file %d: %w", mediaFileID, err)
	}

	res, err := db.ExecContext(ctx, `
		UPDATE media_files
		SET duration_seconds = ?, container = ?, probe_json = ?, probed_at_unix = ?
		WHERE id = ?`,
		probe.DurationSeconds, probe.Container, string(payload), time.Now().Unix(), mediaFileID)
	if err != nil {
		return fmt.Errorf("library: persist probe result for media file %d: %w", mediaFileID, err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("library: persist probe result for media file %d: %w", mediaFileID, err)
	}
	if rows == 0 {
		return fmt.Errorf("library: persist probe result: no media_files row with id %d", mediaFileID)
	}
	return nil
}
