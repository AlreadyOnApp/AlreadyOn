package lineup

import (
	"context"
	"database/sql"
	"fmt"
	"sync/atomic"
	"testing"
	"time"
)

// insertUnclassified inserts an identified media file with no genre tags,
// as a scan without a metadata provider key leaves it.
func insertUnclassified(t *testing.T, db *sql.DB, mediaType, title string, season, episode int) int64 {
	t.Helper()
	ctx := context.Background()
	now := time.Now().Unix()
	path := fmt.Sprintf("/media/%s-%d.mkv", title, atomic.AddInt64(&testPathCounter, 1))
	var id int64
	if err := db.QueryRowContext(ctx, `
		INSERT INTO media_files (root_kind, path, size_bytes, modified_unix, first_seen_unix, last_seen_unix, duration_seconds, status)
		VALUES ('media', ?, 1024, ?, ?, ?, 1800, 'ok') RETURNING id`, path, now, now, now).Scan(&id); err != nil {
		t.Fatalf("insert media_files: %v", err)
	}
	var seriesTitle sql.NullString
	if mediaType == "episode" {
		seriesTitle = sql.NullString{String: title, Valid: true}
	}
	if _, err := db.ExecContext(ctx, `
		INSERT INTO identified_items (media_file_id, media_type, title, series_title, season_number, episode_number)
		VALUES (?, ?, ?, ?, ?, ?)`, id, mediaType, title, seriesTitle, season, episode); err != nil {
		t.Fatalf("insert identified_items: %v", err)
	}
	return id
}

func demoChannelIDs(t *testing.T, db *sql.DB) map[int]int64 {
	t.Helper()
	rows, err := db.Query(`SELECT number, id FROM channels WHERE number BETWEEN 196 AND 199`)
	if err != nil {
		t.Fatalf("query demo channels: %v", err)
	}
	defer rows.Close()
	out := map[int]int64{}
	for rows.Next() {
		var n int
		var id int64
		if err := rows.Scan(&n, &id); err != nil {
			t.Fatalf("scan: %v", err)
		}
		out[n] = id
	}
	return out
}

func TestSyncDemoChannels_CreatesFourWhenNothingClassified(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()
	insertChannel(t, db, 120, "Sci-Fi Network")
	a1 := insertUnclassified(t, db, "episode", "Alpha Show", 1, 1)
	a2 := insertUnclassified(t, db, "episode", "Alpha Show", 1, 2)
	b1 := insertUnclassified(t, db, "episode", "Bravo Show", 1, 1)
	m1 := insertUnclassified(t, db, "movie", "Apple Movie", 0, 0)
	m2 := insertUnclassified(t, db, "movie", "Banana Movie", 0, 0)

	n, err := SyncDemoChannels(ctx, db)
	if err != nil {
		t.Fatalf("SyncDemoChannels: %v", err)
	}
	if n != 4 {
		t.Fatalf("active demo channels = %d, want 4", n)
	}
	ids := demoChannelIDs(t, db)
	checks := map[int][]int64{196: {a1, a2}, 197: {b1}, 198: {m1}, 199: {m2}}
	for number, want := range checks {
		got := channelItemMediaFileIDs(t, db, ids[number])
		if fmt.Sprint(got) != fmt.Sprint(want) {
			t.Errorf("channel %d items = %v, want %v", number, got, want)
		}
	}

	// Idempotent: a second run keeps the same channels and lineups.
	if _, err := SyncDemoChannels(ctx, db); err != nil {
		t.Fatalf("second SyncDemoChannels: %v", err)
	}
	if again := demoChannelIDs(t, db); fmt.Sprint(again) != fmt.Sprint(ids) {
		t.Errorf("demo channel ids changed: %v -> %v", ids, again)
	}
	if got := channelItemMediaFileIDs(t, db, ids[196]); len(got) != 2 {
		t.Errorf("channel 196 items after rerun = %v, want 2 items", got)
	}
}

func TestSyncDemoChannels_OnlyCreatesChannelsWithContent(t *testing.T) {
	db := openTestDB(t)
	insertUnclassified(t, db, "movie", "Only Movie", 0, 0)

	n, err := SyncDemoChannels(context.Background(), db)
	if err != nil {
		t.Fatalf("SyncDemoChannels: %v", err)
	}
	if n != 1 {
		t.Errorf("active demo channels = %d, want 1", n)
	}
	if ids := demoChannelIDs(t, db); len(ids) != 1 || ids[198] == 0 {
		t.Errorf("demo channels = %v, want only 198", ids)
	}
}

func TestSyncDemoChannels_RemovedOnceRealChannelHasContent(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()
	insertChannel(t, db, 120, "Sci-Fi Network")
	insertUnclassified(t, db, "episode", "Plain Show", 1, 1)
	if n, err := SyncDemoChannels(ctx, db); err != nil || n == 0 {
		t.Fatalf("SyncDemoChannels = %d, %v; want demo channels", n, err)
	}

	insertClassified(t, db, "episode", "Space Show", 1, 1, 0, 600, `["scifi"]`)
	insertClassified(t, db, "episode", "Other Space Show", 1, 1, 0, 600, `["scifi"]`) // 2 distinct shows: channel stays visible
	if _, err := Populate(ctx, db); err != nil {
		t.Fatalf("Populate: %v", err)
	}
	n, err := SyncDemoChannels(ctx, db)
	if err != nil {
		t.Fatalf("SyncDemoChannels: %v", err)
	}
	if n != 0 {
		t.Errorf("active demo channels = %d, want 0", n)
	}
	if ids := demoChannelIDs(t, db); len(ids) != 0 {
		t.Errorf("demo channels still present: %v", ids)
	}
}

func TestSyncDemoChannels_HiddenRealChannelDoesNotRetireDemos(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()
	insertChannel(t, db, 120, "Sci-Fi Network")
	insertClassified(t, db, "episode", "Space Show", 1, 1, 0, 600, `["scifi"]`)
	if _, err := Populate(ctx, db); err != nil {
		t.Fatalf("Populate: %v", err)
	}
	if _, err := db.Exec(`UPDATE channels SET hidden = 1 WHERE number = 120`); err != nil {
		t.Fatalf("hide channel: %v", err)
	}
	var hidden bool
	if err := db.QueryRow(`SELECT hidden FROM channels WHERE number = 120`).Scan(&hidden); err != nil || !hidden {
		t.Fatalf("channel 120 hidden = %v (err %v), want true", hidden, err)
	}

	n, err := SyncDemoChannels(ctx, db)
	if err != nil {
		t.Fatalf("SyncDemoChannels: %v", err)
	}
	if n == 0 {
		t.Error("demo channels were retired by a channel the guide doesn't show")
	}
}

func TestSyncDemoChannels_PreservesAdminHiddenFlag(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()
	insertUnclassified(t, db, "movie", "Some Movie", 0, 0)
	if _, err := SyncDemoChannels(ctx, db); err != nil {
		t.Fatalf("SyncDemoChannels: %v", err)
	}
	if _, err := db.Exec(`UPDATE channels SET hidden = 1 WHERE number = 198`); err != nil {
		t.Fatalf("hide: %v", err)
	}
	if _, err := SyncDemoChannels(ctx, db); err != nil {
		t.Fatalf("SyncDemoChannels: %v", err)
	}
	var hidden bool
	if err := db.QueryRow(`SELECT hidden FROM channels WHERE number = 198`).Scan(&hidden); err != nil {
		t.Fatalf("read hidden: %v", err)
	}
	if !hidden {
		t.Error("admin-set hidden flag was reset")
	}
}
