package storage

import (
	"testing"

	"github.com/pressly/goose/v3"
)

// Upgrading a single-account server keeps the account as owner 1 with its
// sessions, keys and exactly the profiles the household already had; a fresh
// install gains the Teen default.
func TestUserAccountsMigration(t *testing.T) {
	t.Cleanup(func() { goose.SetBaseFS(nil) })
	for _, upgrade := range []bool{true, false} {
		db, err := Open(t.TempDir())
		if err != nil {
			t.Fatal(err)
		}
		goose.SetBaseFS(migrationsFS)
		if err := goose.DownTo(db.DB, "migrations", 77); err != nil {
			t.Fatal(err)
		}
		// Down keeps a seeded Teen profile; a real pre-0078 database has none.
		if _, err := db.Exec(`DELETE FROM profiles WHERE name = 'Teen'`); err != nil {
			t.Fatal(err)
		}
		if upgrade {
			for _, q := range []string{
				`INSERT INTO server_account (id, username, password_hash, created_at_unix) VALUES (1, 'mike', 'hash', 42)`,
				`INSERT INTO browser_sessions (token_hash, created_at_unix, expires_at_unix) VALUES ('t', 1, 9999999999)`,
				`UPDATE profiles SET pin_hash = 'pin' WHERE name = 'Kids'`,
			} {
				if _, err := db.Exec(q); err != nil {
					t.Fatal(q, err)
				}
			}
		}
		var before [][2]any
		rows, err := db.Query(`SELECT id, name FROM profiles ORDER BY id`)
		if err != nil {
			t.Fatal(err)
		}
		for rows.Next() {
			var id int64
			var name string
			rows.Scan(&id, &name)
			before = append(before, [2]any{id, name})
		}
		rows.Close()
		if err := migrate(db.DB); err != nil {
			t.Fatal(err)
		}

		var users int
		db.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&users)
		var teen int
		db.QueryRow(`SELECT COUNT(*) FROM profiles WHERE name = 'Teen' AND user_id = 1`).Scan(&teen)
		var total int
		db.QueryRow(`SELECT COUNT(*) FROM profiles WHERE user_id = 1`).Scan(&total)
		for _, p := range before {
			var name string
			if err := db.QueryRow(`SELECT name FROM profiles WHERE id = ? AND user_id = 1`, p[0]).Scan(&name); err != nil || name != p[1] {
				t.Errorf("upgrade=%v: profile %v lost or renamed: %q %v", upgrade, p, name, err)
			}
		}
		if upgrade {
			var name, role, hash string
			var created int64
			if err := db.QueryRow(`SELECT username, role, password_hash, created_at_unix FROM users WHERE id = 1`).Scan(&name, &role, &hash, &created); err != nil {
				t.Fatal(err)
			}
			if users != 1 || name != "mike" || role != "owner" || hash != "hash" || created != 42 {
				t.Errorf("migrated account = %s %s %s %d (users=%d)", name, role, hash, created, users)
			}
			var sessionUser int64
			db.QueryRow(`SELECT user_id FROM browser_sessions`).Scan(&sessionUser)
			if sessionUser != 1 {
				t.Errorf("existing session belongs to user %d, want 1", sessionUser)
			}
			var pin string
			db.QueryRow(`SELECT pin_hash FROM profiles WHERE name = 'Kids'`).Scan(&pin)
			if pin != "pin" {
				t.Error("profile settings were not kept")
			}
			if teen != 0 || total != len(before) {
				t.Errorf("an upgrade must keep exactly the existing profiles: teen=%d total=%d before=%d", teen, total, len(before))
			}
		} else if users != 0 || teen != 1 || total != len(before)+1 {
			t.Errorf("fresh install: users=%d teen=%d total=%d", users, teen, total)
		}
		var exists int
		db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE name = 'server_account'`).Scan(&exists)
		if exists != 0 {
			t.Error("server_account must be dropped")
		}
		if _, err := db.Exec(`INSERT INTO profiles (user_id, name, role, audience_tier) VALUES (2, 'Kids', 'viewer', 'young_child')`); err != nil {
			t.Errorf("another user cannot reuse a profile name: %v", err)
		}
		if _, err := db.Exec(`INSERT INTO profiles (user_id, name, role, audience_tier) VALUES (1, 'kids', 'viewer', 'young_child')`); err == nil {
			t.Error("profile names must stay unique within one user")
		}
		db.Close()
	}
}
