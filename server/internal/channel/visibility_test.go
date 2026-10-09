package channel

import (
	"context"
	"database/sql"
	"testing"
	"time"
)

// insertClassifiedContent inserts a media_files + identified_items row
// and attaches it to channelID's lineup, for series-count/visibility
// tests. seriesTitle == "" makes it its own one-item "movie" series.
func insertClassifiedContent(t *testing.T, db *sql.DB, channelID int64, path, seriesTitle string, position int) {
	t.Helper()
	ctx := context.Background()
	now := time.Now().Unix()
	mediaType := "movie"
	var st sql.NullString
	if seriesTitle != "" {
		mediaType = "episode"
		st = sql.NullString{String: seriesTitle, Valid: true}
	}

	var mediaFileID int64
	if err := db.QueryRowContext(ctx, `
		INSERT INTO media_files (root_kind, path, size_bytes, modified_unix, first_seen_unix, last_seen_unix, duration_seconds, status)
		VALUES ('media', ?, 1024, ?, ?, ?, 600, 'ok') RETURNING id`,
		path, now, now, now).Scan(&mediaFileID); err != nil {
		t.Fatalf("insert media_files: %v", err)
	}
	if _, err := db.ExecContext(ctx, `
		INSERT INTO identified_items (media_file_id, media_type, title, series_title) VALUES (?, ?, ?, ?)`,
		mediaFileID, mediaType, path, st); err != nil {
		t.Fatalf("insert identified_items: %v", err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO channel_items (channel_id, media_file_id, position) VALUES (?, ?, ?)`,
		channelID, mediaFileID, position); err != nil {
		t.Fatalf("insert channel_items: %v", err)
	}
}

func TestSetHidden_BulkUpdatesMultipleChannels(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()
	a, _ := Create(ctx, db, Draft{Number: 1000, Name: "A"})
	b, _ := Create(ctx, db, Draft{Number: 1001, Name: "B"})
	c, _ := Create(ctx, db, Draft{Number: 1002, Name: "C"})

	if err := SetHidden(ctx, db, []int64{a.ID, b.ID}, true); err != nil {
		t.Fatalf("SetHidden: %v", err)
	}

	gotA, _ := Get(ctx, db, a.ID)
	gotB, _ := Get(ctx, db, b.ID)
	gotC, _ := Get(ctx, db, c.ID)
	if !gotA.Hidden || !gotB.Hidden {
		t.Errorf("A/B not hidden: A=%v B=%v", gotA.Hidden, gotB.Hidden)
	}
	if gotC.Hidden {
		t.Error("C should be untouched (not in the ids list)")
	}
}

func TestSetHidden_UnknownID_IsIgnoredNotAnError(t *testing.T) {
	db := openTestDB(t)
	if err := SetHidden(context.Background(), db, []int64{99999}, true); err != nil {
		t.Errorf("SetHidden with an unknown id returned an error: %v", err)
	}
}

func TestSeriesCounts_CountsDistinctShowsNotEpisodes(t *testing.T) {
	db := openTestDB(t)
	ch, _ := Create(context.Background(), db, Draft{Number: 1000, Name: "Test"})
	insertClassifiedContent(t, db, ch.ID, "/a-e1.mkv", "Show A", 0)
	insertClassifiedContent(t, db, ch.ID, "/a-e2.mkv", "Show A", 1) // same show, second episode
	insertClassifiedContent(t, db, ch.ID, "/movie.mkv", "", 2)      // a distinct movie

	counts, err := SeriesCounts(context.Background(), db)
	if err != nil {
		t.Fatalf("SeriesCounts: %v", err)
	}
	if counts[ch.ID] != 2 {
		t.Errorf("counts[%d] = %d, want 2 (Show A once + the movie)", ch.ID, counts[ch.ID])
	}
}

func TestSeriesCounts_ChannelWithNoContent_IsAbsentFromResult(t *testing.T) {
	db := openTestDB(t)
	ch, _ := Create(context.Background(), db, Draft{Number: 1000, Name: "Empty"})

	counts, err := SeriesCounts(context.Background(), db)
	if err != nil {
		t.Fatalf("SeriesCounts: %v", err)
	}
	if _, ok := counts[ch.ID]; ok {
		t.Errorf("counts contains an entry for an empty channel: %v", counts)
	}
}

func TestSeriesCount_MatchesBulkSeriesCounts(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()
	ch, _ := Create(ctx, db, Draft{Number: 1000, Name: "Test"})
	insertClassifiedContent(t, db, ch.ID, "/a-e1.mkv", "Show A", 0)
	insertClassifiedContent(t, db, ch.ID, "/a-e2.mkv", "Show A", 1)
	insertClassifiedContent(t, db, ch.ID, "/movie.mkv", "", 2)

	got, err := SeriesCount(ctx, db, ch.ID)
	if err != nil {
		t.Fatalf("SeriesCount: %v", err)
	}
	if got != 2 {
		t.Errorf("SeriesCount(%d) = %d, want 2 (Show A once + the movie)", ch.ID, got)
	}
}

func TestSeriesCount_ChannelWithNoContent_IsZero(t *testing.T) {
	db := openTestDB(t)
	ch, _ := Create(context.Background(), db, Draft{Number: 1000, Name: "Empty"})

	got, err := SeriesCount(context.Background(), db, ch.ID)
	if err != nil {
		t.Fatalf("SeriesCount: %v", err)
	}
	if got != 0 {
		t.Errorf("SeriesCount(%d) = %d, want 0", ch.ID, got)
	}
}

func TestAutoVisibility_HidesChannelsWithNoContentOnly(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()
	empty, _ := Create(ctx, db, Draft{Number: 1000, Name: "Empty"})
	oneShow, _ := Create(ctx, db, Draft{Number: 1001, Name: "OneShow"})
	insertClassifiedContent(t, db, oneShow.ID, "/a-e1.mkv", "Show A", 0)
	insertClassifiedContent(t, db, oneShow.ID, "/a-e2.mkv", "Show A", 1)
	twoShows, _ := Create(ctx, db, Draft{Number: 1002, Name: "TwoShows"})
	insertClassifiedContent(t, db, twoShows.ID, "/a-e1.mkv2", "Show A", 0)
	insertClassifiedContent(t, db, twoShows.ID, "/b-e1.mkv", "Show B", 1)

	hidden, shown, err := ApplyAutoVisibility(ctx, db)
	if err != nil {
		t.Fatalf("ApplyAutoVisibility: %v", err)
	}
	if hidden != 1 {
		t.Errorf("hidden = %d, want 1 (only the genuinely empty channel)", hidden)
	}
	if shown != 0 {
		t.Errorf("shown = %d, want 0 (nothing was previously hidden)", shown)
	}

	gotEmpty, _ := Get(ctx, db, empty.ID)
	gotOne, _ := Get(ctx, db, oneShow.ID)
	gotTwo, _ := Get(ctx, db, twoShows.ID)
	if !gotEmpty.Hidden {
		t.Error("empty should be hidden")
	}
	if gotOne.Hidden {
		t.Error("oneShow has real content and should stay visible — \"Hide Empty\" isn't the same bar as lineup's own population-time default")
	}
	if gotTwo.Hidden {
		t.Error("twoShows should stay visible")
	}
}

func TestAutoVisibility_RevealsPreviouslyHiddenChannelThatNowHasContent(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()
	ch, _ := Create(ctx, db, Draft{Number: 1000, Name: "WasEmpty"})
	if err := SetHidden(ctx, db, []int64{ch.ID}, true); err != nil {
		t.Fatalf("SetHidden: %v", err)
	}
	insertClassifiedContent(t, db, ch.ID, "/movie.mkv", "", 0)

	hidden, shown, err := ApplyAutoVisibility(ctx, db)
	if err != nil {
		t.Fatalf("ApplyAutoVisibility: %v", err)
	}
	if hidden != 0 {
		t.Errorf("hidden = %d, want 0", hidden)
	}
	if shown != 1 {
		t.Errorf("shown = %d, want 1 (the channel that gained content)", shown)
	}

	got, _ := Get(ctx, db, ch.ID)
	if got.Hidden {
		t.Error("channel now has content and should have been revealed")
	}
}

func TestAutoVisibility_AlreadyHiddenChannel_IsNotRecounted(t *testing.T) {
	// A channel the admin already hid (e.g. manually) shouldn't be
	// double-counted in HideEmpty's return value even though it's still
	// below the threshold.
	db := openTestDB(t)
	ctx := context.Background()
	ch, _ := Create(ctx, db, Draft{Number: 1000, Name: "Empty"})
	if err := SetHidden(ctx, db, []int64{ch.ID}, true); err != nil {
		t.Fatalf("SetHidden: %v", err)
	}

	hidden, shown, err := ApplyAutoVisibility(ctx, db)
	if err != nil {
		t.Fatalf("ApplyAutoVisibility: %v", err)
	}
	if hidden != 0 {
		t.Errorf("hidden = %d, want 0 (already hidden)", hidden)
	}
	if shown != 0 {
		t.Errorf("shown = %d, want 0 (still empty, nothing to reveal)", shown)
	}
}

func TestSetVisibility_AutoShowHide(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()
	empty, _ := Create(ctx, db, Draft{Number: 1000, Name: "Empty"})
	full, _ := Create(ctx, db, Draft{Number: 1001, Name: "Full"})
	insertClassifiedContent(t, db, full.ID, "/movie.mkv", "", 0)
	if got, _ := Get(ctx, db, empty.ID); got.Visibility != VisibilityAuto {
		t.Fatalf("a new channel is %q, want auto", got.Visibility)
	}
	state := func(id int64) (string, bool) {
		c, _ := Get(ctx, db, id)
		return c.Visibility, c.Hidden
	}

	// Auto: hidden while empty, shown once it has content.
	if err := SetVisibility(ctx, db, []int64{empty.ID, full.ID}, VisibilityAuto); err != nil {
		t.Fatal(err)
	}
	if _, h := state(empty.ID); !h {
		t.Error("an empty auto channel should be hidden")
	}
	if _, h := state(full.ID); h {
		t.Error("an auto channel with content should be shown")
	}
	// Show keeps an empty channel; Hide hides one with content, and both survive a refresh.
	SetVisibility(ctx, db, []int64{empty.ID}, VisibilityShow)
	SetVisibility(ctx, db, []int64{full.ID}, VisibilityHide)
	if _, h := state(empty.ID); h {
		t.Error("show must reveal an empty channel")
	}
	if _, h := state(full.ID); !h {
		t.Error("hide must hide a channel with content")
	}
	if _, _, err := ApplyAutoVisibility(ctx, db); err != nil {
		t.Fatal(err)
	}
	if _, h := state(empty.ID); h {
		t.Error("a refresh must not hide a Show channel")
	}
	if _, h := state(full.ID); !h {
		t.Error("a refresh must not reveal a Hide channel")
	}
	// Back to auto follows content again.
	SetVisibility(ctx, db, []int64{empty.ID, full.ID}, VisibilityAuto)
	if _, h := state(empty.ID); !h {
		t.Error("auto should hide the empty channel again")
	}
	if _, h := state(full.ID); h {
		t.Error("auto should show the channel with content again")
	}
	if err := SetVisibility(ctx, db, []int64{full.ID}, "sometimes"); err != ErrInvalidVisibility {
		t.Errorf("invalid mode: %v", err)
	}
	if err := SetVisibility(ctx, db, []int64{99999}, VisibilityHide); err != nil {
		t.Errorf("an unknown id must be ignored: %v", err)
	}
}
