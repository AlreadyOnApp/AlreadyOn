package httpapi

import (
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestServePlaylistTokensLadderEntries(t *testing.T) {
	dir := t.TempDir()
	master := "#EXTM3U\n#EXT-X-STREAM-INF:BANDWIDTH=1700000,RESOLUTION=640x360\nstream-0.m3u8\n#EXT-X-STREAM-INF:BANDWIDTH=5200000,RESOLUTION=1280x720\nstream-1.m3u8\n"
	variant := "#EXTM3U\n#EXTINF:2.0,\nsegment-1-000000007.ts\n"
	single := "#EXTM3U\n#EXTINF:2.0,\nsegment-000000007.ts\n"
	for name, body := range map[string]string{"index.m3u8": master, "stream-1.m3u8": variant, "single.m3u8": single} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	for file, want := range map[string][]string{
		"index.m3u8":    {"stream-0.m3u8?token=abc", "stream-1.m3u8?token=abc"},
		"stream-1.m3u8": {"segment-1-000000007.ts?token=abc"},
		"single.m3u8":   {"segment-000000007.ts?token=abc"},
	} {
		rec := httptest.NewRecorder()
		servePlaylist(rec, httptest.NewRequest("GET", "/media/1/x?token=abc", nil), dir, file)
		body := rec.Body.String()
		if rec.Code != 200 {
			t.Fatalf("%s: status %d %s", file, rec.Code, body)
		}
		for _, w := range want {
			if !strings.Contains(body, w) {
				t.Errorf("%s lacks %q:\n%s", file, w, body)
			}
		}
	}
	// A playlist may only name files this server produced.
	if err := os.WriteFile(filepath.Join(dir, "bad.m3u8"), []byte("#EXTM3U\n../../etc/passwd\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	servePlaylist(rec, httptest.NewRequest("GET", "/media/1/x?token=abc", nil), dir, "bad.m3u8")
	if rec.Code != 500 {
		t.Fatalf("a playlist naming other files was served: %d", rec.Code)
	}
	for name, ok := range map[string]bool{"stream-0.m3u8": true, "stream-12.m3u8": false, "index.m3u8": false, "segment-1-000000001.ts": true, "segment-000000001.ts": true, "segment-x-000000001.ts": false, "../stream-0.m3u8": false} {
		if variantName.MatchString(name) != (ok && strings.HasPrefix(name, "stream-")) || segmentName.MatchString(name) != (ok && strings.HasPrefix(name, "segment-")) {
			t.Errorf("name %q classified wrongly", name)
		}
	}
}
