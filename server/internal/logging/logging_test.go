package logging

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"
)

func TestLogger_WritesToUnderlyingWriter(t *testing.T) {
	var buf bytes.Buffer
	l := New(&buf, 10)

	l.Info("server started on %s", ":19900")

	out := buf.String()
	if !strings.Contains(out, "[INFO]") || !strings.Contains(out, "server started on :19900") {
		t.Errorf("unexpected output: %q", out)
	}
}

func TestLogger_Entries_FiltersByLevel(t *testing.T) {
	l := New(&bytes.Buffer{}, 10)
	l.Debug("debug message")
	l.Info("info message")
	l.Warn("warn message")
	l.Error("error message")
	l.Critical("critical message")

	entries := l.Entries([]Level{LevelWarn, LevelError}, 0)
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d: %v", len(entries), entries)
	}
	if entries[0].Level != LevelWarn || entries[1].Level != LevelError {
		t.Errorf("unexpected levels: %v, %v", entries[0].Level, entries[1].Level)
	}
}

func TestLogger_Entries_EmptyLevelsMeansAll(t *testing.T) {
	l := New(&bytes.Buffer{}, 10)
	l.Debug("a")
	l.Info("b")

	entries := l.Entries(nil, 0)
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
}

func TestLogger_Entries_LimitReturnsMostRecent(t *testing.T) {
	l := New(&bytes.Buffer{}, 10)
	for i := range 5 {
		l.Info("message %d", i)
	}

	entries := l.Entries(nil, 2)
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0].Message != "message 3" || entries[1].Message != "message 4" {
		t.Errorf("expected the 2 most recent messages, got %q, %q", entries[0].Message, entries[1].Message)
	}
}

func TestLogger_RingBuffer_RespectsCapacity(t *testing.T) {
	l := New(&bytes.Buffer{}, 3)
	for i := range 10 {
		l.Info("message %d", i)
	}

	entries := l.Entries(nil, 0)
	if len(entries) != 3 {
		t.Fatalf("expected the ring buffer to cap at 3 entries, got %d", len(entries))
	}
	if entries[0].Message != "message 7" || entries[2].Message != "message 9" {
		t.Errorf("expected only the last 3 messages retained, got %v", entries)
	}
}

func TestLogger_FilterThenLimit_AppliesInThatOrder(t *testing.T) {
	l := New(&bytes.Buffer{}, 100)
	l.Info("info 1")
	l.Warn("warn 1")
	l.Info("info 2")
	l.Warn("warn 2")
	l.Info("info 3")

	// Asking for the last 1 "warn" entry should skip over the info lines
	// entirely, not just take the last raw line and then filter it away.
	entries := l.Entries([]Level{LevelWarn}, 1)
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d: %v", len(entries), entries)
	}
	if entries[0].Message != "warn 2" {
		t.Errorf("expected the most recent warn entry, got %q", entries[0].Message)
	}
}

func TestStartHeartbeat_EmitsWhenIdle(t *testing.T) {
	l := New(&bytes.Buffer{}, 10)
	l.Info("startup")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	l.StartHeartbeat(ctx, 30*time.Millisecond, 10*time.Millisecond)

	deadline := time.Now().Add(1 * time.Second)
	for time.Now().Before(deadline) {
		for _, e := range l.Entries(nil, 0) {
			if strings.Contains(e.Message, "heartbeat") {
				return // success
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("expected a heartbeat entry after being idle, got none")
}

func TestStartHeartbeat_DoesNotFireWhileActive(t *testing.T) {
	l := New(&bytes.Buffer{}, 100)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	l.StartHeartbeat(ctx, 300*time.Millisecond, 20*time.Millisecond)

	until := time.Now().Add(200 * time.Millisecond)
	for time.Now().Before(until) {
		l.Info("keeping busy")
		time.Sleep(10 * time.Millisecond)
	}

	for _, e := range l.Entries(nil, 0) {
		if strings.Contains(e.Message, "heartbeat") {
			t.Fatalf("did not expect a heartbeat while continuously active, got: %v", e)
		}
	}
}

func TestLevel_IsValid(t *testing.T) {
	if !LevelInfo.IsValid() {
		t.Error("expected \"info\" to be a valid level")
	}
	if Level("bogus").IsValid() {
		t.Error("expected \"bogus\" to be invalid")
	}
}
