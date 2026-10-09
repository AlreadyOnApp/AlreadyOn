package governor

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"os/exec"
	"sync"
	"sync/atomic"
	"time"
)

var defaultPool atomic.Pointer[Governor]

// SetDefault connects cooperative Go work and outbound metadata to the same
// governor as subprocesses. Tests without a production pool remain isolated.
func SetDefault(g *Governor) { defaultPool.Store(g) }

// ReserveForeground pauses the production background pool until release. A
// process without a configured pool gets a no-op release for isolated tests.
func ReserveForeground() func() {
	if g := defaultPool.Load(); g != nil {
		return g.ReserveForeground()
	}
	return func() {}
}

// Checkpoint yields between bounded units of background Go/SQLite work. Never
// call while holding a database transaction or another worker's required lock.
// Control-plane health/metrics and foreground playback do not use this gate.
func Checkpoint(ctx context.Context) error {
	g := defaultPool.Load()
	if g == nil {
		return ctx.Err()
	}
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		g.mu.Lock()
		ready, wake := g.target > 0, g.changed
		g.mu.Unlock()
		if ready {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-wake:
		}
	}
}

// BackgroundOutput enrolls calibration commands too; no benchmark is allowed
// to bypass the shared budget. Timings include throttling and are conservative.
func BackgroundOutput(ctx context.Context, cmd *exec.Cmd) ([]byte, error) {
	if g := defaultPool.Load(); g != nil && TicketFrom(ctx) == nil {
		t, err := g.Acquire(ctx, PriorityLow)
		if err != nil {
			return nil, err
		}
		defer t.Release()
		t.SetTask("calibration")
		ctx = WithTicket(ctx, t)
	}
	var output bytes.Buffer
	cmd.Stdout, cmd.Stderr = &output, &output
	err := RunCommand(ctx, cmd)
	return output.Bytes(), err
}

// One byte scheduler is shared by all background HTTP bodies (receive + send).
// Mbps is decimal megabits: 750 Mbps = 93,750,000 bytes per second. Reservations
// precede reads, so concurrent requests cannot each consume their own allowance.
const NetworkBytesPerSecond = 750_000_000 / 8

var networkPacer bytePacer

type bytePacer struct {
	mu   sync.Mutex
	next time.Time
}

func (p *bytePacer) wait(ctx context.Context, n int) error {
	p.mu.Lock()
	now := time.Now()
	start := p.next
	if start.Before(now) {
		start = now
	}
	p.next = start.Add(time.Duration(float64(n) / NetworkBytesPerSecond * float64(time.Second)))
	until := p.next
	p.mu.Unlock()
	timer := time.NewTimer(time.Until(until))
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

type pacedBody struct {
	io.ReadCloser
	ctx context.Context
}

func (b *pacedBody) Read(p []byte) (int, error) {
	if err := Checkpoint(b.ctx); err != nil {
		return 0, err
	}
	p = p[:min(len(p), 32*1024)]
	if err := networkPacer.wait(b.ctx, len(p)); err != nil {
		return 0, err
	}
	return b.ReadCloser.Read(p)
}

// BackgroundTransport throttles metadata transfers without throttling streaming
// responses. HTTP/TLS headers and kernel buffering are outside body pacing.
type BackgroundTransport struct{ Base http.RoundTripper }

func (t BackgroundTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if err := Checkpoint(req.Context()); err != nil {
		return nil, err
	}
	clone := req.Clone(req.Context())
	if clone.Body != nil {
		clone.Body = &pacedBody{clone.Body, req.Context()}
	}
	base := t.Base
	if base == nil {
		base = http.DefaultTransport
	}
	resp, err := base.RoundTrip(clone)
	if err == nil {
		resp.Body = &pacedBody{resp.Body, req.Context()}
	}
	return resp, err
}
