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

func TestChannelsAPI_CreateListGetUpdateDelete(t *testing.T) {
	router := NewRouter(t.TempDir(), t.TempDir(), openTestDB(t))

	// Create
	createBody, _ := json.Marshal(map[string]any{"number": 1000, "name": "AlreadyOn Movies"})
	req := httptest.NewRequest(http.MethodPost, "/api/channels", bytes.NewReader(createBody))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create status = %d, want %d, body=%s", rec.Code, http.StatusCreated, rec.Body.String())
	}
	var created channelResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	if created.ID == 0 || created.Number != 1000 || created.Name != "AlreadyOn Movies" {
		t.Errorf("created = %+v", created)
	}

	// List
	req = httptest.NewRequest(http.MethodGet, "/api/channels", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("list status = %d, want %d", rec.Code, http.StatusOK)
	}
	var list []channelResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil {
		t.Fatalf("decode list response: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("len(list) = %d, want 1", len(list))
	}

	// Get
	req = httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/channels/%d", created.ID), nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("get status = %d, want %d", rec.Code, http.StatusOK)
	}

	// Update
	updateBody, _ := json.Marshal(map[string]any{"number": 1001, "name": "Renamed"})
	req = httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/channels/%d", created.ID), bytes.NewReader(updateBody))
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("update status = %d, want %d, body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}
	var updated channelResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &updated); err != nil {
		t.Fatalf("decode update response: %v", err)
	}
	if updated.Number != 1001 || updated.Name != "Renamed" {
		t.Errorf("updated = %+v", updated)
	}

	// Delete
	req = httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/channels/%d", created.ID), nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("delete status = %d, want %d", rec.Code, http.StatusNoContent)
	}

	// Get after delete -> 404
	req = httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/channels/%d", created.ID), nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("get-after-delete status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestChannelsAPI_Create_InvalidInput_Returns400(t *testing.T) {
	router := NewRouter(t.TempDir(), t.TempDir(), openTestDB(t))

	body, _ := json.Marshal(map[string]any{"number": 0, "name": ""})
	req := httptest.NewRequest(http.MethodPost, "/api/channels", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
}

func TestChannelsAPI_Create_DuplicateNumber_Returns400(t *testing.T) {
	router := NewRouter(t.TempDir(), t.TempDir(), openTestDB(t))

	body, _ := json.Marshal(map[string]any{"number": 1000, "name": "First"})
	req := httptest.NewRequest(http.MethodPost, "/api/channels", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("first create status = %d, want %d", rec.Code, http.StatusCreated)
	}

	body2, _ := json.Marshal(map[string]any{"number": 1000, "name": "Second"})
	req = httptest.NewRequest(http.MethodPost, "/api/channels", bytes.NewReader(body2))
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("second create status = %d, want %d, body=%s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
}

func TestChannelsAPI_Get_UnknownID_Returns404(t *testing.T) {
	router := NewRouter(t.TempDir(), t.TempDir(), openTestDB(t))

	req := httptest.NewRequest(http.MethodGet, "/api/channels/999", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestChannelsAPI_SetVisibility(t *testing.T) {
	db := openTestDB(t)
	channelID := insertTestChannelWithContent(t, db, 1000, "Test Channel")
	router := NewRouter(t.TempDir(), t.TempDir(), db)
	put := func(id int64, body string) (*httptest.ResponseRecorder, channelResponse) {
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/channels/%d/visibility", id), strings.NewReader(body)))
		var got channelResponse
		_ = json.Unmarshal(rec.Body.Bytes(), &got)
		return rec, got
	}

	if rec, got := put(channelID, `{"visibility":"hide"}`); rec.Code != http.StatusOK || got.Visibility != "hide" || !got.Hidden {
		t.Fatalf("hide: %d %+v", rec.Code, got)
	}
	if _, got := put(channelID, `{"visibility":"show"}`); got.Visibility != "show" || got.Hidden {
		t.Errorf("show: %+v", got)
	}
	if _, got := put(channelID, `{"visibility":"auto"}`); got.Visibility != "auto" {
		t.Errorf("auto: %+v", got)
	}
	if rec, _ := put(channelID, `{"visibility":"sometimes"}`); rec.Code != http.StatusBadRequest {
		t.Errorf("invalid mode: %d", rec.Code)
	}
	if rec, _ := put(999, `{"visibility":"hide"}`); rec.Code != http.StatusNotFound {
		t.Errorf("unknown channel: %d", rec.Code)
	}
}

func TestChannelsAPI_BulkSetVisibility(t *testing.T) {
	db := openTestDB(t)
	a := insertTestChannelWithContent(t, db, 1000, "Channel A")
	b := insertTestChannelWithContent(t, db, 1001, "Channel B")
	router := NewRouter(t.TempDir(), t.TempDir(), db)

	body, _ := json.Marshal(map[string]any{"ids": []int64{a, b}, "visibility": "hide"})
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/channels/bulk-visibility", bytes.NewReader(body)))
	var resp map[string]int
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil || rec.Code != http.StatusOK || resp["updated"] != 2 {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	for _, id := range []int64{a, b} {
		rec = httptest.NewRecorder()
		router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/channels/%d", id), nil))
		var got channelResponse
		_ = json.Unmarshal(rec.Body.Bytes(), &got)
		if !got.Hidden || got.Visibility != "hide" {
			t.Errorf("channel %d after bulk hide: %+v", id, got)
		}
	}
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/channels/bulk-visibility", strings.NewReader(`{"ids":[1],"visibility":"nope"}`)))
	if rec.Code != http.StatusBadRequest {
		t.Errorf("invalid mode: %d", rec.Code)
	}
}

func TestChannelsAPI_AutoHidesEmptyChannelsOnly(t *testing.T) {
	db := openTestDB(t)
	empty := insertTestChannelWithContent(t, db, 1000, "Empty Channel")
	a := insertTestMediaFile(t, db, "/tv/show-a.mkv", 600)
	b := insertTestMediaFile(t, db, "/tv/show-b.mkv", 600)
	wellStocked := insertTestChannelWithContent(t, db, 1001, "Two-Show Channel", a, b)
	router := NewRouter(t.TempDir(), t.TempDir(), db)

	body, _ := json.Marshal(map[string]any{"ids": []int64{empty, wellStocked}, "visibility": "auto"})
	router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/api/channels/bulk-visibility", bytes.NewReader(body)))
	hidden := func(id int64) bool {
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/channels/%d", id), nil))
		var got channelResponse
		_ = json.Unmarshal(rec.Body.Bytes(), &got)
		return got.Hidden
	}
	if !hidden(empty) || hidden(wellStocked) {
		t.Errorf("auto: empty hidden=%v (want true), stocked hidden=%v (want false)", hidden(empty), hidden(wellStocked))
	}
}
