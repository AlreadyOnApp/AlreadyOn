package playback

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"
)

// LoadCapacity reads what the server last measured about itself.
func LoadCapacity(ctx context.Context, db *sql.DB) (CapacityRecord, error) {
	var r CapacityRecord
	var measured, processesAt int64
	var samples string
	err := db.QueryRowContext(ctx, `SELECT measured_at_unix, encoder, samples, max_height, max_fps, processes, processes_at_unix
		FROM server_capacity WHERE id = 1`).Scan(&measured, &r.Encoder, &samples, &r.MaxHeight, &r.MaxFPS, &r.Processes, &processesAt)
	if err != nil {
		return CapacityRecord{}, err
	}
	if measured > 0 {
		r.MeasuredAt = time.Unix(measured, 0)
	}
	if processesAt > 0 {
		r.ProcessesAt = time.Unix(processesAt, 0)
	}
	_ = json.Unmarshal([]byte(samples), &r.Samples) // damaged samples only hide the table
	return r, nil
}

// SaveCapacity stores the measurement for every viewer and restart.
func SaveCapacity(ctx context.Context, db *sql.DB, r CapacityRecord) error {
	samples, err := json.Marshal(r.Samples)
	if err != nil {
		return err
	}
	_, err = db.ExecContext(ctx, `UPDATE server_capacity SET measured_at_unix = ?, encoder = ?, samples = ?, max_height = ?, max_fps = ?,
		processes = ?, processes_at_unix = ? WHERE id = 1`,
		r.MeasuredAt.Unix(), r.Encoder, string(samples), r.MaxHeight, r.MaxFPS, r.Processes, r.ProcessesAt.Unix())
	return err
}
