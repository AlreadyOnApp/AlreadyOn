package instancetls

import (
	"context"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"database/sql"
	"encoding/pem"
	"errors"
	"path/filepath"
	"strings"
	"time"
)

// Uploaded certificates are PEM only. The explicitly supplied CA bundle is a
// trust decision by the administrator, not a claim of public browser trust.
func uploadedMaterial(certPEM, keyPEM, caPEM []byte, now time.Time) (*Material, error) {
	if len(certPEM)+len(keyPEM)+len(caPEM) > 256<<10 {
		return nil, errors.New("TLS: certificate files are too large")
	}
	// Reject mixed bundles rather than accepting/retaining extra private keys
	// in a certificate field that administrators expect to be non-secret.
	for _, data := range [][]byte{certPEM, caPEM} {
		rest := data
		if len(strings.TrimSpace(string(rest))) == 0 {
			return nil, errors.New("TLS: certificate and CA files are required")
		}
		for len(strings.TrimSpace(string(rest))) > 0 {
			if !strings.HasPrefix(strings.TrimSpace(string(rest)), "-----BEGIN CERTIFICATE-----") {
				return nil, errors.New("TLS: certificate files must contain only PEM certificates")
			}
			block, remaining := pem.Decode(rest)
			if block == nil || block.Type != "CERTIFICATE" {
				return nil, errors.New("TLS: certificate files must contain only PEM certificates")
			}
			rest = remaining
		}
	}
	keyBlock, remaining := pem.Decode(keyPEM)
	if keyBlock == nil || len(strings.TrimSpace(string(remaining))) != 0 || len(keyBlock.Headers) != 0 || (keyBlock.Type != "PRIVATE KEY" && keyBlock.Type != "RSA PRIVATE KEY" && keyBlock.Type != "EC PRIVATE KEY") {
		return nil, errors.New("TLS: upload exactly one unencrypted PEM private key")
	}
	pair, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return nil, errors.New("TLS: certificate and unencrypted PEM key must match")
	}
	leaf := pair.Leaf
	if leaf == nil {
		leaf, err = x509.ParseCertificate(pair.Certificate[0])
		if err != nil {
			return nil, errors.New("TLS: invalid server certificate")
		}
		pair.Leaf = leaf
	}
	if leaf.IsCA || len(leaf.DNSNames)+len(leaf.IPAddresses) == 0 {
		return nil, errors.New("TLS: upload a server certificate containing DNS/IP names, not a CA certificate")
	}
	if _, err := HealthName(leaf); err != nil {
		return nil, err
	}
	for _, name := range leaf.DNSNames {
		if _, _, err := addresses([]string{strings.TrimPrefix(name, "*.")}); err != nil || strings.TrimSpace(name) != name {
			return nil, errors.New("TLS: certificate DNS names must be valid names, not URLs or ports")
		}
	}
	strong := false
	switch key := leaf.PublicKey.(type) {
	case *rsa.PublicKey:
		strong = key.N.BitLen() >= 2048
	case *ecdsa.PublicKey:
		strong = key.Curve.Params().BitSize >= 256
	case ed25519.PublicKey:
		strong = true
	}
	if !strong {
		return nil, errors.New("TLS: use RSA 2048-bit or stronger, ECDSA P-256 or stronger, or Ed25519")
	}
	roots := x509.NewCertPool()
	var root *x509.Certificate
	rest := caPEM
	for len(strings.TrimSpace(string(rest))) > 0 {
		block, remaining := pem.Decode(rest)
		if block == nil || block.Type != "CERTIFICATE" {
			return nil, errors.New("TLS: provide a PEM CA certificate bundle without private keys")
		}
		ca, err := x509.ParseCertificate(block.Bytes)
		if err != nil || !ca.IsCA || now.Before(ca.NotBefore) || !now.Before(ca.NotAfter) {
			return nil, errors.New("TLS: CA certificates must be valid and unexpired")
		}
		roots.AddCert(ca)
		if root == nil {
			root = ca
		}
		rest = remaining
	}
	if root == nil {
		return nil, errors.New("TLS: include the issuing CA trust bundle")
	}
	intermediates := x509.NewCertPool()
	previous := leaf
	for _, der := range pair.Certificate[1:] {
		cert, err := x509.ParseCertificate(der)
		if err != nil {
			return nil, errors.New("TLS: invalid certificate chain")
		}
		if previous.CheckSignatureFrom(cert) != nil {
			return nil, errors.New("TLS: certificate chain must be ordered leaf first, then its issuing intermediates")
		}
		intermediates.AddCert(cert)
		previous = cert
	}
	chains, err := leaf.Verify(x509.VerifyOptions{Roots: roots, Intermediates: intermediates, CurrentTime: now, KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}})
	if err != nil {
		return nil, errors.New("TLS: server certificate must be current, valid for server authentication and chain to the supplied CA")
	}
	chain := chains[0]
	root = chain[len(chain)-1]
	// Export only parsed certificates, never mixed PEM input containing keys.
	cleanCA := []byte{}
	for _, ca := range chains[0][1:] {
		cleanCA = append(cleanCA, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: ca.Raw})...)
	}
	if len(cleanCA) == 0 {
		return nil, errors.New("TLS: a separate issuing CA is required")
	}
	return &Material{certificate: pair, rootPEM: cleanCA, root: root}, nil
}

