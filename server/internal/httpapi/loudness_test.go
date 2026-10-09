package httpapi

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLoudnessOptInAPI(t *testing.T) {
	router := NewRouter(t.TempDir(), t.TempDir(), openTestDB(t))
	for _, tc := range []struct {
		method, path, body string
		code               int
		want               string
	}{
		{"GET", "/api/settings/loudness", "", 200, `"enabled":false`},
		{"POST", "/api/settings/loudness/run", "", 409, "enable Loudness Detection"},
		{"PUT", "/api/settings/loudness", `{}`, 400, "enabled is required"},
		{"PUT", "/api/settings/loudness", `{"enabled":true}`, 200, `"enabled":true`},
		{"POST", "/api/settings/loudness/run", "", 200, `"enabled":true`},
		{"PUT", "/api/settings/loudness", `{"enabled":false}`, 200, `"enabled":false`},
	} {
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body)))
		if rec.Code != tc.code || !strings.Contains(rec.Body.String(), tc.want) {
			t.Fatalf("%s %s: %d %s", tc.method, tc.path, rec.Code, rec.Body.String())
		}
	}
}
