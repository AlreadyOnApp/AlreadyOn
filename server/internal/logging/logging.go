// Package logging is a small leveled logger used in place of the
// standard library's log package throughout the server. Every entry
// still goes to its underlying writer (stdout in production, so
// `journalctl`/the terminal keep working exactly as before), and is
// also kept in a bounded in-memory ring buffer that the admin log
// viewer API (GET /api/logs) reads from.
package logging

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"
)

// Level is a log entry's severity.
type Level string

const (
	LevelDebug    Level = "debug"
	LevelInfo     Level = "info"
	LevelWarn     Level = "warn"
	LevelError    Level = "error"
	LevelCritical Level = "critical"
)

// Levels lists every known level, in increasing severity order.
var Levels = []Level{LevelDebug, LevelInfo, LevelWarn, LevelError, LevelCritical}

// IsValid reports whether l is one of the known levels.
func (l Level) IsValid() bool {
	for _, known := range Levels {
		if l == known {
			return true
		}
	}
	return false
}

// Entry is one recorded log line.
type Entry struct {
	Time    time.Time
	Level   Level
	Message string
}

// Logger writes formatted entries to an underlying writer and retains
// the most recent ones (up to capacity) for later retrieval.
type Logger struct {
	mu       sync.Mutex
	out      io.Writer
	capacity int
	buf      []Entry
}

// New creates a Logger that writes to out and retains up to capacity
// recent entries.
func New(out io.Writer, capacity int) *Logger {
	if capacity <= 0 {
		capacity = 1000
	}
	return &Logger{out: out, capacity: capacity}
}

func (l *Logger) logf(level Level, format string, args ...any) {
	entry := Entry{Time: time.Now(), Level: level, Message: fmt.Sprintf(format, args...)}

	l.mu.Lock()
	l.buf = append(l.buf, entry)
	if len(l.buf) > l.capacity {
		l.buf = l.buf[len(l.buf)-l.capacity:]
	}
	out := l.out
	l.mu.Unlock()

	fmt.Fprintf(out, "%s [%s] %s\n", entry.Time.Format(time.RFC3339), strings.ToUpper(string(level)), entry.Message)
}

func (l *Logger) Debug(format string, args ...any)    { l.logf(LevelDebug, format, args...) }
func (l *Logger) Info(format string, args ...any)     { l.logf(LevelInfo, format, args...) }
func (l *Logger) Warn(format string, args ...any)     { l.logf(LevelWarn, format, args...) }
func (l *Logger) Error(format string, args ...any)    { l.logf(LevelError, format, args...) }
func (l *Logger) Critical(format string, args ...any) { l.logf(LevelCritical, format, args...) }

// Fatal logs at critical severity, then exits the process — a drop-in
// replacement for log.Fatalf.
func (l *Logger) Fatal(format string, args ...any) {
	l.logf(LevelCritical, format, args...)
	os.Exit(1)
}

// Entries returns up to limit of the most recent entries whose level is
// in levels (all levels, if levels is empty), oldest-first.
func (l *Logger) Entries(levels []Level, limit int) []Entry {
	l.mu.Lock()
	defer l.mu.Unlock()

	var matched []Entry
	for _, e := range l.buf {
		if len(levels) == 0 || levelIn(e.Level, levels) {
			matched = append(matched, e)
		}
	}
	if limit > 0 && len(matched) > limit {
		matched = matched[len(matched)-limit:]
	}
	return matched
}

// StartHeartbeat runs in the background, checking every checkInterval
// whether idleThreshold has passed since the last log entry. If so, it
// emits a low-severity "heartbeat" entry — otherwise a genuinely quiet
// server would leave the live log viewer looking frozen or dead rather
// than merely uneventful. Stops when ctx is done.
func (l *Logger) StartHeartbeat(ctx context.Context, idleThreshold, checkInterval time.Duration) {
	go func() {
		ticker := time.NewTicker(checkInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if l.idleFor() >= idleThreshold {
					l.Debug("heartbeat — no other log activity in the last %s", idleThreshold)
				}
			}
		}
	}()
}

func (l *Logger) idleFor() time.Duration {
	l.mu.Lock()
	defer l.mu.Unlock()
	if len(l.buf) == 0 {
		return 0
	}
	return time.Since(l.buf[len(l.buf)-1].Time)
}

func levelIn(level Level, levels []Level) bool {
	for _, l := range levels {
		if level == l {
			return true
		}
	}
	return false
}

// Default is the process-wide logger every package logs through,
// mirroring the standard library log package's own default-logger
// convention that the rest of this codebase already follows.
var Default = New(os.Stdout, 2000)

func Debug(format string, args ...any)    { Default.Debug(format, args...) }
func Info(format string, args ...any)     { Default.Info(format, args...) }
func Warn(format string, args ...any)     { Default.Warn(format, args...) }
func Error(format string, args ...any)    { Default.Error(format, args...) }
func Critical(format string, args ...any) { Default.Critical(format, args...) }
func Fatal(format string, args ...any)    { Default.Fatal(format, args...) }

// Entries returns entries from the default logger. See Logger.Entries.
func Entries(levels []Level, limit int) []Entry { return Default.Entries(levels, limit) }

// StartHeartbeat starts a heartbeat on the default logger. See
// Logger.StartHeartbeat.
func StartHeartbeat(ctx context.Context, idleThreshold, checkInterval time.Duration) {
	Default.StartHeartbeat(ctx, idleThreshold, checkInterval)
}
