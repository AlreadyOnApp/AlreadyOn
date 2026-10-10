package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/alreadyonapp/alreadyon/server/internal/auth"
	"github.com/alreadyonapp/alreadyon/server/internal/playback"
)

// Exercise deletion only against disposable files, including rejected boundaries.
func TestDeleteCommercial(t *testing.T) {
	for _, mode := range []string{"delete", "unauthenticated", "stale", "program", "outside", "symlink", "readonly"} {
		t.Run(mode, func(t *testing.T) {
			db := openTestDB(t)
			root, outside := t.TempDir(), t.TempDir()
			path := filepath.Join(root, "ad.mp4")
			if mode == "outside" || mode == "symlink" {
				path = filepath.Join(outside, "ad.mp4")
			}
			if err := os.WriteFile(path, []byte("synthetic commercial"), 0600); err != nil {
				t.Fatal(err)
			}
			catalogPath := path
			if mode == "symlink" {
				catalogPath = filepath.Join(root, "escape.mp4")
				if err := os.Symlink(path, catalogPath); err != nil {
					t.Fatal(err)
				}
			}
			now := time.Now().Truncate(time.Second)
			starts := now.Add(-10 * time.Second)
			kind := "commercial"
			if mode == "program" {
				kind = "program"
			}
			roots, _ := json.Marshal([]string{root})
			for _, q := range []struct {
				sql  string
				args []any
			}{
				{`UPDATE folder_settings SET commercial_paths=?,tv_paths='[]',movie_paths='[]'`, []any{string(roots)}},
				{`INSERT INTO channels(id,number,name) VALUES(999,999,'test')`, nil},
				{`INSERT INTO media_files(id,root_kind,path,size_bytes,modified_unix,first_seen_unix,last_seen_unix,duration_seconds,status) VALUES(999,'commercial',?,1,0,0,0,60,'ok')`, []any{catalogPath}},
				{`INSERT INTO schedule_entries(channel_id,media_file_id,kind,source_offset_seconds,duration_seconds,starts_at_unix,ends_at_unix) VALUES(999,999,?,0,60,?,?)`, []any{kind, starts.Unix(), starts.Add(time.Minute).Unix()}},
			} {
				if _, err := db.Exec(q.sql, q.args...); err != nil {
					t.Fatal(err)
				}
			}
			if mode == "readonly" {
				if os.Geteuid() == 0 {
					t.Skip("requires unprivileged filesystem permissions")
				}
				if err := os.Chmod(root, 0500); err != nil {
					t.Fatal(err)
				}
				defer os.Chmod(root, 0700)
			}
			key, _, err := auth.GenerateAPIKey(context.Background(), db, testOwnerID(t, db), "synthetic")
			if err != nil {
				t.Fatal(err)
			}
			clock := &httpClock{now}
			p := NewPlaybackHTTP(db, playback.NewSessions(clock), nil)
			p.Clock = clock
			if mode == "stale" {
				starts = starts.Add(-time.Minute)
			}
			body := fmt.Sprintf(`{"channelId":999,"mediaFileId":999,"entryStartsAt":%q}`, starts.Format(time.RFC3339))
			handler := auth.RequireAuth(db, NewRouter(t.TempDir(), t.TempDir(), db, p))
			request := httptest.NewRequest("DELETE", "/api/testing/commercial", strings.NewReader(body))
			if mode != "unauthenticated" {
				request.Header.Set("Authorization", "Bearer "+key)
			}
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			want := map[string]int{"delete": 204, "unauthenticated": 401, "stale": 409, "program": 409, "outside": 403, "symlink": 403, "readonly": 500}[mode]
			if response.Code != want {
				t.Fatalf("status %d: %s", response.Code, response.Body.String())
			}
			_, statErr := os.Stat(path)
			var status string
			if err := db.QueryRow(`SELECT status FROM media_files WHERE id=999`).Scan(&status); err != nil {
				t.Fatal(err)
			}
			if mode == "delete" {
				if !os.IsNotExist(statErr) || status != "missing" {
					t.Fatalf("file=%v status=%s", statErr, status)
				}
				response = httptest.NewRecorder()
				handler.ServeHTTP(response, request.Clone(request.Context()))
				if response.Code == 204 {
					t.Fatal("duplicate deletion succeeded")
				}
			} else if statErr != nil || status != "ok" {
				t.Fatalf("rejected removal modified file/catalog: %v %s", statErr, status)
			}
		})
	}
}

func TestHostIPv4(t *testing.T) {
	for address, want := range map[string]bool{"192.168.1.210": true, "10.0.0.2": true, "172.17.0.2": true, "172.16.0.2": true, "fd00:1234::1": false, "2606:4700::30": false, "127.0.0.1": true, "169.254.1.1": true} {
		if got := hostIPv4(net.ParseIP(address)); got != want {
			t.Errorf("%s: %v, want %v", address, got, want)
		}
	}
}
