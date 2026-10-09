package profile

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"
)

var ErrInvalidPIN = errors.New("PIN must be 4-6 digits")

var pinPattern = regexp.MustCompile(`^\d{4,6}$`)

// hashCost is bcrypt's work factor; tests lower it to keep them fast.
var hashCost = bcrypt.DefaultCost

// Wrong PINs back off: the third consecutive failure locks the profile
// for lockBase, and each further failure doubles that, up to lockMax.
const (
	freeAttempts = 2
	lockBase     = 5 * time.Second
	lockMax      = 15 * time.Minute
)

// verifyMu serializes verification so a burst of parallel guesses can't
// all pass the lock check before the failure count catches up. A
// household-sized server can afford one bcrypt comparison at a time.
var verifyMu sync.Mutex

// SetPIN stores a hash of pin and clears any failed-attempt lockout.
func SetPIN(ctx context.Context, db *sql.DB, id int64, pin string) error {
	if !pinPattern.MatchString(pin) {
		return ErrInvalidPIN
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(pin), hashCost)
	if err != nil {
		return err
	}
	return execPIN(ctx, db, id, string(hash))
}

// ClearPIN removes a profile's PIN. Until sessions exist this doubles as
// the admin reset for a forgotten PIN.
func ClearPIN(ctx context.Context, db *sql.DB, id int64) error {
	return execPIN(ctx, db, id, nil)
}

func execPIN(ctx context.Context, db *sql.DB, id int64, hash any) error {
	res, err := db.ExecContext(ctx, `UPDATE profiles SET pin_hash = ?, pin_failures = 0, pin_locked_until_unix = 0,
		updated_at_unix = unixepoch() WHERE id = ?`, hash, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

// VerifyResult reports one PIN check. When RetryAfter is set the profile
// is locked and the PIN was not even compared.
type VerifyResult struct {
	Valid      bool
	RetryAfter time.Duration
}

// VerifyPIN checks pin against the stored hash. A profile with no PIN
// always verifies. The comparison is constant-time (bcrypt), and the
// response never reveals the hash.
func VerifyPIN(ctx context.Context, db *sql.DB, id int64, pin string, now time.Time) (VerifyResult, error) {
	verifyMu.Lock()
	defer verifyMu.Unlock()

	var hash sql.NullString
	var failures int
	var lockedUntil int64
	err := db.QueryRowContext(ctx, `SELECT pin_hash, pin_failures, pin_locked_until_unix FROM profiles WHERE id = ?`, id).
		Scan(&hash, &failures, &lockedUntil)
	if errors.Is(err, sql.ErrNoRows) {
		return VerifyResult{}, ErrNotFound
	}
	if err != nil {
		return VerifyResult{}, err
	}
	if !hash.Valid {
		return VerifyResult{Valid: true}, nil
	}
	if wait := time.Unix(lockedUntil, 0).Sub(now); wait > 0 {
		return VerifyResult{RetryAfter: wait.Truncate(time.Second) + time.Second}, nil
	}

	if bcrypt.CompareHashAndPassword([]byte(hash.String), []byte(pin)) == nil {
		if failures > 0 {
			if _, err := db.ExecContext(ctx, `UPDATE profiles SET pin_failures = 0, pin_locked_until_unix = 0 WHERE id = ?`, id); err != nil {
				return VerifyResult{}, err
			}
		}
		return VerifyResult{Valid: true}, nil
	}

	failures++
	var until int64
	var wait time.Duration
	if failures > freeAttempts {
		wait = lockDuration(failures)
		until = now.Add(wait).Unix()
	}
	if _, err := db.ExecContext(ctx, `UPDATE profiles SET pin_failures = ?, pin_locked_until_unix = ? WHERE id = ?`, failures, until, id); err != nil {
		return VerifyResult{}, err
	}
	return VerifyResult{RetryAfter: wait}, nil
}

func lockDuration(failures int) time.Duration {
	d := lockBase
	for i := freeAttempts + 1; i < failures && d < lockMax; i++ {
		d *= 2
	}
	return min(d, lockMax)
}
