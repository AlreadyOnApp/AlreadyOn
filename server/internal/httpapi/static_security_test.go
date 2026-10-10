package httpapi

import (
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

// Test the handler directly as well as its normal ServeMux path cleaning.
func TestSPAFileServerConfinesFilesAndKeepsCachePolicy(t *testing.T) {
	dir := writeTestWebDir(t)
	outside := filepath.Join(t.TempDir(), "secret.js")
	if err := os.WriteFile(outside, []byte("outside secret"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(dir, "assets"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "assets", "app-hash.js"), []byte("built asset"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(dir, "assets", "escape.js")); err != nil {
		t.Fatal(err)
	}
	relative, err := filepath.Rel(filepath.Join(dir, "assets"), outside)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(relative, filepath.Join(dir, "assets", "relative-escape.js")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("app.js", filepath.Join(dir, "linked.js")); err != nil {
		t.Fatal(err)
	}
	h := newSPAFileServer(dir)
	for _, tc := range []struct {
		path, body, cache string
		status            int
	}{
		{"/assets/app-hash.js", "built asset", "public, max-age=31536000, immutable", 200},
		{"/app.js", "console.log('hi')", "no-cache", 200},
		{"/linked.js", "console.log('hi')", "no-cache", 200},
		{"/assets/escape.js", "<html>app shell</html>", "no-cache", 200},
		{"/assets/relative-escape.js", "<html>app shell</html>", "no-cache", 200},
		{"/admin/logs", "<html>app shell</html>", "no-cache", 200},
		{"/admin/logs/", "<html>app shell</html>", "no-cache", 200},
		{"/", "<html>app shell</html>", "no-cache", 200},
		{"/../secret.js", "404 page not found\n", "", 404},
		{"/assets/../../secret.js", "404 page not found\n", "", 404},
		{"/assets/%2e%2e/%2e%2e/secret.js", "404 page not found\n", "", 404},
	} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest("GET", tc.path, nil))
		if w.Code != tc.status || w.Body.String() != tc.body || w.Header().Get("Cache-Control") != tc.cache {
			t.Errorf("%s: status=%d, body=%q, cache=%q", tc.path, w.Code, w.Body.String(), w.Header().Get("Cache-Control"))
		}
	}
}
