package auth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

// Role is a user account's server-wide permission level (#278). It is
// separate from a household profile's admin/viewer role, which only gates
// profile PINs and settings inside one account.
type Role string

const (
	// RoleOwner can see and change everything, including other users.
	RoleOwner Role = "owner"
	// RoleViewer can watch and manage the profiles of its own account only.
	RoleViewer Role = "viewer"
)

// Valid reports whether r is a known role.
func (r Role) Valid() bool { return r == RoleOwner || r == RoleViewer }

const (
	// MaxUsers bounds how many accounts one server holds, so a compromised
	// owner session or a script cannot fill the database.
	MaxUsers = 100
	// MaxStreamsLimit is the highest per-user stream limit; 0 means the
	// user has no limit of its own (only the server-wide limits apply).
	MaxStreamsLimit = 32
	// DefaultViewerStreams is suggested for new viewer accounts.
	DefaultViewerStreams = 2
)

var (
	ErrUserNotFound     = errors.New("auth: user not found")
	ErrUsernameTaken    = errors.New("auth: that username is already in use")
	ErrInvalidRole      = errors.New("auth: role must be owner or viewer")
	ErrInvalidMaxStream = fmt.Errorf("auth: stream limit must be 0 (unlimited) to %d", MaxStreamsLimit)
	ErrTooManyUsers     = fmt.Errorf("auth: a server can have at most %d users", MaxUsers)
	ErrLastOwner        = errors.New("auth: the last enabled owner cannot be removed, disabled or demoted")
	ErrPasswordLocked   = errors.New("auth: this account's password can only be changed by an owner")
	// ErrLockedMustChange refuses an account that would have to change a
	// password it is not allowed to change, which would lock it out.
	ErrLockedMustChange = errors.New("auth: a password-locked account cannot be required to change its password")
)

// User is one sign-in account. Profiles, browser sessions and API keys all
// belong to exactly one user.
type User struct {
	ID                 int64
	Username           string
	Role               Role
	Disabled           bool
	MustChangePassword bool
	PasswordLocked     bool
	MaxStreams         int
	CreatedAtUnix      int64
	UpdatedAtUnix      int64
}

// IsOwner reports whether the user has the owner role.
func (u User) IsOwner() bool { return u.Role == RoleOwner }

const userColumns = `id, username, role, disabled, must_change_password, password_locked, max_streams, created_at_unix, updated_at_unix`

type rowScanner interface{ Scan(...any) error }

func scanUser(row rowScanner) (User, error) {
	var u User
	var role string
	err := row.Scan(&u.ID, &u.Username, &role, &u.Disabled, &u.MustChangePassword, &u.PasswordLocked, &u.MaxStreams, &u.CreatedAtUnix, &u.UpdatedAtUnix)
	u.Role = Role(role)
	return u, err
}

// NewUser describes an account an owner creates.
type NewUser struct {
	Username           string
	Password           string
	Role               Role
	MustChangePassword bool
	PasswordLocked     bool
	MaxStreams         int
}

