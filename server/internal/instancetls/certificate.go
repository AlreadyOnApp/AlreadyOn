// Package instancetls maintains a private trust identity per database. Listener
// integration is separate: preparing material never opens a network port.
package instancetls

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"database/sql"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"net"
	"regexp"
	"strings"
	"time"
)

var hostname = regexp.MustCompile(`^([a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?\.)*[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?$`)

// Material contains the server pair and its public trust anchor. Private keys
// remain unexported, and callers must not serialize or log this value.
type Material struct {
	certificate tls.Certificate
	rootPEM     []byte
	root        *x509.Certificate
}

// Config returns a fresh server-only TLS configuration for either listener.
// Both listeners should receive configurations from the same Material.
func (m *Material) Config() *tls.Config {
	return &tls.Config{MinVersion: tls.VersionTLS12, Certificates: []tls.Certificate{m.certificate}}
}

// TrustPEM exports only the public instance CA certificate, never its key.
func (m *Material) TrustPEM() []byte { return append([]byte(nil), m.rootPEM...) }

// Fingerprint is the SHA-256 trust-anchor digest for out-of-band verification.
func (m *Material) Fingerprint() string {
	digest := sha256.Sum256(m.root.Raw)
	return hex.EncodeToString(digest[:])
}

// Ensure loads the database identity or creates it once. Names are explicit DNS
// names/IP addresses, not URLs or ports; never trust a request's Host header to
// mint certificates. Expired/corrupt identities fail closed, not silently reset.
// Changed addresses or an earlier short-lived leaf renew under the same CA,
// without extending the instance's original ten-year trust lifetime.
func Ensure(ctx context.Context, db *sql.DB, names []string, now time.Time) (*Material, error) {
	dns, ips, err := addresses(names)
	if err != nil {
		return nil, err
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	var rootPEM, rootKeyPEM, certPEM, keyPEM []byte
	err = tx.QueryRowContext(ctx, `SELECT root_cert_pem, root_key_pem, server_cert_pem, server_key_pem FROM instance_tls WHERE id = 1`).Scan(&rootPEM, &rootKeyPEM, &certPEM, &keyPEM)
	newIdentity := errors.Is(err, sql.ErrNoRows)
	if err != nil && !newIdentity {
		return nil, err
	}
	if newIdentity {
		rootPEM, rootKeyPEM, err = createRoot(now)
		if err != nil {
			return nil, err
		}
	}
	rootPair, err := tls.X509KeyPair(rootPEM, rootKeyPEM)
	if err != nil {
		return nil, errors.New("TLS: invalid stored trust identity")
	}
	root := rootPair.Leaf
	if root == nil {
		root, err = x509.ParseCertificate(rootPair.Certificate[0])
	}
	if err != nil || !root.IsCA || root.CheckSignatureFrom(root) != nil || now.Before(root.NotBefore) || !now.Before(root.NotAfter) {
		return nil, errors.New("TLS: invalid or expired instance trust anchor; administrator recovery required")
	}
	var pair tls.Certificate
	renew := newIdentity
	if !newIdentity {
		pair, err = tls.X509KeyPair(certPEM, keyPEM)
		if err != nil {
			return nil, errors.New("TLS: invalid stored server key pair")
		}
		leaf := pair.Leaf
		if leaf == nil {
			leaf, err = x509.ParseCertificate(pair.Certificate[0])
			pair.Leaf = leaf
		}
		roots := x509.NewCertPool()
		roots.AddCert(root)
		if err != nil {
			return nil, errors.New("TLS: invalid stored server certificate")
		}
		if _, err := leaf.Verify(x509.VerifyOptions{Roots: roots, CurrentTime: now, KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}}); err != nil {
			// An expired leaf signed by our valid CA can be renewed without changing trust.
			var invalid x509.CertificateInvalidError
			if !errors.As(err, &invalid) || invalid.Reason != x509.Expired || now.Before(leaf.NotAfter) || leaf.CheckSignatureFrom(root) != nil {
				return nil, errors.New("TLS: stored certificate verification failed")
			}
			renew = true
		}
		// Upgrade earlier one-year leaves without changing the trusted CA. The
		// ten-year identity has a fixed expiry; never silently extend its trust.
		renew = renew || leaf.NotAfter.Before(root.NotAfter) || !sameAddresses(leaf, dns, ips)
	}
	if renew {
		certPEM, keyPEM, err = createLeaf(now, root, rootPair.PrivateKey, dns, ips)
		if err != nil {
			return nil, err
		}
		pair, err = tls.X509KeyPair(certPEM, keyPEM)
		if err != nil {
			return nil, err
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO instance_tls(id, root_cert_pem, root_key_pem, server_cert_pem, server_key_pem) VALUES(1, ?, ?, ?, ?) ON CONFLICT(id) DO UPDATE SET server_cert_pem = excluded.server_cert_pem, server_key_pem = excluded.server_key_pem`, rootPEM, rootKeyPEM, certPEM, keyPEM)
		if err != nil {
			return nil, err
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return &Material{certificate: pair, rootPEM: rootPEM, root: root}, nil
}

func addresses(names []string) ([]string, []net.IP, error) {
	if len(names) == 0 || len(names) > 32 {
		return nil, nil, errors.New("TLS: supply 1 to 32 server DNS names/IP addresses")
	}
	var dns []string
	var ips []net.IP
	seen := make(map[string]bool)
	for _, name := range names {
		name = strings.ToLower(strings.TrimSpace(name))
		if ip := net.ParseIP(name); ip != nil {
			name = ip.String()
			if !seen[name] {
				ips = append(ips, ip)
			}
		} else {
			if len(name) > 253 || !hostname.MatchString(name) {
				return nil, nil, errors.New("TLS: names must be DNS names/IP addresses without URLs, wildcards or ports")
			}
			if !seen[name] {
				dns = append(dns, name)
			}
		}
		seen[name] = true
	}
	return dns, ips, nil
}

func sameAddresses(leaf *x509.Certificate, dns []string, ips []net.IP) bool {
	if len(leaf.DNSNames) != len(dns) || len(leaf.IPAddresses) != len(ips) {
		return false
	}
	for _, name := range dns {
		found := false
		for _, old := range leaf.DNSNames {
			found = found || old == name
		}
		if !found {
			return false
		}
	}
	for _, ip := range ips {
		found := false
		for _, old := range leaf.IPAddresses {
			found = found || old.Equal(ip)
		}
		if !found {
			return false
		}
	}
	return true
}

func serial() (*big.Int, error) {
	// Nonzero, unpredictable serials also distinguish fresh database identities.
	n, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, err
	}
	return n.Add(n, big.NewInt(1)), nil
}

func encodeCertificate(template, parent *x509.Certificate, parentKey any) ([]byte, []byte, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, err
	}
	if parent == nil {
		parent = template
		parentKey = key
	}
	der, err := x509.CreateCertificate(rand.Reader, template, parent, &key.PublicKey, parentKey)
	if err != nil {
		return nil, nil, err
	}
	keyDER, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return nil, nil, err
	}
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER}), nil
}

func createRoot(now time.Time) ([]byte, []byte, error) {
	n, err := serial()
	if err != nil {
		return nil, nil, err
	}
	root := &x509.Certificate{SerialNumber: n, Subject: pkix.Name{CommonName: fmt.Sprintf("AlreadyOn private CA %x", n)}, NotBefore: now.Add(-5 * time.Minute), NotAfter: now.AddDate(10, 0, 0), IsCA: true, BasicConstraintsValid: true, MaxPathLenZero: true, KeyUsage: x509.KeyUsageCertSign | x509.KeyUsageCRLSign}
	return encodeCertificate(root, nil, nil)
}

func createLeaf(now time.Time, root *x509.Certificate, rootKey any, dns []string, ips []net.IP) ([]byte, []byte, error) {
	n, err := serial()
	if err != nil {
		return nil, nil, err
	}
	expires := now.AddDate(10, 0, 0)
	if expires.After(root.NotAfter) {
		expires = root.NotAfter
	}
	leaf := &x509.Certificate{SerialNumber: n, Subject: pkix.Name{CommonName: "AlreadyOn"}, NotBefore: now.Add(-5 * time.Minute), NotAfter: expires, BasicConstraintsValid: true, DNSNames: dns, IPAddresses: ips, KeyUsage: x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}}
	return encodeCertificate(leaf, root, rootKey)
}
