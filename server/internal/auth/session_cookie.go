package auth

import (
	"context"
	"database/sql"
	"encoding/hex"
	"fmt"
	"strings"
)

// SessionCookiePrefix identifies AlreadyOn session cookies without making
// separate server instances on the same browser hostname share a cookie.
const SessionCookiePrefix = "alreadyon_"

// SessionCookieName returns the stable random cookie name created for this
// database by its schema migration. It is intentionally stored with the
// database so restarts and upgrades do not sign every browser out.
func SessionCookieName(ctx context.Context, db *sql.DB) (string, error) {
	var name string
	if err := db.QueryRowContext(ctx, `SELECT session_cookie_name FROM server_settings WHERE id = 1`).Scan(&name); err != nil {
		return "", fmt.Errorf("auth: read session cookie name: %w", err)
	}
	suffix := strings.TrimPrefix(name, SessionCookiePrefix)
	if len(suffix) != 32 || len(name) != len(SessionCookiePrefix)+32 {
		return "", fmt.Errorf("auth: invalid session cookie name")
	}
	if _, err := hex.DecodeString(suffix); err != nil {
		return "", fmt.Errorf("auth: invalid session cookie name")
	}
	return name, nil
}