// CreateUser adds an account and gives it the default Admin, Teen and Kids
// profiles in the same transaction.
func CreateUser(ctx context.Context, db *sql.DB, n NewUser) (User, error) {
	username, err := ValidateCredentials(n.Username, n.Password)
	if err != nil {
		return User{}, err
	}
	if !n.Role.Valid() {
		return User{}, ErrInvalidRole
	}
	if n.MaxStreams < 0 || n.MaxStreams > MaxStreamsLimit {
		return User{}, ErrInvalidMaxStream
	}
	if n.PasswordLocked && n.MustChangePassword {
		return User{}, ErrLockedMustChange
	}
	hash, err := HashPassword(n.Password)
	if err != nil {
		return User{}, err
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return User{}, fmt.Errorf("auth: create user: %w", err)
	}
	defer tx.Rollback()
	var count int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM users`).Scan(&count); err != nil {
		return User{}, fmt.Errorf("auth: create user: %w", err)
	}
	if count >= MaxUsers {
		return User{}, ErrTooManyUsers
	}
	u, err := scanUser(tx.QueryRowContext(ctx, `
		INSERT INTO users (username, password_hash, role, must_change_password, password_locked, max_streams)
		VALUES (?, ?, ?, ?, ?, ?) ON CONFLICT(username) DO NOTHING RETURNING `+userColumns,
		username, hash, string(n.Role), n.MustChangePassword, n.PasswordLocked, n.MaxStreams))
	if errors.Is(err, sql.ErrNoRows) {
		return User{}, ErrUsernameTaken
	}
	if err != nil {
		return User{}, fmt.Errorf("auth: create user: %w", err)
	}
	if err := seedDefaultProfiles(ctx, tx, u.ID); err != nil {
		return User{}, err
	}
	return u, tx.Commit()
}

// seedDefaultProfiles gives an account with no profiles the household
// defaults. A fresh install's migrations already seed them for user 1.
func seedDefaultProfiles(ctx context.Context, tx *sql.Tx, userID int64) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO profiles (user_id, name, role, audience_tier)
		SELECT ?, d.name, d.role, d.tier FROM (
			SELECT 1 AS ord, 'Admin' AS name, 'admin' AS role, 'adult' AS tier
			UNION ALL SELECT 2, 'Kids', 'viewer', 'young_child'
			UNION ALL SELECT 3, 'Teen', 'viewer', 'teen'
		) AS d
		WHERE NOT EXISTS (SELECT 1 FROM profiles WHERE user_id = ?)
		ORDER BY d.ord`, userID, userID)
	if err != nil {
		return fmt.Errorf("auth: seed profiles: %w", err)
	}
	return nil
}