// Upload validates everything before persisting and activating one pair for
// both ports. No private material is returned; generated CA/key stay intact.
func (m *Manager) Upload(ctx context.Context, certificate, key, ca []byte) error {
	if !m.acmeMu.TryLock() {
		return errors.New("ACME validation is running; wait before changing certificates")
	}
	defer m.acmeMu.Unlock()
	if m.acmeStatus(ctx).Running {
		return errors.New("ACME validation is running; wait before changing certificates")
	}
	material, err := uploadedMaterial(certificate, key, ca, time.Now())
	if err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.activate(ctx, material, "user-provided", func() error {
		tx, err := m.db.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		defer tx.Rollback()
		if _, err = tx.ExecContext(ctx, `UPDATE acme_tls SET enabled=0,active=0,configured=0 WHERE id=1`); err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO uploaded_tls(id, certificate_pem, private_key_pem, ca_pem, active) VALUES(1, ?, ?, ?, 1) ON CONFLICT(id) DO UPDATE SET certificate_pem=excluded.certificate_pem, private_key_pem=excluded.private_key_pem, ca_pem=excluded.ca_pem, active=1`, certificate, key, material.rootPEM)
		if err != nil {
			return err
		}
		return tx.Commit()
	})
}

// UseSelfSigned restores the original database identity, not a fresh CA.
func (m *Manager) UseSelfSigned(ctx context.Context) error {
	if !m.acmeMu.TryLock() {
		return errors.New("ACME validation is running; wait before changing certificates")
	}
	defer m.acmeMu.Unlock()
	if m.acmeStatus(ctx).Running {
		return errors.New("ACME validation is running; wait before changing certificates")
	}
	names, err := storedNames(ctx, m.db)
	if err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	material, err := Ensure(ctx, m.db, names, time.Now())
	if err != nil {
		return err
	}
	return m.activate(ctx, material, "self-signed", func() error {
		tx, err := m.db.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		defer tx.Rollback()
		if _, err = tx.ExecContext(ctx, `UPDATE uploaded_tls SET active=0 WHERE id=1`); err != nil {
			return err
		}
		if _, err = tx.ExecContext(ctx, `UPDATE acme_tls SET enabled=0,active=0,configured=0 WHERE id=1`); err != nil {
			return err
		}
		return tx.Commit()
	})
}

// RestoreSelfSigned is explicit offline recovery for an expired upload. Run
// from a trusted server terminal, then restart both listeners. It does not
// require loading or accepting the invalid uploaded certificate.
func RestoreSelfSigned(ctx context.Context, db *sql.DB, dataDir string) error {
	names, err := storedNames(ctx, db)
	if err != nil {
		return err
	}
	material, err := Ensure(ctx, db, names, time.Now())
	if err != nil {
		return err
	}
	if err := material.Export(filepath.Join(dataDir, "tls")); err != nil {
		return err
	}
	_, err = db.ExecContext(ctx, `UPDATE acme_tls SET enabled=0,active=0,configured=0 WHERE id=1`)
	if err != nil {
		return err
	}
	_, err = db.ExecContext(ctx, `UPDATE uploaded_tls SET active=0 WHERE id=1`)
	return err
}

// Export first, then persist selection and finally swap handshake material.
// Any export/persistence failure restores the previous exports and fails closed.
func (m *Manager) activate(_ context.Context, material *Material, mode string, persist func() error) error {
	if err := material.Export(m.dir); err != nil {
		if restore := m.material.Export(m.dir); restore != nil {
			return errors.New("TLS: activation and export recovery failed; restart required")
		}
		return errors.New("TLS: could not export certificate; active certificate retained")
	}
	if err := persist(); err != nil {
		if restore := m.material.Export(m.dir); restore != nil {
			return errors.New("TLS: persistence and export recovery failed; restart required")
		}
		return errors.New("TLS: could not persist certificate; active certificate retained")
	}
	m.material, m.mode = material, mode
	return nil
}

// HealthName selects a syntactically valid SAN for local verified checks.
// Wildcard-only certificates need an additional concrete name, not a bypass.
func HealthName(leaf *x509.Certificate) (string, error) {
	for _, name := range certificateNames(leaf) {
		if dns, ips, err := addresses([]string{name}); err == nil {
			if len(dns) > 0 {
				return dns[0], nil
			}
			return ips[0].String(), nil
		}
	}
	return "", errors.New("TLS: include a concrete DNS/IP name alongside any wildcard for verified local health checks")
}
