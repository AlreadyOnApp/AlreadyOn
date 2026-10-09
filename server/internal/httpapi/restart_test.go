package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRestartHandlerAnswersThenRestarts(t *testing.T) {
	calls := 0
	rec := httptest.NewRecorder()
	RestartHandler(func() { calls++ })(rec, httptest.NewRequest(http.MethodPost, "/api/server/restart", nil))
	if rec.Code != http.StatusAccepted || !strings.Contains(rec.Body.String(), `"restarting"`) || calls != 1 {
		t.Fatalf("code %d body %s restarts %d", rec.Code, rec.Body.String(), calls)
	}
}
