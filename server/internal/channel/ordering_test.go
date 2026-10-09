package channel

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"
)

// insertTestMediaFile inserts a minimal, valid media_files row so
// channel_items' foreign key is satisfiable in tests — these tests
// exercise lineup ordering, not the scanner that would normally create
// these rows (see library.Scanner for that).
func insertTestMediaFile(t *testing.T, db *sql.DB, path string) int64 {
	t.Helper()
	now := time.Now().Unix()
	res, err := db.Exec(`
		INSERT INTO media_files (root_kind, path, size_bytes, modified_unix, first_seen_unix, last_seen_unix)
		VALUES ('media', ?, 1024, ?, ?, ?)`, path, now, now, now)
	if err != nil {
		t.Fatalf("insert media_files: %v", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		t.Fatalf("LastInsertId: %v", err)
	}
	return id
}

func TestSortEpisodes_OrdersBySeasonThenEpisode(t *testing.T) {
	in := []EpisodeInfo{
		{MediaFileID: 1, Season: 2, Episode: 1},
		{MediaFileID: 2, Season: 1, Episode: 2},
		{MediaFileID: 3, Season: 1, Episode: 1},
		{MediaFileID: 4, Season: 2, Episode: 2},
	}
	got := SortEpisodes(in)

	want := []int64{3, 2, 1, 4} // S01E01, S01E02, S02E01, S02E02
	for i, id := range want {
		if got[i].MediaFileID != id {
			t.Errorf("got[%d].MediaFileID = %d, want %d (full result: %+v)", i, got[i].MediaFileID, id, got)
		}
	}
}

func TestSortEpisodes_StableForTies(t *testing.T) {
	// Two items both claiming S01E01 — the sort must not reorder them
	// relative to each other (or to anything else), so the result is
	// reproducible regardless of what order the caller happened to
	// gather them in (e.g. a DB query's own row order).
	in := []EpisodeInfo{
		{MediaFileID: 10, Season: 1, Episode: 1},
		{MediaFileID: 20, Season: 1, Episode: 1},
		{MediaFileID: 30, Season: 1, Episode: 1},
	}
	got := SortEpisodes(in)
	for i, id := range []int64{10, 20, 30} {
		if got[i].MediaFileID != id {
			t.Errorf("got[%d].MediaFileID = %d, want %d — stable sort broke tie order", i, got[i].MediaFileID, id)
		}
	}
}

func TestSortEpisodes_DoesNotMutateInput(t *testing.T) {
	in := []EpisodeInfo{{MediaFileID: 1, Season: 2, Episode: 1}, {MediaFileID: 2, Season: 1, Episode: 1}}
	_ = SortEpisodes(in)
	if in[0].MediaFileID != 1 || in[1].MediaFileID != 2 {
		t.Errorf("SortEpisodes mutated its input: %+v", in)
	}
}

func TestAppendItems_AssignsSequentialPositions(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()
	c, err := Create(ctx, db, Draft{Number: 1000, Name: "Test Channel"})
	if err != nil {
		t.Fatalf("Create channel: %v", err)
	}
	m1 := insertTestMediaFile(t, db, "/media/a.mkv")
	m2 := insertTestMediaFile(t, db, "/media/b.mkv")
	m3 := insertTestMediaFile(t, db, "/media/c.mkv")

	items, err := AppendItems(ctx, db, c.ID, []int64{m1, m2, m3})
	if err != nil {
		t.Fatalf("AppendItems: %v", err)
	}
	if len(items) != 3 {
		t.Fatalf("len(items) = %d, want 3", len(items))
	}
	for i, want := range []int64{m1, m2, m3} {
		if items[i].MediaFileID != want || items[i].Position != i {
			t.Errorf("items[%d] = %+v, want MediaFileID=%d Position=%d", i, items[i], want, i)
		}
	}

	listed, err := ListItems(ctx, db, c.ID)
	if err != nil {
		t.Fatalf("ListItems: %v", err)
	}
	if len(listed) != 3 {
		t.Fatalf("len(listed) = %d, want 3", len(listed))
	}
}

func TestAppendItems_SecondCallContinuesFromLastPosition(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()
	c, err := Create(ctx, db, Draft{Number: 1000, Name: "Test Channel"})
	if err != nil {
		t.Fatalf("Create channel: %v", err)
	}
	m1 := insertTestMediaFile(t, db, "/media/a.mkv")
	m2 := insertTestMediaFile(t, db, "/media/b.mkv")

	if _, err := AppendItems(ctx, db, c.ID, []int64{m1}); err != nil {
		t.Fatalf("first AppendItems: %v", err)
	}
	second, err := AppendItems(ctx, db, c.ID, []int64{m2})
	if err != nil {
		t.Fatalf("second AppendItems: %v", err)
	}
	if second[0].Position != 1 {
		t.Errorf("second batch's position = %d, want 1", second[0].Position)
	}
}

func TestAppendItems_UnknownChannel_ReturnsErrNotFound(t *testing.T) {
	db := openTestDB(t)
	m1 := insertTestMediaFile(t, db, "/media/a.mkv")
	_, err := AppendItems(context.Background(), db, 999, []int64{m1})
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
}

func TestAppendItems_UnknownMediaFile_ReturnsErrMediaFileNotFound(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()
	c, err := Create(ctx, db, Draft{Number: 1000, Name: "Test Channel"})
	if err != nil {
		t.Fatalf("Create channel: %v", err)
	}

	_, err = AppendItems(ctx, db, c.ID, []int64{999})
	if !errors.Is(err, ErrMediaFileNotFound) {
		t.Errorf("err = %v, want ErrMediaFileNotFound", err)
	}
}

func TestAppendItems_DuplicateMediaFile_ReturnsErrDuplicateItem(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()
	c, err := Create(ctx, db, Draft{Number: 1000, Name: "Test Channel"})
	if err != nil {
		t.Fatalf("Create channel: %v", err)
	}
	m1 := insertTestMediaFile(t, db, "/media/a.mkv")

	if _, err := AppendItems(ctx, db, c.ID, []int64{m1}); err != nil {
		t.Fatalf("first AppendItems: %v", err)
	}
	_, err = AppendItems(ctx, db, c.ID, []int64{m1})
	if !errors.Is(err, ErrDuplicateItem) {
		t.Errorf("err = %v, want ErrDuplicateItem", err)
	}
}

func TestAppendItems_PartialFailureAppendsNothing(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()
	c, err := Create(ctx, db, Draft{Number: 1000, Name: "Test Channel"})
	if err != nil {
		t.Fatalf("Create channel: %v", err)
	}
	m1 := insertTestMediaFile(t, db, "/media/a.mkv")

	// m1 is valid but 999 doesn't exist — the whole batch should fail
	// atomically, not append m1 and then error on 999.
	_, err = AppendItems(ctx, db, c.ID, []int64{m1, 999})
	if !errors.Is(err, ErrMediaFileNotFound) {
		t.Fatalf("err = %v, want ErrMediaFileNotFound", err)
	}

	listed, err := ListItems(ctx, db, c.ID)
	if err != nil {
		t.Fatalf("ListItems: %v", err)
	}
	if len(listed) != 0 {
		t.Errorf("expected no items appended after a partial failure, got %d", len(listed))
	}
}

func TestListItems_EmptyLineup(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()
	c, err := Create(ctx, db, Draft{Number: 1000, Name: "Test Channel"})
	if err != nil {
		t.Fatalf("Create channel: %v", err)
	}

	items, err := ListItems(ctx, db, c.ID)
	if err != nil {
		t.Fatalf("ListItems: %v", err)
	}
	if len(items) != 0 {
		t.Errorf("len(items) = %d, want 0", len(items))
	}
}

func TestRemoveItem(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()
	c, err := Create(ctx, db, Draft{Number: 1000, Name: "Test Channel"})
	if err != nil {
		t.Fatalf("Create channel: %v", err)
	}
	m1 := insertTestMediaFile(t, db, "/media/a.mkv")
	m2 := insertTestMediaFile(t, db, "/media/b.mkv")
	items, err := AppendItems(ctx, db, c.ID, []int64{m1, m2})
	if err != nil {
		t.Fatalf("AppendItems: %v", err)
	}

	if err := RemoveItem(ctx, db, c.ID, items[0].ID); err != nil {
		t.Fatalf("RemoveItem: %v", err)
	}

	remaining, err := ListItems(ctx, db, c.ID)
	if err != nil {
		t.Fatalf("ListItems: %v", err)
	}
	if len(remaining) != 1 || remaining[0].MediaFileID != m2 {
		t.Errorf("remaining = %+v, want just m2 (%d)", remaining, m2)
	}
}

func TestRemoveItem_LeavesGapButOrderingStaysCorrect(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()
	c, err := Create(ctx, db, Draft{Number: 1000, Name: "Test Channel"})
	if err != nil {
		t.Fatalf("Create channel: %v", err)
	}
	m1 := insertTestMediaFile(t, db, "/media/a.mkv")
	m2 := insertTestMediaFile(t, db, "/media/b.mkv")
	m3 := insertTestMediaFile(t, db, "/media/c.mkv")
	items, err := AppendItems(ctx, db, c.ID, []int64{m1, m2, m3})
	if err != nil {
		t.Fatalf("AppendItems: %v", err)
	}

	if err := RemoveItem(ctx, db, c.ID, items[1].ID); err != nil { // remove m2
		t.Fatalf("RemoveItem: %v", err)
	}

	remaining, err := ListItems(ctx, db, c.ID)
	if err != nil {
		t.Fatalf("ListItems: %v", err)
	}
	if len(remaining) != 2 || remaining[0].MediaFileID != m1 || remaining[1].MediaFileID != m3 {
		t.Errorf("remaining = %+v, want [m1, m3] in order despite the position gap", remaining)
	}
}

func TestRemoveItem_NotFound(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()
	c, err := Create(ctx, db, Draft{Number: 1000, Name: "Test Channel"})
	if err != nil {
		t.Fatalf("Create channel: %v", err)
	}

	if err := RemoveItem(ctx, db, c.ID, 999); !errors.Is(err, ErrNotFound) {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
}

func TestClearItems(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()
	c, err := Create(ctx, db, Draft{Number: 1000, Name: "Test Channel"})
	if err != nil {
		t.Fatalf("Create channel: %v", err)
	}
	m1 := insertTestMediaFile(t, db, "/media/a.mkv")
	if _, err := AppendItems(ctx, db, c.ID, []int64{m1}); err != nil {
		t.Fatalf("AppendItems: %v", err)
	}

	if err := ClearItems(ctx, db, c.ID); err != nil {
		t.Fatalf("ClearItems: %v", err)
	}

	items, err := ListItems(ctx, db, c.ID)
	if err != nil {
		t.Fatalf("ListItems: %v", err)
	}
	if len(items) != 0 {
		t.Errorf("len(items) = %d, want 0 after ClearItems", len(items))
	}
}

func TestDeleteChannel_CascadesToItems(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()
	c, err := Create(ctx, db, Draft{Number: 1000, Name: "Test Channel"})
	if err != nil {
		t.Fatalf("Create channel: %v", err)
	}
	m1 := insertTestMediaFile(t, db, "/media/a.mkv")
	if _, err := AppendItems(ctx, db, c.ID, []int64{m1}); err != nil {
		t.Fatalf("AppendItems: %v", err)
	}

	if err := Delete(ctx, db, c.ID); err != nil {
		t.Fatalf("Delete channel: %v", err)
	}

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM channel_items WHERE channel_id = ?`, c.ID).Scan(&count); err != nil {
		t.Fatalf("count channel_items: %v", err)
	}
	if count != 0 {
		t.Errorf("expected deleting a channel to cascade-delete its items, got %d remaining", count)
	}
}
