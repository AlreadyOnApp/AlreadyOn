// Package loudness measures the playback audio track without modifying media.
package loudness

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
)

const Version = 2
const fullFileVersion = 1
const Target = -16.0
const Peak = -1.5
const Downmix = "aformat=sample_rates=48000:channel_layouts=stereo"

type Measurement struct {
	Integrated float64 `json:"integrated"`
	TruePeak   float64 `json:"truePeak"`
	Range      float64 `json:"range"`
	Threshold  float64 `json:"threshold"`
	Offset     float64 `json:"offset"`
	GainDB     float64 `json:"gainDb"`
}

func (m Measurement) Valid() bool {
	for _, v := range []float64{m.Integrated, m.TruePeak, m.Range, m.Threshold, m.Offset, m.GainDB} {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return false
		}
	}
	return m.Integrated >= -99 && m.Integrated <= 0 && m.TruePeak >= -99 && m.TruePeak <= 99 && m.Range >= 0 && m.Range <= 99 && m.Threshold >= -99 && m.Threshold <= 0 && math.Abs(m.Offset) <= 99
}
func (m Measurement) Filter() string {
	return fmt.Sprintf("%s,loudnorm=I=-16:TP=-1.5:LRA=50:measured_I=%.6f:measured_TP=%.6f:measured_LRA=%.6f:measured_thresh=%.6f:offset=%.6f:linear=true", Downmix, m.Integrated, m.TruePeak, m.Range, m.Threshold, m.Offset)
}
func parse(raw string) (*Measurement, string, error) {
	start, end := strings.LastIndex(raw, "{"), strings.LastIndex(raw, "}")
	if start < 0 || end < start {
		return nil, "failed", errors.New("FFmpeg did not return loudness measurements")
	}
	var fields map[string]string
	if err := json.Unmarshal([]byte(raw[start:end+1]), &fields); err != nil {
		return nil, "failed", err
	}
	if fields["input_i"] == "-inf" && fields["input_tp"] == "-inf" {
		return nil, "silent", nil
	}
	m := &Measurement{}
	for key, out := range map[string]*float64{"input_i": &m.Integrated, "input_tp": &m.TruePeak, "input_lra": &m.Range, "input_thresh": &m.Threshold, "target_offset": &m.Offset} {
		value, err := strconv.ParseFloat(fields[key], 64)
		if err != nil {
			return nil, "failed", fmt.Errorf("invalid loudness field %s", key)
		}
		*out = value
	}
	m.GainDB = Target - m.Integrated
	if !m.Valid() {
		return nil, "failed", errors.New("invalid or non-finite loudness measurement")
	}
	return m, "ready", nil
}

// Lookup rejects stale measurements and only enables normalization for the exact
// track/file analyzed. Missing/failed/silent files retain normal playback.
func Lookup(ctx context.Context, db *sql.DB, id int64, path string, track int) (*Measurement, error) {
	var raw string
	var size, ns int64
	err := db.QueryRowContext(ctx, `SELECT lm.measurement,lm.size_bytes,lm.modified_ns FROM loudness_measurements lm JOIN media_files mf ON mf.id=lm.media_file_id JOIN loudness_settings s ON s.id=1 WHERE s.enabled=1 AND lm.media_file_id=? AND lm.stream_index=? AND lm.version IN (?,?) AND lm.status='ready' AND mf.status='ok' AND mf.size_bytes=lm.size_bytes AND mf.modified_unix=lm.modified_unix`, id, track, fullFileVersion, Version).Scan(&raw, &size, &ns)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(path)
	if err != nil || info.Size() != size || info.ModTime().UnixNano() != ns {
		return nil, nil
	}
	var m Measurement
	if err = json.Unmarshal([]byte(raw), &m); err != nil || !m.Valid() {
		return nil, nil
	}
	return &m, nil
}

type Status struct {
	Enabled     bool   `json:"enabled"`
	Running     bool   `json:"running"`
	CurrentFile string `json:"currentFile"`
	// Active is how many files are being measured now; Workers is how many the
	// governor currently allows and CPUPercent the host CPU it last read.
	Active     int `json:"active"`
	Workers    int `json:"workers"`
	CPUPercent int `json:"cpuPercent"`
	// LimitedBy reports "cpu", "disk", "ram", "network", "swap", "playback",
	// "recovery", "metrics" or empty.
	LimitedBy string `json:"limitedBy"`
	Ready     int    `json:"ready"`
	Pending   int    `json:"pending"`
	Skipped   int    `json:"skipped"`
	Failed    int    `json:"failed"`
	LastError string `json:"lastError,omitempty"`
}

func Get(ctx context.Context, db *sql.DB) (Status, error) {
	var s Status
	err := db.QueryRowContext(ctx, `SELECT enabled,(enabled=1 AND running=1 AND heartbeat>=unixepoch()-10),CASE WHEN enabled=1 AND heartbeat>=unixepoch()-10 THEN current_file ELSE '' END FROM loudness_settings WHERE id=1`).Scan(&s.Enabled, &s.Running, &s.CurrentFile)
	if err != nil {
		return s, err
	}
	if s.Running {
		inflight.Lock()
		s.Active = len(inflight.files)
		inflight.Unlock()
		if g := governed.Load(); g != nil {
			s.Workers, s.CPUPercent, s.LimitedBy = g.Target(), int(g.CPU()+0.5), g.Limit()
		}
	}
	rows, err := db.QueryContext(ctx, `SELECT CASE WHEN lm.media_file_id IS NULL OR lm.size_bytes!=mf.size_bytes OR lm.modified_unix!=mf.modified_unix OR lm.version NOT IN (?,?) THEN 'pending' ELSE lm.status END,COUNT(*) FROM media_files mf LEFT JOIN loudness_measurements lm ON lm.media_file_id=mf.id WHERE mf.status='ok' GROUP BY 1`, fullFileVersion, Version)
	if err != nil {
		return s, err
	}
	for rows.Next() {
		var state string
		var n int
		if err = rows.Scan(&state, &n); err != nil {
			rows.Close()
			return s, err
		}
		switch state {
		case "ready":
			s.Ready += n
		case "failed":
			s.Failed += n
		case "silent", "no-audio":
			s.Skipped += n
		default:
			s.Pending += n
		}
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return s, err
	}
	_ = db.QueryRowContext(ctx, `SELECT error FROM loudness_measurements WHERE status='failed' ORDER BY analyzed_at DESC LIMIT 1`).Scan(&s.LastError)
	return s, nil
}
func SetEnabled(ctx context.Context, db *sql.DB, enabled bool) error {
	_, err := db.ExecContext(ctx, `UPDATE loudness_settings SET generation=generation+CASE WHEN enabled!=? THEN 1 ELSE 0 END,enabled=? WHERE id=1`, enabled, enabled)
	return err
}
func Request(ctx context.Context, db *sql.DB) error {
	res, err := db.ExecContext(ctx, `UPDATE loudness_settings SET generation=generation+1 WHERE id=1 AND enabled=1`)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return errors.New("enable Loudness Detection before running analysis")
	}
	return nil
}
