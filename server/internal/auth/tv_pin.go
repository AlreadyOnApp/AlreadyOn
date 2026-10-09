package auth

import (
	"context"
	"database/sql"
	"errors"
	"strings"
)

var ErrInvalidTVPIN = errors.New("TV PIN must contain 6–12 numeric digits")
var ErrTVPINTaken = errors.New("That TV PIN is already in use; choose a different PIN")

// Discovery never advertises usernames or PINs; sign-in requires both.
// A server-specific Argon2id salt permits a unique indexed lookup with one
// expensive hash per guess, rather than a fast hash or work proportional to users.
func tvPINDigest(ctx context.Context, db *sql.DB, pin string) ([]byte, error) {
	if len(pin) < 6 || len(pin) > 12 || strings.IndexFunc(pin, func(r rune) bool { return r < '0' || r > '9' }) >= 0 {
		return nil, ErrInvalidTVPIN
	}
	var salt []byte
	if err := db.QueryRowContext(ctx, `SELECT salt FROM tv_pairing_salt WHERE id = 1`).Scan(&salt); err != nil {
		return nil, err
	}
	return argonKey([]byte(pin), salt, argonTime, argonMemoryKiB, argonThreads, argonKeyLen), nil
}

func HasTVPIN(ctx context.Context, db *sql.DB, userID int64) (bool, error) {
	var enabled bool
	err := db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM tv_pairing_pins WHERE user_id = ?)`, userID).Scan(&enabled)
	return enabled, err
}

func TVPairingEnabled(ctx context.Context, db *sql.DB) bool {
	var enabled bool
	_ = db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM tv_pairing_pins p JOIN users u ON u.id = p.user_id WHERE u.disabled = 0 AND u.must_change_password = 0)`).Scan(&enabled)
	return enabled
}

func SetTVPIN(ctx context.Context, db *sql.DB, userID int64, pin string) error {
	digest, err := tvPINDigest(ctx, db, pin)
	if err != nil {
		return err
	}
	var other int64
	err = db.QueryRowContext(ctx, `SELECT user_id FROM tv_pairing_pins WHERE pin_digest = ?`, digest).Scan(&other)
	if err == nil && other != userID {
		return ErrTVPINTaken
	}
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	_, err = db.ExecContext(ctx, `INSERT INTO tv_pairing_pins(user_id, pin_digest) VALUES (?, ?) ON CONFLICT(user_id) DO UPDATE SET pin_digest = excluded.pin_digest`, userID, digest)
	// The unique constraint also handles concurrent attempts to select the same PIN.
	if err != nil && strings.Contains(err.Error(), "UNIQUE constraint failed") {
		return ErrTVPINTaken
	}
	return err
}

func ClearTVPIN(ctx context.Context, db *sql.DB, userID int64) error {
	_, err := db.ExecContext(ctx, `DELETE FROM tv_pairing_pins WHERE user_id = ?`, userID)
	return err
}

func AuthenticateTVPIN(ctx context.Context, db *sql.DB, username, pin string) (User, bool, error) {
	digest, err := tvPINDigest(ctx, db, pin)
	if errors.Is(err, ErrInvalidTVPIN) {
		return User{}, false, nil
	}
	if err != nil {
		return User{}, false, err
	}
	var id int64
	err = db.QueryRowContext(ctx, `SELECT p.user_id FROM tv_pairing_pins p JOIN users u ON u.id = p.user_id WHERE p.pin_digest = ? AND u.username = ?`, digest, strings.TrimSpace(username)).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return User{}, false, nil
	}
	if err != nil {
		return User{}, false, err
	}
	user, err := GetUser(ctx, db, id)
	return user, err == nil && !user.Disabled && !user.MustChangePassword, err
}
