// These isolated fixtures verify TV pairing permissions and brute-force limits.
package httpapi

import (
	"crypto/tls"
	"github.com/lynchbros/alreadyon/server/internal/auth"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestTVPairingAccountIsolationAndLocalBoundary(t *testing.T) {
	db := openTestDB(t)
	owner, err := auth.CreateAccount(t.Context(), db, "owner", "synthetic-owner-pass")
	if err != nil {
		t.Fatal(err)
	}
	viewer, err := auth.CreateUser(t.Context(), db, auth.NewUser{Username: "viewer", Password: "synthetic-viewer-pass", Role: auth.RoleViewer})
	if err != nil {
		t.Fatal(err)
	}
	if err := auth.SetTVPIN(t.Context(), db, owner.ID, "123456"); err != nil {
		t.Fatal(err)
	}
	if err := auth.SetTVPIN(t.Context(), db, viewer.ID, "654321"); err != nil {
		t.Fatal(err)
	}
	handler := auth.RequireAuth(db, NewRouter(t.TempDir(), t.TempDir(), db))
	request := func(pin, peer string, secure bool) *httptest.ResponseRecorder {
		r := httptest.NewRequest("POST", "/api/auth/tv-login", strings.NewReader(`{"pin":"`+pin+`"}`))
		r.RemoteAddr = peer
		r.Header.Set("X-Forwarded-For", "192.168.1.2")
		if strings.HasPrefix(peer, "192.168.") {
			r.Header.Del("X-Forwarded-For")
		}
		if secure {
			r.TLS = &tls.ConnectionState{}
		}
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		return w
	}
	for _, boundary := range []struct {
		peer   string
		secure bool
	}{{"8.8.8.8:1234", true}, {"192.168.1.2:1234", false}} {
		if w := request("654321", boundary.peer, boundary.secure); w.Code != 403 {
			t.Fatalf("boundary: %d %s", w.Code, w.Body.String())
		}
	}
	forwarded := httptest.NewRequest("POST", "/api/auth/tv-login", strings.NewReader(`{"pin":"654321"}`))
	forwarded.TLS = &tls.ConnectionState{}
	forwarded.RemoteAddr = "192.168.1.2:1234"
	forwarded.Header.Set("X-Forwarded-For", "8.8.8.8")
	rejected := httptest.NewRecorder()
	handler.ServeHTTP(rejected, forwarded)
	if rejected.Code != 403 {
		t.Fatal("proxied internet peer allowed to pair")
	}
	w := request("654321", "192.168.1.2:1234", true)
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"username":"viewer"`) {
		t.Fatalf("viewer pairing: %d %s", w.Code, w.Body.String())
	}
	cookies := w.Result().Cookies()
	if len(cookies) == 0 || !cookies[0].Secure {
		t.Fatal("missing secure TV session")
	}
	r := httptest.NewRequest("GET", "/api/auth/tv-pin", nil)
	r.AddCookie(cookies[0])
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, r)
	if rec.Code != 200 || strings.Contains(rec.Body.String(), "654321") {
		t.Fatalf("own status: %d %s", rec.Code, rec.Body.String())
	}
	r = httptest.NewRequest("DELETE", "/api/auth/tv-pin", nil)
	r.AddCookie(cookies[0])
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, r)
	if rec.Code != 204 {
		t.Fatalf("delete: %d", rec.Code)
	}
	if enabled, _ := auth.HasTVPIN(t.Context(), db, owner.ID); !enabled {
		t.Fatal("viewer removed owner's PIN")
	}
	if w := request("654321", "192.168.1.2:1234", true); w.Code != 401 {
		t.Fatal("removed PIN still pairs")
	}
}

func TestTVPINValidationUniquenessDisabledAndThrottling(t *testing.T) {
	db := openTestDB(t)
	owner, _ := auth.CreateAccount(t.Context(), db, "owner", "synthetic-owner-pass")
	viewer, _ := auth.CreateUser(t.Context(), db, auth.NewUser{Username: "viewer", Password: "synthetic-viewer-pass", Role: auth.RoleViewer})
	for _, pin := range []string{"12345", "1234567890123", "12345a", "１２３４５６"} {
		if err := auth.SetTVPIN(t.Context(), db, owner.ID, pin); err != auth.ErrInvalidTVPIN {
			t.Fatalf("invalid PIN accepted: %v", err)
		}
	}
	if err := auth.SetTVPIN(t.Context(), db, owner.ID, "001234"); err != nil {
		t.Fatal(err)
	}
	if err := auth.SetTVPIN(t.Context(), db, viewer.ID, "001234"); err != auth.ErrTVPINTaken {
		t.Fatalf("duplicate PIN: %v", err)
	}
	if _, ok, err := auth.AuthenticateTVPIN(t.Context(), db, "001234"); err != nil || !ok {
		t.Fatalf("leading zeros: %v %v", ok, err)
	}
	_, _ = db.Exec(`UPDATE users SET must_change_password = 1 WHERE id = ?`, owner.ID)
	if _, ok, _ := auth.AuthenticateTVPIN(t.Context(), db, "001234"); ok {
		t.Fatal("forced password change bypassed")
	}
	_, _ = db.Exec(`UPDATE users SET disabled = 1, must_change_password = 0 WHERE id = ?`, owner.ID)
	if _, ok, _ := auth.AuthenticateTVPIN(t.Context(), db, "001234"); ok {
		t.Fatal("disabled account paired")
	}
	handler := auth.RequireAuth(db, NewRouter(t.TempDir(), t.TempDir(), db))
	for i := 0; i < 6; i++ {
		r := httptest.NewRequest("POST", "/api/auth/tv-login", strings.NewReader(`{"pin":"999999"}`))
		r.TLS = &tls.ConnectionState{}
		r.RemoteAddr = "192.168.1.4:1234"
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		expected := 401
		if i == 5 {
			expected = 429
		}
		if w.Code != expected {
			t.Fatalf("attempt %d: %d", i, w.Code)
		}
	}
}
