package storage

import (
	"github.com/pressly/goose/v3"
	"testing"
	"time"
)

// Legacy native sessions survive upgrade without reviving expired rows or changing browsers.
func TestNativeSessionMigrationDoesNotReviveExpiredSessions(t *testing.T) {
	db, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	goose.SetBaseFS(migrationsFS)
	defer goose.SetBaseFS(nil)
	if err = goose.DownTo(db.DB, "migrations", 82); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec("INSERT INTO users(id,username,password_hash,role) VALUES(999,'fixture','unused','viewer')"); err != nil {
		t.Fatal(err)
	}
	now := time.Now().Unix()
	for i, agent := range []string{"AlreadyOn-Android/TV", "AlreadyOn-Android/Mobile", "browser", "AlreadyOn-Android/TV"} {
		expiry := now + 3600
		if i == 3 {
			expiry = now - 3600
		}
		if _, err = db.Exec("INSERT INTO browser_sessions(token_hash,created_at_unix,expires_at_unix,user_agent,user_id) VALUES(?,?,?,?,999)", agent+string(rune('a'+i)), now, expiry, agent); err != nil {
			t.Fatal(err)
		}
	}
	if err = migrate(db.DB); err != nil {
		t.Fatal(err)
	}
	var durable, expired, browsers int
	db.QueryRow("SELECT count(*) FROM browser_sessions WHERE expires_at_unix=253402300799").Scan(&durable)
	db.QueryRow("SELECT count(*) FROM browser_sessions WHERE expires_at_unix < ?", now).Scan(&expired)
	db.QueryRow("SELECT count(*) FROM browser_sessions WHERE user_agent='browser' AND expires_at_unix=?", now+3600).Scan(&browsers)
	if durable != 2 || expired != 1 || browsers != 1 {
		t.Fatalf("durable=%d expired=%d browsers=%d", durable, expired, browsers)
	}
}
