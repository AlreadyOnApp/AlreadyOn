package storage

import (
	"github.com/pressly/goose/v3"
	"strings"
	"testing"
)

// Default upgrades must not silently broaden a saved custom parental policy.
func TestTeenRatingMigrationPreservesCustomPolicy(t *testing.T) {
	for _, custom := range []bool{false, true} {
		db, err := Open(t.TempDir())
		if err != nil {
			t.Fatal(err)
		}
		goose.SetBaseFS(migrationsFS)
		if err := goose.DownTo(db.DB, "migrations", 62); err != nil {
			t.Fatal(err)
		}
		ratings := `["TV-Y","TV-Y7","TV-G","TV-PG","TV-14","G","PG","PG-13"]`
		if custom {
			ratings = `["TV-G"]`
		}
		if _, err := db.Exec(`UPDATE channel_filter_options SET definition=json_set(definition,'$.ratings',json(?)) WHERE kind='age' AND id='teen'`, ratings); err != nil {
			t.Fatal(err)
		}
		if err := migrate(db.DB); err != nil {
			t.Fatal(err)
		}
		var got string
		if err := db.QueryRow(`SELECT definition FROM channel_filter_options WHERE kind='age' AND id='teen'`).Scan(&got); err != nil {
			t.Fatal(err)
		}
		if strings.Contains(got, "TV-Y7-FV") == custom {
			t.Fatalf("custom=%v: %s", custom, got)
		}
		db.Close()
	}
	goose.SetBaseFS(nil)
}
