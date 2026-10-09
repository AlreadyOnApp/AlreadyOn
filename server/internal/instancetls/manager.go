package instancetls

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"database/sql"
	"encoding/pem"
	"errors"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Manager supplies one identity to both listeners and replaces it only after
// durable generation/export succeeds. Existing TLS connections keep their leaf.
type Manager struct {
	acmeMu        sync.Mutex
	challengeMu   sync.RWMutex
	acmeRunning   bool
	challenges    map[string]string
	challengeHost string
	mu            sync.RWMutex
	db            *sql.DB
	dir           string
	material      *Material
	mode          string
}

// Status is public certificate metadata; it intentionally contains no keys.
type Status struct {
	IssuedAt       time.Time  `json:"issuedAt"`
	Mode           string     `json:"mode"`
	Names          []string   `json:"names"`
	Fingerprint    string     `json:"fingerprint"`
	ExpiresAt      time.Time  `json:"expiresAt"`
	TrustExpiresAt time.Time  `json:"trustExpiresAt"`
	ACME           ACMEStatus `json:"acme"`
}

// Open prepares persistent TLS before network listeners accept requests.
func Open(ctx context.Context, db *sql.DB, dataDir string) (*Manager, error) {
	m := &Manager{db: db, dir: filepath.Join(dataDir, "tls")}
	// Interrupted orders keep their durable cooldown and previous identity.
	if _, err := db.ExecContext(ctx, `UPDATE acme_tls SET message='Previous ACME attempt interrupted; existing certificate retained' WHERE id=1 AND message='Starting Let''s Encrypt validation'`); err != nil {
		return nil, err
	}
	var cert, key, ca []byte
	var host string
	var enabled bool
	if err := db.QueryRowContext(ctx, `SELECT active,active_hostname,certificate_pem,private_key_pem FROM acme_tls WHERE id=1`).Scan(&enabled, &host, &cert, &key); err != nil {
		return nil, err
	}
	if enabled {
		material, err := publicACMEMaterial(cert, key, host)
		if err != nil {
			return nil, err
		}
		if err = material.Export(m.dir); err != nil {
			return nil, err
		}
		m.material, m.mode = material, "acme"
		return m, nil
	}
	err := db.QueryRowContext(ctx, `SELECT certificate_pem, private_key_pem, ca_pem FROM uploaded_tls WHERE id=1 AND active=1`).Scan(&cert, &key, &ca)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	if err == nil {
		material, err := uploadedMaterial(cert, key, ca, time.Now())
		if err != nil {
			return nil, err
		}
		if err := material.Export(m.dir); err != nil {
			return nil, err
		}
		m.material, m.mode = material, "user-provided"
		return m, nil
	}
	names, err := storedNames(ctx, db)
	if err != nil {
		return nil, err
	}
	if err := m.SetNames(ctx, names); err != nil {
		return nil, err
	}
	return m, nil
}

func storedNames(ctx context.Context, db *sql.DB) ([]string, error) {
	var data []byte
	err := db.QueryRowContext(ctx, `SELECT server_cert_pem FROM instance_tls WHERE id = 1`).Scan(&data)
	if errors.Is(err, sql.ErrNoRows) {
		names := []string{"localhost", "127.0.0.1", "::1"}
		if host, err := os.Hostname(); err == nil && hostname.MatchString(host) {
			names = append(names, host)
		}
		// Native hosts can advertise their interfaces; container host/LAN names
		// must still be supplied explicitly because bridge addresses are not them.
		if addrs, err := net.InterfaceAddrs(); err == nil {
			for _, addr := range addrs {
				if ip, _, err := net.ParseCIDR(addr.String()); err == nil && !ip.IsLoopback() && !ip.IsLinkLocalUnicast() && !ip.IsUnspecified() && len(names) < 32 {
					names = append(names, ip.String())
				}
			}
		}
		return names, nil
	}
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, errors.New("TLS: invalid stored certificate")
	}
	leaf, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, errors.New("TLS: invalid stored certificate")
	}
	return certificateNames(leaf), nil
}

func certificateNames(leaf *x509.Certificate) []string {
	names := append([]string(nil), leaf.DNSNames...)
	for _, ip := range leaf.IPAddresses {
		names = append(names, ip.String())
	}
	return names
}

