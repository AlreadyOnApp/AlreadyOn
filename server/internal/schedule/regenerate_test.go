package schedule

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"
)

func countScheduleEntries(t *testing.T, db *sql.DB, channelID int64) int {
	t.Helper()
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM schedule_entries WHERE channel_id = ?`, channelID).Scan(&n); err != nil {
		t.Fatalf("count schedule_entries: %v", err)
	}
	return n
}

func TestRegenerate_FillsThroughLeadingWindow(t *testing.T) {
	db := openTestDB(t)
	channelID := setupChannel(t, db, 1000, 100, 200, 300) // 600s loop

	if err := Regenerate(context.Background(), db, channelID, testAnchor); err != nil {
		t.Fatalf("Regenerate: %v", err)
	}

	var maxEndsAtUnix int64
	if err := db.QueryRow(`SELECT MAX(ends_at_unix) FROM schedule_entries WHERE channel_id = ?`, channelID).Scan(&maxEndsAtUnix); err != nil {
		t.Fatalf("query max ends_at: %v", err)
	}
	maxEndsAt := time.Unix(maxEndsAtUnix, 0).UTC()
	wantAtLeast := testAnchor.Add(LeadingWindow)
	if maxEndsAt.Before(wantAtLeast) {
		t.Errorf("last entry ends at %v, want at least %v (LeadingWindow ahead of the anchor)", maxEndsAt, wantAtLeast)
	}
}

func TestRegenerate_NeverChangesTheCurrentlyAiringEntry(t *testing.T) {
	db := openTestDB(t)
	channelID := setupChannel(t, db, 1000, 100, 200, 300)
	if err := Regenerate(context.Background(), db, channelID, testAnchor); err != nil {
		t.Fatalf("Regenerate (1st): %v", err)
	}

	clock := FixedClock{Time: testAnchor}
	before, _, err := Current(context.Background(), db, clock, channelID)
	if err != nil {
		t.Fatalf("Current (before): %v", err)
	}

	// Regenerating again immediately (nothing about the lineup changed)
	// must leave the entry actually airing right now untouched.
	if err := Regenerate(context.Background(), db, channelID, testAnchor); err != nil {
		t.Fatalf("Regenerate (2nd): %v", err)
	}
	after, _, err := Current(context.Background(), db, clock, channelID)
	if err != nil {
		t.Fatalf("Current (after): %v", err)
	}
	if before != after {
		t.Errorf("the currently-airing entry changed across a redundant regenerate: before=%+v after=%+v", before, after)
	}
}

func TestRegenerate_ReplacesUnavailableCurrentEntry(t *testing.T) {
	for _, status := range []string{"missing", "corrupt", "unreadable"} {
		t.Run(status, func(t *testing.T) {
			db := openTestDB(t)
			channelID := setupChannel(t, db, 1000, 100, 200)
			ctx := context.Background()
			if err := Regenerate(ctx, db, channelID, testAnchor); err != nil {
				t.Fatal(err)
			}
			var unavailable int64
			if err := db.QueryRow(`SELECT media_file_id FROM schedule_entries WHERE channel_id=? AND starts_at_unix<=? AND ends_at_unix>?`, channelID, testAnchor.Unix(), testAnchor.Unix()).Scan(&unavailable); err != nil {
				t.Fatal(err)
			}
			if _, err := db.Exec(`UPDATE media_files SET status=? WHERE id=?`, status, unavailable); err != nil {
				t.Fatal(err)
			}
			if err := Regenerate(ctx, db, channelID, testAnchor); err != nil {
				t.Fatal(err)
			}
			var current int64
			if err := db.QueryRow(`SELECT media_file_id FROM schedule_entries WHERE channel_id=? AND starts_at_unix<=? AND ends_at_unix>?`, channelID, testAnchor.Unix(), testAnchor.Unix()).Scan(&current); err != nil {
				t.Fatal(err)
			}
			if current == unavailable {
				t.Fatal("unavailable source is still airing after regeneration")
			}
			var stale int
			if err := db.QueryRow(`SELECT count(*) FROM schedule_entries WHERE channel_id=? AND media_file_id=? AND ends_at_unix>?`, channelID, unavailable, testAnchor.Unix()).Scan(&stale); err != nil {
				t.Fatal(err)
			}
			if stale != 0 {
				t.Fatalf("retained %d future entries for unavailable media", stale)
			}
		})
	}
}

func TestRegenerate_PrunesEntriesOlderThanTrailingWindow(t *testing.T) {
	db := openTestDB(t)
	channelID := setupChannel(t, db, 1000, 100) // a single 100s item, looped
	if err := Regenerate(context.Background(), db, channelID, testAnchor); err != nil {
		t.Fatalf("Regenerate (1st): %v", err)
	}

	var minStartsAtUnix int64
	if err := db.QueryRow(`SELECT MIN(starts_at_unix) FROM schedule_entries WHERE channel_id = ?`, channelID).Scan(&minStartsAtUnix); err != nil {
		t.Fatalf("query min starts_at: %v", err)
	}
	if got := time.Unix(minStartsAtUnix, 0).UTC(); !got.Equal(testAnchor.Add(-TrailingWindow)) {
		t.Fatalf("earliest entry starts at %v, want exactly %v (TrailingWindow behind the anchor)", got, testAnchor.Add(-TrailingWindow))
	}

	// Regenerating much later should prune everything before the new
	// trailing edge.
	laterAnchor := testAnchor.Add(24 * time.Hour)
	if err := Regenerate(context.Background(), db, channelID, laterAnchor); err != nil {
		t.Fatalf("Regenerate (2nd): %v", err)
	}
	if err := db.QueryRow(`SELECT MIN(starts_at_unix) FROM schedule_entries WHERE channel_id = ?`, channelID).Scan(&minStartsAtUnix); err != nil {
		t.Fatalf("query min starts_at (after): %v", err)
	}
	newTrailingEdge := laterAnchor.Add(-TrailingWindow)
	if got := time.Unix(minStartsAtUnix, 0).UTC(); got.Before(newTrailingEdge) {
		t.Errorf("earliest entry starts at %v, want nothing before %v", got, newTrailingEdge)
	}
}

func TestRegenerate_EmptyLineup_ClearsAnyStaleEntries(t *testing.T) {
	db := openTestDB(t)
	channelID := setupChannel(t, db, 1000, 100)
	if err := Regenerate(context.Background(), db, channelID, testAnchor); err != nil {
		t.Fatalf("Regenerate (1st, with content): %v", err)
	}
	if n := countScheduleEntries(t, db, channelID); n == 0 {
		t.Fatalf("expected some entries after the first regenerate, got 0")
	}

	if _, err := db.Exec(`DELETE FROM channel_items WHERE channel_id = ?`, channelID); err != nil {
		t.Fatalf("clear channel_items: %v", err)
	}
	if err := Regenerate(context.Background(), db, channelID, testAnchor); err != nil {
		t.Fatalf("Regenerate (2nd, now empty): %v", err)
	}
	if n := countScheduleEntries(t, db, channelID); n != 0 {
		t.Errorf("countScheduleEntries = %d, want 0 once the lineup is empty", n)
	}
}

func TestRegenerateAll_CoversEveryChannel(t *testing.T) {
	db := openTestDB(t)
	a := setupChannel(t, db, 1000, 100)
	b := setupChannel(t, db, 1001, 200)

	n, err := RegenerateAll(context.Background(), db, testAnchor)
	if err != nil {
		t.Fatalf("RegenerateAll: %v", err)
	}
	if n != 2 {
		t.Fatalf("RegenerateAll returned %d, want 2", n)
	}
	if countScheduleEntries(t, db, a) == 0 || countScheduleEntries(t, db, b) == 0 {
		t.Errorf("expected both channels to have schedule_entries after RegenerateAll")
	}
}

func TestManualCommercialRotationAcrossLoopsAndRefresh(t *testing.T) {
	db := openTestDB(t)
	channelID := setupChannel(t, db, 1000, 600)
	if _, err := db.Exec(`UPDATE commercial_settings SET enabled=1,mode='between-only',tv_commercials_per_break=2,movie_commercials_per_break=2`); err != nil {
		t.Fatal(err)
	}
	ads := []int64{}
	for i := 0; i < 5; i++ {
		var id int64
		if err := db.QueryRow(`INSERT INTO media_files(root_kind,path,size_bytes,modified_unix,first_seen_unix,last_seen_unix,duration_seconds,status) VALUES('commercial',?,1,0,0,0,30,'ok') RETURNING id`, fmt.Sprintf("/commercial-%d.mp4", i)).Scan(&id); err != nil {
			t.Fatal(err)
		}
		ads = append(ads, id)
	}
	check := func() {
		rows, err := db.Query(`SELECT media_file_id FROM schedule_entries WHERE channel_id=? AND kind='commercial' ORDER BY starts_at_unix`, channelID)
		if err != nil {
			t.Fatal(err)
		}
		defer rows.Close()
		last := -1
		count := 0
		for rows.Next() {
			var id int64
			if err := rows.Scan(&id); err != nil {
				t.Fatal(err)
			}
			idx := -1
			for i, ad := range ads {
				if ad == id {
					idx = i
				}
			}
			if idx < 0 || (last >= 0 && idx != (last+1)%len(ads)) {
				t.Fatalf("commercial cycle jumped from %d to %d", last, idx)
			}
			last = idx
			count++
		}
		if err := rows.Err(); err != nil {
			t.Fatal(err)
		}
		if count < 10 {
			t.Fatal("fixture did not span multiple full cycles")
		}
	}
	regenerate(t, db, channelID)
	check()
	if err := Regenerate(context.Background(), db, channelID, testAnchor.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	check()
}
