package httpapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAPIKeysAPI_CreateReturnsRawKeyOnce_ListNeverDoes(t *testing.T) {
	db := openTestDB(t)
	testOwnerID(t, db)
	router := NewRouter(t.TempDir(), t.TempDir(), db)

	body, _ := json.Marshal(map[string]string{"name": "Living room TV"})
	req := httptest.NewRequest(http.MethodPost, "/api/admin/api-keys", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create status = %d, want %d, body=%s", rec.Code, http.StatusCreated, rec.Body.String())
	}
	var created apiKeyCreateResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	if created.ID == 0 || created.Name != "Living room TV" {
		t.Errorf("created = %+v", created)
	}
	if !strings.HasPrefix(created.Key, "ao_") {
		t.Errorf("created.Key = %q, want an ao_-prefixed raw key", created.Key)
	}
	if created.KeyPrefix == "" || !strings.HasPrefix(created.Key, created.KeyPrefix) {
		t.Errorf("created.KeyPrefix = %q, want a prefix of created.Key (%q)", created.KeyPrefix, created.Key)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/admin/api-keys", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("list status = %d, want %d", rec.Code, http.StatusOK)
	}
	if bytes.Contains(rec.Body.Bytes(), []byte(created.Key)) {
		t.Errorf("GET list response contained the raw key: %s", rec.Body.String())
	}
	var list []apiKeyResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil {
		t.Fatalf("decode list response: %v", err)
	}
	if len(list) != 1 || list[0].ID != created.ID {
		t.Fatalf("list = %+v, want one entry matching the created key", list)
	}
}

func TestAPIKeysAPI_Create_EmptyName_Returns400(t *testing.T) {
	db := openTestDB(t)
	testOwnerID(t, db)
	router := NewRouter(t.TempDir(), t.TempDir(), db)

	body, _ := json.Marshal(map[string]string{"name": ""})
	req := httptest.NewRequest(http.MethodPost, "/api/admin/api-keys", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
}

func TestAPIKeysAPI_Revoke_RemovesItFromTheList(t *testing.T) {
	db := openTestDB(t)
	testOwnerID(t, db)
	router := NewRouter(t.TempDir(), t.TempDir(), db)

	body, _ := json.Marshal(map[string]string{"name": "Temporary"})
	req := httptest.NewRequest(http.MethodPost, "/api/admin/api-keys", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	var created apiKeyCreateResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode create response: %v", err)
	}

	req = httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/admin/api-keys/%d", created.ID), nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("revoke status = %d, want %d, body=%s", rec.Code, http.StatusNoContent, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/admin/api-keys", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	var list []apiKeyResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil {
		t.Fatalf("decode list response: %v", err)
	}
	if len(list) != 0 {
		t.Errorf("list = %+v, want empty after revoking the only key", list)
	}
}

func TestAPIKeysAPI_Revoke_UnknownID_Returns404(t *testing.T) {
	db := openTestDB(t)
	testOwnerID(t, db)
	router := NewRouter(t.TempDir(), t.TempDir(), db)

	req := httptest.NewRequest(http.MethodDelete, "/api/admin/api-keys/999", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusNotFound, rec.Body.String())
	}
}
