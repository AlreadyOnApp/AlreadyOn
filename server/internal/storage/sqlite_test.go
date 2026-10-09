package storage

import (
	"strings"
	"testing"
	"time"
)

func TestOpen_CleanDatabase_CreatesSchema(t *testing.T) {
	db, err := Open(t.TempDir())
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer db.Close()

	for _, table := range []string{"media_files", "identified_items"} {
		var name string
		err := db.QueryRow(`SELECT name FROM sqlite_master WHERE type = 'table' AND name = ?`, table).Scan(&name)
		if err != nil {
			t.Fatalf("expected table %q to exist: %v", table, err)
		}
	}
}

func TestOpen_AlreadyMigratedDatabase_IsIdempotent(t *testing.T) {
	dir := t.TempDir()

	first, err := Open(dir)
	if err != nil {
		t.Fatalf("first Open: %v", err)
	}
	first.Close()

	second, err := Open(dir)
	if err != nil {
		t.Fatalf("second Open on already-migrated database: %v", err)
	}
	defer second.Close()

	var name string
	if err := second.QueryRow(`SELECT name FROM sqlite_master WHERE type = 'table' AND name = 'media_files'`).Scan(&name); err != nil {
		t.Fatalf("expected media_files to still exist after reopening: %v", err)
	}
}

// A cookie name belongs to the database identity: reopening must preserve it,
// while an independent server database must not collide with it.
func TestOpen_AssignsStableUniqueSessionCookieNames(t *testing.T) {
	readName := func(dir string) string {
		db, err := Open(dir)
		if err != nil {
			t.Fatalf("Open: %v", err)
		}
		defer db.Close()
		var name string
		if err := db.QueryRow(`SELECT session_cookie_name FROM server_settings WHERE id = 1`).Scan(&name); err != nil {
			t.Fatalf("read cookie name: %v", err)
		}
		if !strings.HasPrefix(name, "alreadyon_") || len(name) != len("alreadyon_")+32 {
			t.Fatalf("cookie name %q does not have the expected prefix and random suffix", name)
		}
		return name
	}

	firstDir := t.TempDir()
	first := readName(firstDir)
	if reopened := readName(firstDir); reopened != first {
		t.Fatalf("cookie name changed after reopen: %q to %q", first, reopened)
	}
	if second := readName(t.TempDir()); second == first {
		t.Fatalf("separate databases received the same cookie name %q", first)
	}
}

func TestSchema_MediaFilePathIsUnique(t *testing.T) {
	db, err := Open(t.TempDir())
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer db.Close()

	now := time.Now().Unix()
	insert := `INSERT INTO media_files (root_kind, path, size_bytes, modified_unix, first_seen_unix, last_seen_unix)
		VALUES (?, ?, ?, ?, ?, ?)`

	if _, err := db.Exec(insert, "media", "/media/movies/example.mkv", 1024, now, now, now); err != nil {
		t.Fatalf("first insert: %v", err)
	}
	if _, err := db.Exec(insert, "media", "/media/movies/example.mkv", 2048, now, now, now); err == nil {
		t.Fatal("expected duplicate path insert to fail the UNIQUE constraint, but it succeeded")
	}
}

func TestSchema_IdentifiedItemCascadesOnMediaFileDelete(t *testing.T) {
	db, err := Open(t.TempDir())
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer db.Close()

	now := time.Now().Unix()
	res, err := db.Exec(`INSERT INTO media_files (root_kind, path, size_bytes, modified_unix, first_seen_unix, last_seen_unix)
		VALUES (?, ?, ?, ?, ?, ?)`, "media", "/media/tv/show/s01e01.mkv", 4096, now, now, now)
	if err != nil {
		t.Fatalf("insert media_files: %v", err)
	}
	mediaFileID, err := res.LastInsertId()
	if err != nil {
		t.Fatalf("LastInsertId: %v", err)
	}

	if _, err := db.Exec(`INSERT INTO identified_items (media_file_id, media_type, title, series_title, season_number, episode_number)
		VALUES (?, 'episode', ?, ?, ?, ?)`, mediaFileID, "Pilot", "Example Show", 1, 1); err != nil {
		t.Fatalf("insert identified_items: %v", err)
	}

	if _, err := db.Exec(`DELETE FROM media_files WHERE id = ?`, mediaFileID); err != nil {
		t.Fatalf("delete media_files: %v", err)
	}

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM identified_items WHERE media_file_id = ?`, mediaFileID).Scan(&count); err != nil {
		t.Fatalf("count identified_items: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected ON DELETE CASCADE to remove the identified_items row, but %d remain", count)
	}
}

func TestSchema_MediaTypeCheckConstraintRejectsInvalidValue(t *testing.T) {
	db, err := Open(t.TempDir())
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer db.Close()

	now := time.Now().Unix()
	res, err := db.Exec(`INSERT INTO media_files (root_kind, path, size_bytes, modified_unix, first_seen_unix, last_seen_unix)
		VALUES (?, ?, ?, ?, ?, ?)`, "media", "/media/movies/other.mkv", 1024, now, now, now)
	if err != nil {
		t.Fatalf("insert media_files: %v", err)
	}
	mediaFileID, _ := res.LastInsertId()

	_, err = db.Exec(`INSERT INTO identified_items (media_file_id, media_type, title) VALUES (?, 'cartoon', ?)`, mediaFileID, "Nope")
	if err == nil {
		t.Fatal("expected an invalid media_type to violate the CHECK constraint, but it succeeded")
	}
}
