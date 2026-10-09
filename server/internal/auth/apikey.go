package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
)

// APIKey is a generated credential for authenticating directly to the
// server's HTTP API — e.g. a script, or a future TV client that isn't
// going through the local admin account/session system above (see
// Account/Session's own "device credentials for future TV clients
// follow the same model" note). Production management routes verify
// these through RequireAuth; playback exchanges them for scoped tokens.
//
// Every key belongs to the owner who created it and acts as that user. Only
// owners hold keys: demoting, disabling or deleting the user revokes them.
type APIKey struct {
	ID            int64
	UserID        int64
	Username      string
	Name          string
	Prefix        string // first several characters of the real key — safe to display, never enough to reconstruct it
	CreatedAtUnix int64
}

// apiKeyPrefixLen is how much of the raw key (including its "ao_"
// marker) is kept as Prefix, for an admin to tell keys apart in a list
// without ever storing enough to reconstruct the real value.
const apiKeyPrefixLen = 11

var (
	ErrInvalidAPIKeyName = errors.New("auth: api key name is required")
	ErrAPIKeyNotFound    = errors.New("auth: api key not found")
	ErrAPIKeyOwnerOnly   = errors.New("auth: only an enabled owner can hold api keys")
)

// GenerateAPIKey creates and persists a new API key, returning the raw
// key exactly once. Like the TMDb credentials in internal/metadata,
// it's never stored or retrievable in plain text again after this call —
// only its SHA-256 hash (for eventual verification, once RequireAuth
// checks it) and a short prefix are kept.
func GenerateAPIKey(ctx context.Context, db *sql.DB, userID int64, name string) (rawKey string, key APIKey, err error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", APIKey{}, ErrInvalidAPIKeyName
	}
	u, err := GetUser(ctx, db, userID)
	if err != nil {
		return "", APIKey{}, err
	}
	if !u.IsOwner() || u.Disabled {
		return "", APIKey{}, ErrAPIKeyOwnerOnly
	}

	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", APIKey{}, fmt.Errorf("auth: generate api key: %w", err)
	}
	rawKey = "ao_" + base64.RawURLEncoding.EncodeToString(raw)
	sum := sha256.Sum256([]byte(rawKey))

	key = APIKey{UserID: u.ID, Username: u.Username, Name: name, Prefix: rawKey[:apiKeyPrefixLen]}
	err = db.QueryRowContext(ctx, `
		INSERT INTO api_keys (user_id, name, key_hash, key_prefix) VALUES (?, ?, ?, ?) RETURNING id, created_at_unix`,
		u.ID, key.Name, hex.EncodeToString(sum[:]), key.Prefix).Scan(&key.ID, &key.CreatedAtUnix)
	if err != nil {
		return "", APIKey{}, fmt.Errorf("auth: generate api key: %w", err)
	}
	return rawKey, key, nil
}

// ProvisionTestAPIKey is exposed only by the local provisioning command. The
// chosen credential persists across restarts and remains normally revocable.
// Its display prefix must not reveal a short test credential in the admin UI.
// The key belongs to the first owner.
func ProvisionTestAPIKey(ctx context.Context, db *sql.DB, name, raw string) (APIKey, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return APIKey{}, ErrInvalidAPIKeyName
	}
	if len(raw) < 4 || len(raw) > 128 || strings.ContainsAny(raw, " \t\r\n") {
		return APIKey{}, errors.New("auth: test key must contain 4 through 128 bytes without whitespace")
	}
	sum := sha256.Sum256([]byte(raw))
	key := APIKey{Name: name, Prefix: "test-key"}
	owner, err := FirstOwner(ctx, db)
	if err != nil {
		return APIKey{}, err
	}
	key.UserID, key.Username = owner.ID, owner.Username
	err = db.QueryRowContext(ctx, `
		INSERT INTO api_keys (user_id, name, key_hash, key_prefix) VALUES (?, ?, ?, ?)
		ON CONFLICT(key_hash) DO UPDATE SET user_id=excluded.user_id, name=excluded.name, key_prefix=excluded.key_prefix
		RETURNING id, created_at_unix`, key.UserID, name, hex.EncodeToString(sum[:]), key.Prefix).Scan(&key.ID, &key.CreatedAtUnix)
	return key, err
}

// ListAPIKeys returns every generated key with its owner, newest first —
// never the hash or the raw value, just what's needed to identify and revoke
// one. Only owners may call the API that lists them.
func ListAPIKeys(ctx context.Context, db *sql.DB) ([]APIKey, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT k.id, k.user_id, COALESCE(u.username, ''), k.name, k.key_prefix, k.created_at_unix
		FROM api_keys k LEFT JOIN users u ON u.id = k.user_id ORDER BY k.created_at_unix DESC, k.id DESC`)
	if err != nil {
		return nil, fmt.Errorf("auth: list api keys: %w", err)
	}
	defer rows.Close()

	var keys []APIKey
	for rows.Next() {
		var k APIKey
		if err := rows.Scan(&k.ID, &k.UserID, &k.Username, &k.Name, &k.Prefix, &k.CreatedAtUnix); err != nil {
			return nil, fmt.Errorf("auth: list api keys: %w", err)
		}
		keys = append(keys, k)
	}
	return keys, rows.Err()
}

// RevokeAPIKey permanently deletes the key with the given id. Returns
// ErrAPIKeyNotFound if it doesn't exist.
func RevokeAPIKey(ctx context.Context, db *sql.DB, id int64) error {
	res, err := db.ExecContext(ctx, `DELETE FROM api_keys WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("auth: revoke api key %d: %w", id, err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("auth: revoke api key %d: %w", id, err)
	}
	if n == 0 {
		return ErrAPIKeyNotFound
	}
	return nil
}

// FirstOwner returns the enabled owner with the lowest id: the account the
// server's command-line tools act as. ErrNoAccount when there is none.
func FirstOwner(ctx context.Context, db *sql.DB) (User, error) {
	u, err := scanUser(db.QueryRowContext(ctx, `SELECT `+userColumns+` FROM users WHERE role = 'owner' AND disabled = 0 ORDER BY id LIMIT 1`))
	if errors.Is(err, sql.ErrNoRows) || errors.Is(err, ErrUserNotFound) {
		return User{}, ErrNoAccount
	}
	return u, err
}
