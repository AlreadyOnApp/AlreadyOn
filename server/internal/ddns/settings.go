// Package ddns manages one public IPv4 hostname per provider, using fixed HTTPS
// endpoints. Credentials are write-only and protected by database permissions.
package ddns

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
	"slices"
	"strings"
)

var Providers = []string{"none", "cloudflare", "no-ip"}
var ErrInvalid = errors.New("DDNS configuration is invalid")
var ErrBusy = errors.New("a DDNS operation is already running; try again shortly")
var hostPattern = regexp.MustCompile(`^([a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?\.)+[a-z]([a-z0-9-]{0,61}[a-z0-9])?$`)
var zonePattern = regexp.MustCompile(`^[a-fA-F0-9]{32}$`)

// Config exposes only non-secret setup and the latest update/verification state.
// A successful provider update is distinct from public DNS verification.
type Config struct {
	Provider     string `json:"provider"`
	Hostname     string `json:"hostname"`
	ZoneID       string `json:"zoneId"`
	Username     string `json:"username"`
	Secret       string `json:"-"`
	HasSecret    bool   `json:"hasSecret"`
	Enabled      bool   `json:"enabled"`
	LastAttempt  int64  `json:"lastAttempt"`
	LastUpdate   int64  `json:"lastUpdate"`
	LastVerified int64  `json:"lastVerified"`
	LastIP       string `json:"lastIp"`
	LastError    string `json:"lastError"`
	Failures     int    `json:"-"`
	NextAttempt  int64  `json:"nextAttempt"`
	Blocked      bool   `json:"blocked"`
}

type Update struct {
	Provider    string  `json:"provider"`
	Hostname    string  `json:"hostname"`
	ZoneID      string  `json:"zoneId"`
	Username    string  `json:"username"`
	Secret      *string `json:"secret,omitempty"`
	ClearSecret bool    `json:"clearSecret,omitempty"`
	Enabled     bool    `json:"enabled"`
}

type Status struct {
	Provider string   `json:"provider"`
	Configs  []Config `json:"configs"`
	Running  string   `json:"running"`
}

func load(ctx context.Context, db *sql.DB, provider string) (Config, error) {
	c := Config{Provider: provider}
	err := db.QueryRowContext(ctx, `SELECT hostname,zone_id,username,secret,enabled,last_attempt,last_update,last_verified,last_ip,last_error,failures,next_attempt,blocked FROM ddns_configs WHERE provider=?`, provider).Scan(&c.Hostname, &c.ZoneID, &c.Username, &c.Secret, &c.Enabled, &c.LastAttempt, &c.LastUpdate, &c.LastVerified, &c.LastIP, &c.LastError, &c.Failures, &c.NextAttempt, &c.Blocked)
	if errors.Is(err, sql.ErrNoRows) {
		err = nil
	}
	c.HasSecret = c.Secret != ""
	return c, err
}

func (m *Manager) Status(ctx context.Context) (Status, error) {
	s := Status{Configs: []Config{}, Running: m.running.Load().(string)}
	if err := m.db.QueryRowContext(ctx, `SELECT ddns_provider FROM server_settings WHERE id=1`).Scan(&s.Provider); err != nil {
		return s, err
	}
	for _, p := range Providers[1:] {
		c, err := load(ctx, m.db, p)
		if err != nil {
			return s, err
		}
		s.Configs = append(s.Configs, c)
	}
	return s, nil
}

// Save keeps secrets provider-local, preserves an omitted secret and clears
// status when setup changes. Saving alone does not make a network request.
func (m *Manager) Save(ctx context.Context, u Update) (Status, error) {
	if !m.mu.TryLock() {
		return Status{}, ErrBusy
	}
	defer m.mu.Unlock()
	if !slices.Contains(Providers, u.Provider) {
		return Status{}, ErrInvalid
	}
	if u.Provider == "none" {
		_, err := m.db.ExecContext(ctx, `UPDATE server_settings SET ddns_provider='none' WHERE id=1`)
		if err != nil {
			return Status{}, err
		}
		return m.Status(ctx)
	}
	c, err := load(ctx, m.db, u.Provider)
	if err != nil {
		return Status{}, err
	}
	oldHostname := c.Hostname
	c.Hostname = strings.ToLower(strings.TrimSpace(u.Hostname))
	// Zone IDs are resolved from the token, never trusted from form input.
	if u.Provider == "cloudflare" && (oldHostname != c.Hostname || u.Secret != nil) {
		c.ZoneID = ""
	}
	c.Username = strings.TrimSpace(u.Username)
	c.Enabled = true
	if u.Secret != nil {
		c.Secret = *u.Secret
	}
	if u.ClearSecret {
		c.Secret = ""
		c.Enabled = false
	}
	if len(c.Hostname) > 253 || !hostPattern.MatchString(c.Hostname) || strings.HasSuffix(c.Hostname, ".local") || strings.HasSuffix(c.Hostname, ".localhost") || len(c.Username) > 256 || strings.ContainsAny(c.Username, "\r\n:") || len(c.Secret) > 4096 || strings.ContainsAny(c.Secret, "\r\n") {
		return Status{}, ErrInvalid
	}
	switch u.Provider {
	case "cloudflare":
		c.Username = ""
	default:
		if c.Provider == "no-ip" && c.Hostname == "all.ddnskey.com" {
			return Status{}, ErrInvalid
		}
		c.ZoneID = ""
	}
	if c.Enabled && (c.Secret == "" || (c.Provider == "no-ip" && c.Username == "")) {
		return Status{}, ErrInvalid
	}
	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return Status{}, err
	}
	defer tx.Rollback()
	_, err = tx.ExecContext(ctx, `INSERT INTO ddns_configs(provider,hostname,zone_id,username,secret,enabled) VALUES(?,?,?,?,?,?) ON CONFLICT(provider) DO UPDATE SET hostname=excluded.hostname,zone_id=excluded.zone_id,username=excluded.username,secret=excluded.secret,enabled=excluded.enabled,last_attempt=0,last_update=0,last_verified=0,last_ip='',last_error='',failures=0,next_attempt=0,blocked=0`, c.Provider, c.Hostname, c.ZoneID, c.Username, c.Secret, c.Enabled)
	if err != nil {
		return Status{}, err
	}
	if _, err = tx.ExecContext(ctx, `UPDATE server_settings SET ddns_provider=? WHERE id=1`, c.Provider); err != nil {
		return Status{}, err
	}
	if err = tx.Commit(); err != nil {
		return Status{}, err
	}
	return m.Status(ctx)
}
