package storage

import (
	"testing"

	"github.com/pressly/goose/v3"
)

// Migration 0033 raises stored leases that the new application minimum
// (20s, above the browser heartbeat) would otherwise reject on every tune.
func TestLeaseMinimumMigration(t *testing.T) {
	db, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	goose.SetBaseFS(migrationsFS)
	defer goose.SetBaseFS(nil)
	if err := goose.DownTo(db.DB, "migrations", 32); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE playback_settings SET lease_seconds=10`); err != nil {
		t.Fatal(err)
	}
	if err := migrate(db.DB); err != nil {
		t.Fatal(err)
	}
	var lease int
	if err := db.QueryRow(`SELECT lease_seconds FROM playback_settings WHERE id=1`).Scan(&lease); err != nil || lease != 20 {
		t.Fatalf("lease %d %v", lease, err)
	}
}