// SetNames validates administrator-entered SANs and renews under the same CA.
// Loopback names stay present for verified local health checks and recovery.
func (m *Manager) SetNames(ctx context.Context, names []string) error {
	if _, _, err := addresses(names); err != nil {
		return err
	}
	names = append(append([]string(nil), names...), "localhost", "127.0.0.1", "::1")
	dns, ips, err := addressesWithDefaults(names)
	if err != nil {
		return err
	}
	canonical := append([]string(nil), dns...)
	for _, ip := range ips {
		canonical = append(canonical, ip.String())
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.mode == "user-provided" || m.mode == "acme" {
		return errors.New("TLS: certificate names are managed by the uploaded certificate")
	}
	material, err := Ensure(ctx, m.db, canonical, time.Now())
	if err != nil {
		return err
	}
	if err := material.Export(m.dir); err != nil {
		return err
	}
	m.material = material
	m.mode = "self-signed"
	return nil
}

func addressesWithDefaults(names []string) ([]string, []net.IP, error) {
	// Deduplicate before applying the bounded SAN count, including mandatory names.
	seen := make(map[string]bool)
	var unique []string
	for _, name := range names {
		if !seen[name] {
			seen[name] = true
			unique = append(unique, name)
		}
	}
	return addresses(unique)
}

// Config chooses the currently active leaf on each new TLS handshake.
func (m *Manager) Config() *tls.Config {
	return &tls.Config{MinVersion: tls.VersionTLS12, GetCertificate: func(_ *tls.ClientHelloInfo) (*tls.Certificate, error) {
		m.mu.RLock()
		defer m.mu.RUnlock()
		return &m.material.certificate, nil
	}}
}

func (m *Manager) Status() Status {
	m.mu.RLock()
	defer m.mu.RUnlock()
	leaf := m.material.certificate.Leaf
	return Status{leaf.NotBefore, m.mode, certificateNames(leaf), m.material.Fingerprint(), leaf.NotAfter, m.material.root.NotAfter, m.acmeStatus(context.Background())}
}

func (m *Manager) TrustPEM() []byte {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.material.TrustPEM()
}

// Renew keeps long-running instances from depending on a restart for leaf
// renewal. The lock prevents a timer from overwriting concurrently edited SANs.
func (m *Manager) Renew(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	// Uploaded certificates are renewed by their owner; never replace them
	// automatically with a generated certificate or silently change trust.
	if m.mode == "acme" || m.mode == "user-provided" || m.material.certificate.Leaf.NotAfter.Equal(m.material.root.NotAfter) {
		return nil
	}
	if time.Now().Add(30 * 24 * time.Hour).Before(m.material.certificate.Leaf.NotAfter) {
		return nil
	}
	material, err := Ensure(ctx, m.db, certificateNames(m.material.certificate.Leaf), time.Now())
	if err != nil {
		return err
	}
	if err := material.Export(m.dir); err != nil {
		return err
	}
	m.material = material
	return nil
}

// Export creates owner-only local files for Vite and verified health tools.
// The root private key is never exported. SQLite remains authoritative.
func (m *Material) Export(dir string) error {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	if err := os.Chmod(dir, 0o700); err != nil {
		return err
	}
	keyDER, err := x509.MarshalPKCS8PrivateKey(m.certificate.PrivateKey)
	if err != nil {
		return err
	}
	var cert []byte
	for _, der := range m.certificate.Certificate {
		cert = append(cert, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})...)
	}
	key := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER})
	// Publish the watched bundle last so consumers observe matching trust/pair.
	for _, file := range []struct {
		name string
		data []byte
	}{{"ca.pem", m.rootPEM}, {"server.pem", cert}, {"server.key", key}, {"server-bundle.pem", append(append([]byte(nil), cert...), key...)}} {
		if err := atomicFile(dir, file.name, file.data); err != nil {
			return err
		}
	}
	return nil
}

func atomicFile(dir, name string, data []byte) error {
	f, err := os.CreateTemp(dir, ".tls-")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if _, err := f.Write(data); err != nil {
		f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(f.Name(), filepath.Join(dir, name))
}
