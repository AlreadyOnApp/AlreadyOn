package httpapi

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// A direct file can take far longer than the idle limit to deliver while the
// browser keeps reading. Only a stalled client may be cut off.
func TestDirectFileIdleDeadlineAllowsLongSlowReads(t *testing.T) {
	old := mediaWriteIdle
	mediaWriteIdle = 300 * time.Millisecond
	defer func() { mediaWriteIdle = old }()
	payload := bytes.Repeat([]byte("0123456789abcdef"), 2<<20) // 32 MiB exceeds socket buffering
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeContent(&idleWriter{w, http.NewResponseController(w)}, r, "program.mp4", time.Time{}, bytes.NewReader(payload))
	}))
	defer server.Close()
	res, err := http.Get(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	start, received := time.Now(), 0
	buf := make([]byte, 1<<20)
	for {
		n, err := io.ReadFull(res.Body, buf)
		received += n
		if err != nil {
			break
		}
		time.Sleep(40 * time.Millisecond) // steady reads, each well inside the idle limit
	}
	if received != len(payload) {
		t.Fatalf("received %d of %d bytes after %v", received, len(payload), time.Since(start))
	}
	if time.Since(start) < 3*mediaWriteIdle {
		t.Fatal("transfer finished too quickly to exercise the deadline")
	}
}
