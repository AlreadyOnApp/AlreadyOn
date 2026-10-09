package storage

import (
	"github.com/pressly/goose/v3"
	"strings"
	"testing"
)

func TestRotationMigrationsRebuildOnlyGeneratedPools(t *testing.T) {
	db, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	goose.SetBaseFS(migrationsFS)
	defer goose.SetBaseFS(nil)
	if err := goose.DownTo(db.DB, "migrations", 43); err != nil {
		t.Fatal(err)
	}
	for _, q := range []string{
		`INSERT INTO channels(id,number,name) VALUES(1,201,'Generated TV'),(2,202,'Manual TV'),(3,203,'Movies')`,
		`INSERT INTO channel_filters(channel_id,audience_tier) VALUES(1,'adult'),(3,'adult')`,
		`INSERT INTO media_files(id,root_kind,path,size_bytes,modified_unix,first_seen_unix,last_seen_unix) VALUES(1,'media','/tv',1,1,1,1),(2,'media','/movie',1,1,1,1)`,
		`INSERT INTO identified_items(media_file_id,media_type,title) VALUES(1,'episode','TV'),(2,'movie','Film')`,
		`INSERT INTO channel_items(channel_id,media_file_id,position) VALUES(1,1,0),(2,1,0),(3,2,0)`,
	} {
		if _, err := db.Exec(q); err != nil {
			t.Fatal(err)
		}
	}
	if err := migrate(db.DB); err != nil {
		t.Fatal(err)
	}
	for id, want := range map[int]int{1: 0, 2: 1, 3: 0} {
		var got int
		if err := db.QueryRow(`SELECT COUNT(*) FROM channel_items WHERE channel_id=?`, id).Scan(&got); err != nil {
			t.Fatal(err)
		}
		if got != want {
			t.Fatalf("channel %d has %d entries, want %d", id, got, want)
		}
	}
}

// Test the historical coverage upgrade in isolation before later default replacements.
func TestCoverageMigrationPreservesEditsAndAddsDefault(t *testing.T) {
	db, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	goose.SetBaseFS(migrationsFS)
	defer goose.SetBaseFS(nil)
	if err := goose.DownTo(db.DB, "migrations", 44); err != nil {
		t.Fatal(err)
	}
	for _, q := range []string{
		`INSERT INTO channels(id,number,name) VALUES(1,145,'Golden Age Cinema')`,
		`INSERT INTO channel_filters(channel_id,audience_tier,genres,media_type,era,title_contains) VALUES(1,'adult','["drama","history"]','movie','nostalgia','My phrase')`,
	} {
		if _, err := db.Exec(q); err != nil {
			t.Fatal(err)
		}
	}
	if err := goose.UpTo(db.DB, "migrations", 45); err != nil {
		t.Fatal(err)
	}
	var genres, title, era string
	if err := db.QueryRow(`SELECT genres,title_contains,era FROM channel_filters WHERE channel_id=1`).Scan(&genres, &title, &era); err != nil {
		t.Fatal(err)
	}
	for _, tag := range []string{"drama", "history", "war", "western"} {
		if !strings.Contains(genres, tag) {
			t.Fatalf("missing %s: %s", tag, genres)
		}
	}
	if title != "My phrase" || era != "nostalgia" {
		t.Fatal("administrator edits lost")
	}
	var n int
	if err := db.QueryRow(`SELECT count(*) FROM channels WHERE number=152 AND name='Western and War'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("new default %d %v", n, err)
	}
	if err := goose.UpTo(db.DB, "migrations", 45); err != nil {
		t.Fatal(err)
	}
}
