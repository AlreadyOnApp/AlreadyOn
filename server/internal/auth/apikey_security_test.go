package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"sync"
	"testing"
)

// Chosen keys need salted password hashes; generated random keys retain their
// indexed SHA-256 lookup. Upgrades preserve IDs so playback scopes stay valid.
func TestTestingKeysUseSaltedHashesAndUpgradeLegacyRecords(t *testing.T) {
	ctx := context.Background()
	db := openAPIKeyTestDB(t)
	const raw = "4826"
	sum := sha256.Sum256([]byte(raw))
	legacy := hex.EncodeToString(sum[:])
	var id int64
	if err := db.QueryRow(`INSERT INTO api_keys (user_id,name,key_hash,key_prefix) VALUES (?,'legacy',?,'test-key') RETURNING id`, ownerID(t, db), legacy).Scan(&id); err != nil {
		t.Fatal(err)
	}
	stored := func() string {
		t.Helper()
		var hash string
		if err := db.QueryRow(`SELECT key_hash FROM api_keys WHERE id=?`, id).Scan(&hash); err != nil {
			t.Fatal(err)
		}
		return hash
	}
	if _, _, err := VerifyAPIKey(ctx, db, "4827"); err == nil {
		t.Fatal("wrong legacy credential accepted")
	}
	if stored() != legacy {
		t.Fatal("wrong credential changed legacy hash")
	}
	if got, _, err := VerifyAPIKey(ctx, db, raw); err != nil || got != id {
		t.Fatalf("legacy key rejected: %d, %v", got, err)
	}
	first := stored()
	if !strings.HasPrefix(first, "$argon2id$") || !VerifyPassword(first, raw) {
		t.Fatal("legacy hash was not upgraded")
	}
	key, err := ProvisionTestAPIKey(ctx, db, "renamed", raw)
	if err != nil || key.ID != id {
		t.Fatalf("reprovision changed identity: %+v, %v", key, err)
	}
	second := stored()
	if first == second || !VerifyPassword(second, raw) {
		t.Fatal("reprovision did not use a fresh salt")
	}
	generated, generatedKey, err := GenerateAPIKey(ctx, db, ownerID(t, db), "generated")
	if err != nil {
		t.Fatal(err)
	}
	if got, _, err := VerifyAPIKey(ctx, db, generated); err != nil || got != generatedKey.ID {
		t.Fatalf("generated key rejected: %v", err)
	}
	if _, err := db.Exec(`UPDATE users SET disabled=1 WHERE id=?`, ownerID(t, db)); err != nil {
		t.Fatal(err)
	}
	if _, _, err := VerifyAPIKey(ctx, db, raw); err == nil {
		t.Fatal("disabled owner's test key accepted")
	}
}

func TestFreshTestingKeyNeverStoresFastDigest(t *testing.T) {
	ctx := context.Background()
	db := openAPIKeyTestDB(t)
	key, err := ProvisionTestAPIKey(ctx, db, "new testing key", "4826")
	if err != nil {
		t.Fatal(err)
	}
	var hash string
	if err := db.QueryRow(`SELECT key_hash FROM api_keys WHERE id=?`, key.ID).Scan(&hash); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(hash, "$argon2id$") || !VerifyPassword(hash, "4826") {
		t.Fatal("new testing key must use Argon2id")
	}
}

// Random salts must not create duplicate credentials when local provisioning
// races; the lookup/save transaction preserves the original idempotence.
func TestConcurrentTestingKeyProvisioningKeepsOneRecord(t *testing.T) {
	db := openAPIKeyTestDB(t)
	var wg sync.WaitGroup
	ids := make(chan int64, 4)
	for range 4 {
		wg.Go(func() {
			key, err := ProvisionTestAPIKey(context.Background(), db, "same key", "4826")
			if err != nil {
				t.Errorf("provision: %v", err)
				return
			}
			ids <- key.ID
		})
	}
	wg.Wait()
	close(ids)
	var first int64
	for id := range ids {
		if first == 0 {
			first = id
		}
		if id != first {
			t.Errorf("duplicate key id: %d, want %d", id, first)
		}
	}
	var count int
	if err := db.QueryRow(`SELECT count(*) FROM api_keys WHERE key_prefix='test-key'`).Scan(&count); err != nil || count != 1 {
		t.Fatalf("records=%d, err=%v", count, err)
	}
}
