// Package storage manages the SQLite connection and versioned schema
// migrations. See docs/PROJECT_BRIEF.md section 5 and docs/TOOLCHAIN.md
// for the driver/migration-tool choice (modernc.org/sqlite + goose).
package storage

import (
	"database/sql"
	"embed"
	"fmt"
	"os"
	"path/filepath"

	"github.com/pressly/goose/v3"
	_ "modernc.org/sqlite"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// DB wraps the SQLite connection used by all other internal packages.
type DB struct {
	*sql.DB
}

// Open creates dataDir if needed, opens (or creates) the SQLite database
// file inside it, and applies any pending migrations. Migrations only
// ever run forward here — rolling back an applied migration is a manual
// developer action (`goose down`), never something the running server
// does on its own.
func Open(dataDir string) (*DB, error) {
	if err := os.MkdirAll(dataDir, 0o700); err != nil {
		return nil, fmt.Errorf("storage: create data dir: %w", err)
	}
	// Database/WAL files can contain authentication and TLS private keys. Protect
	// existing directories too; this never changes configured media permissions.
	if err := os.Chmod(dataDir, 0o700); err != nil {
		return nil, fmt.Errorf("storage: protect data dir: %w", err)
	}

	dbPath := filepath.Join(dataDir, "alreadyon.db")
	// _pragma params apply per-connection; foreign_keys defaults to off in
	// SQLite and busy_timeout keeps concurrent access from failing fast
	// with SQLITE_BUSY while WAL checkpoints or another writer holds the
	// database briefly.
	dsn := fmt.Sprintf("file:%s?_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)", dbPath)

	conn, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("storage: open %s: %w", dbPath, err)
	}

	// This is an embedded single-file database, not a connection-pooled
	// server; SQLite only allows one writer at a time regardless, so a
	// single connection avoids SQLITE_BUSY errors from this process
	// contending with itself.
	conn.SetMaxOpenConns(1)
	if err := conn.Ping(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("storage: connect: %w", err)
	}
	if err := os.Chmod(dbPath, 0o600); err != nil {
		conn.Close()
		return nil, fmt.Errorf("storage: protect database: %w", err)
	}

	if err := migrate(conn); err != nil {
		conn.Close()
		return nil, err
	}

	return &DB{conn}, nil
}

func migrate(conn *sql.DB) error {
	goose.SetBaseFS(migrationsFS)
	defer goose.SetBaseFS(nil)

	if err := goose.SetDialect("sqlite3"); err != nil {
		return fmt.Errorf("storage: set migration dialect: %w", err)
	}
	if err := goose.Up(conn, "migrations"); err != nil {
		return fmt.Errorf("storage: apply migrations: %w", err)
	}
	return nil
}
