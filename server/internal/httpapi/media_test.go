package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMediaSettingsAPI_GetDefaultThenPutRoundTrips(t *testing.T) {
	router := NewRouter(t.TempDir(), t.TempDir(), openTestDB(t))

	req := httptest.NewRequest(http.MethodGet, "/api/settings/media", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("get status = %d, want %d", rec.Code, http.StatusOK)
	}
	var got mediaSettingsResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode get response: %v", err)
	}
	if got.PollIntervalMinutes != 60 {
		t.Errorf("default PollIntervalMinutes = %d, want 60", got.PollIntervalMinutes)
	}

	putBody, _ := json.Marshal(map[string]any{"pollIntervalMinutes": 240})
	req = httptest.NewRequest(http.MethodPut, "/api/settings/media", bytes.NewReader(putBody))
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("put status = %d, want %d, body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/settings/media", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode get-after-put response: %v", err)
	}
	if got.PollIntervalMinutes != 240 {
		t.Errorf("PollIntervalMinutes after PUT = %d, want 240", got.PollIntervalMinutes)
	}
}

func TestMediaSettingsAPI_PutUnlistedInterval_ReturnsBadRequest(t *testing.T) {
	router := NewRouter(t.TempDir(), t.TempDir(), openTestDB(t))

	putBody, _ := json.Marshal(map[string]any{"pollIntervalMinutes": 45})
	req := httptest.NewRequest(http.MethodPut, "/api/settings/media", bytes.NewReader(putBody))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d, body=%s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
}