// GetUser returns one account.
func GetUser(ctx context.Context, db *sql.DB, id int64) (User, error) {
	u, err := scanUser(db.QueryRowContext(ctx, `SELECT `+userColumns+` FROM users WHERE id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return User{}, ErrUserNotFound
	}
	if err != nil {
		return User{}, fmt.Errorf("auth: get user: %w", err)
	}
	return u, nil
}

// ListUsers returns every account, oldest first.
func ListUsers(ctx context.Context, db *sql.DB) ([]User, error) {
	rows, err := db.QueryContext(ctx, `SELECT `+userColumns+` FROM users ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("auth: list users: %w", err)
	}
	defer rows.Close()
	out := []User{}
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, fmt.Errorf("auth: list users: %w", err)
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

// UserPatch changes some of an account's settings; nil fields are kept.
type UserPatch struct {
	Username           *string
	Role               *Role
	Disabled           *bool
	MustChangePassword *bool
	PasswordLocked     *bool
	MaxStreams         *int
}

// UpdateUser applies p. Disabling an account or changing its role signs it
// out everywhere; demotion also revokes its API keys, which only owners may
// hold. The last enabled owner cannot be disabled or demoted.
func UpdateUser(ctx context.Context, db *sql.DB, id int64, p UserPatch) (User, error) {
	if p.Username != nil {
		name, err := ValidateUsername(*p.Username)
		if err != nil {
			return User{}, err
		}
		p.Username = &name
	}
	if p.Role != nil && !p.Role.Valid() {
		return User{}, ErrInvalidRole
	}
	if p.MaxStreams != nil && (*p.MaxStreams < 0 || *p.MaxStreams > MaxStreamsLimit) {
		return User{}, ErrInvalidMaxStream
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return User{}, fmt.Errorf("auth: update user: %w", err)
	}
	defer tx.Rollback()
	cur, err := scanUser(tx.QueryRowContext(ctx, `SELECT `+userColumns+` FROM users WHERE id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return User{}, ErrUserNotFound
	}
	if err != nil {
		return User{}, fmt.Errorf("auth: update user: %w", err)
	}
	next := cur
	if p.Username != nil {
		next.Username = *p.Username
	}
	if p.Role != nil {
		next.Role = *p.Role
	}
	if p.Disabled != nil {
		next.Disabled = *p.Disabled
	}
	if p.MustChangePassword != nil {
		next.MustChangePassword = *p.MustChangePassword
	}
	if p.PasswordLocked != nil {
		next.PasswordLocked = *p.PasswordLocked
	}
	if p.MaxStreams != nil {
		next.MaxStreams = *p.MaxStreams
	}
	if next.PasswordLocked && next.MustChangePassword {
		return User{}, ErrLockedMustChange
	}
	losesOwner := cur.IsOwner() && !cur.Disabled && (!next.IsOwner() || next.Disabled)
	if losesOwner {
		if err := requireOtherOwner(ctx, tx, id); err != nil {
			return User{}, err
		}
	}
	_, err = tx.ExecContext(ctx, `
		UPDATE users SET username=?, role=?, disabled=?, must_change_password=?, password_locked=?, max_streams=?, updated_at_unix=unixepoch()
		WHERE id=?`, next.Username, string(next.Role), next.Disabled, next.MustChangePassword, next.PasswordLocked, next.MaxStreams, id)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			return User{}, ErrUsernameTaken
		}
		return User{}, fmt.Errorf("auth: update user: %w", err)
	}
	if next.Role != cur.Role || (next.Disabled && !cur.Disabled) {
		if _, err := tx.ExecContext(ctx, `DELETE FROM browser_sessions WHERE user_id = ?`, id); err != nil {
			return User{}, fmt.Errorf("auth: update user: %w", err)
		}
	}
	if !next.IsOwner() {
		if _, err := tx.ExecContext(ctx, `DELETE FROM api_keys WHERE user_id = ?`, id); err != nil {
			return User{}, fmt.Errorf("auth: update user: %w", err)
		}
	}
	updated, err := scanUser(tx.QueryRowContext(ctx, `SELECT `+userColumns+` FROM users WHERE id = ?`, id))
	if err != nil {
		return User{}, fmt.Errorf("auth: update user: %w", err)
	}
	return updated, tx.Commit()
}

// DeleteUser removes an account with its sessions, API keys and profiles. It
// returns the deleted profiles' icon file names so the caller can remove them.
func DeleteUser(ctx context.Context, db *sql.DB, id int64) (icons []string, err error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("auth: delete user: %w", err)
	}
	defer tx.Rollback()
	cur, err := scanUser(tx.QueryRowContext(ctx, `SELECT `+userColumns+` FROM users WHERE id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("auth: delete user: %w", err)
	}
	if cur.IsOwner() && !cur.Disabled {
		if err := requireOtherOwner(ctx, tx, id); err != nil {
			return nil, err
		}
	}
	rows, err := tx.QueryContext(ctx, `SELECT icon FROM profiles WHERE user_id = ? AND icon IS NOT NULL AND icon <> ''`, id)
	if err != nil {
		return nil, fmt.Errorf("auth: delete user: %w", err)
	}
	for rows.Next() {
		var icon string
		if err := rows.Scan(&icon); err != nil {
			rows.Close()
			return nil, fmt.Errorf("auth: delete user: %w", err)
		}
		icons = append(icons, icon)
	}
	rows.Close()
	for _, q := range []string{
		`DELETE FROM browser_sessions WHERE user_id = ?`,
		`DELETE FROM api_keys WHERE user_id = ?`,
		`DELETE FROM profiles WHERE user_id = ?`,
		`DELETE FROM users WHERE id = ?`,
	} {
		if _, err := tx.ExecContext(ctx, q, id); err != nil {
			return nil, fmt.Errorf("auth: delete user: %w", err)
		}
	}
	return icons, tx.Commit()
}

func requireOtherOwner(ctx context.Context, tx *sql.Tx, id int64) error {
	var n int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM users WHERE role = 'owner' AND disabled = 0 AND id <> ?`, id).Scan(&n); err != nil {
		return fmt.Errorf("auth: check owners: %w", err)
	}
	if n == 0 {
		return ErrLastOwner
	}
	return nil
}

// SetUserPassword is an owner setting another account's password without the
// old one. The account is signed out everywhere; mustChange makes it choose a
// new password at its next sign-in.
func SetUserPassword(ctx context.Context, db *sql.DB, id int64, next string, mustChange bool) error {
	if err := ValidatePassword(next); err != nil {
		return err
	}
	u, err := GetUser(ctx, db, id)
	if err != nil {
		return err
	}
	if u.PasswordLocked && mustChange {
		return ErrLockedMustChange
	}
	return setPassword(ctx, db, id, "", next, mustChange, 0)
}
