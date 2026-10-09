package httpapi

import (
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"strings"
	"testing"
)

// Swagger UI sends Try It Out calls to the spec's server URL. A named address
// would turn them into cross-origin requests whenever the docs are opened by
// any other name, and the CSRF check would refuse them; with none, they go to
// the page's own address.
func TestOpenAPIServersAreRelative(t *testing.T) {
	spec, err := os.ReadFile("../../../api/openapi.yaml")
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range regexp.MustCompile(`(?m)^\s+- url: (\S+)`).FindAllStringSubmatch(string(spec), -1) {
		// Only the media listener is named, and it is rewritten per request.
		if m[1] != specMediaServer {
			t.Errorf("servers url %q: the management API must have no server entry (calls go to the docs page's own address)", m[1])
		}
	}
}

func TestOpenAPISpecPointsMediaCallsAtTheHostTheDocsWereOpenedBy(t *testing.T) {
	old := openAPISpecPath
	openAPISpecPath = "../../../api/openapi.yaml"
	defer func() { openAPISpecPath = old }()
	db := openTestDB(t)
	p := NewPlaybackHTTP(db, nil, nil)
	router := NewRouter(t.TempDir(), t.TempDir(), db, p)
	for _, host := range []string{"127.0.0.1:19900", "localhost:19900", "media-box:19900", "tv.example.com"} {
		r := httptest.NewRequest(http.MethodGet, "/api/openapi.yaml", nil)
		r.Host = host
		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)
		name := strings.Split(host, ":")[0]
		if w.Code != 200 || strings.Contains(w.Body.String(), specMediaServer) && name != "localhost" || !strings.Contains(w.Body.String(), "http://"+host) {
			t.Errorf("%s: %d media server not rewritten", host, w.Code)
		}
	}
}
