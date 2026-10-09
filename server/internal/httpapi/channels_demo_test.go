package httpapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
)

func demoNumbers(t *testing.T, router http.Handler) map[int]bool {
	t.Helper()
	got := map[int]bool{}
	for _, c := range listChannelsFor(t, router) {
		if c.Number >= 196 && c.Number <= 199 {
			got[c.Number] = true
		}
	}
	return got
}

// The demo channels come and go with the real channels' visibility right
// away, not at the next refresh.
func TestDemoChannels_FollowRealChannelVisibilityImmediately(t *testing.T) {
	db := openTestDB(t)
	router := NewRouter(t.TempDir(), t.TempDir(), db)
	addItem(t, db, "movie", "Movie One", "", 2001, `["comedy"]`)
	addItem(t, db, "movie", "Movie Two", "", 2002, `["comedy"]`)

	// A real channel with content that is hidden: the demo channels stand in.
	rec := doJSON(t, router, http.MethodPost, "/api/channels", `{"number":200,"name":"Real","filter":{"audienceTier":"adult","genres":[]}}`)
	id := decodeChannel(t, rec.Body.Bytes()).ID
	doJSON(t, router, http.MethodPut, fmt.Sprintf("/api/channels/%d/visibility", id), `{"visibility":"hide"}`)
	if !demoNumbers(t, router)[198] {
		t.Fatalf("demo channels = %v, want the movie demo channel while every real channel is hidden", demoNumbers(t, router))
	}

	// Showing the real channel removes them at once.
	doJSON(t, router, http.MethodPut, fmt.Sprintf("/api/channels/%d/visibility", id), `{"visibility":"show"}`)
	if got := demoNumbers(t, router); len(got) != 0 {
		t.Errorf("demo channels = %v after showing a real channel, want none", got)
	}

	// Bulk hide brings them back; the guide shows them too.
	doJSON(t, router, http.MethodPost, "/api/channels/bulk-visibility", fmt.Sprintf(`{"ids":[%d],"visibility":"hide"}`, id))
	if !demoNumbers(t, router)[198] {
		t.Errorf("demo channels missing after hiding the only real channel")
	}
	doJSON(t, router, http.MethodDelete, fmt.Sprintf("/api/channels/%d", id), "")
	if !demoNumbers(t, router)[198] {
		t.Errorf("demo channels missing after deleting the only real channel")
	}
}

func TestDemoChannels_DeletingADemoChannelDoesNotRecreateIt(t *testing.T) {
	db := openTestDB(t)
	router := NewRouter(t.TempDir(), t.TempDir(), db)
	addItem(t, db, "movie", "Movie One", "", 2001, "")
	addItem(t, db, "movie", "Movie Two", "", 2002, "")
	// Any action that syncs creates the demo channels when nothing real is visible.
	doJSON(t, router, http.MethodPost, "/api/channels/bulk-visibility", `{"ids":[],"visibility":"auto"}`)
	var demoID int64
	if err := db.QueryRow(`SELECT id FROM channels WHERE number = 198`).Scan(&demoID); err != nil {
		t.Fatalf("demo channel missing: %v", err)
	}
	doJSON(t, router, http.MethodDelete, fmt.Sprintf("/api/channels/%d", demoID), "")
	if demoNumbers(t, router)[198] {
		t.Error("deleting a demo channel recreated it")
	}
}

func listChannelsFor(t *testing.T, router http.Handler) []channelResponse {
	t.Helper()
	rec := doJSON(t, router, http.MethodGet, "/api/channels", "")
	var list []channelResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil {
		t.Fatalf("decode channels: %v", err)
	}
	return list
}
