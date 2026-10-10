//go:build linux

package governor

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"testing"
	"time"
)

// The helper creates CPU pressure without files, media, network or privileges.
func TestBudgetHelperProcess(t *testing.T) {
	if os.Getenv("ALREADYON_BUDGET_TEST_HELPER") != "1" {
		return
	}
	runtime.GOMAXPROCS(4)
	for range 4 {
		go func() {
			for {
				runtime.KeepAlive(1)
			}
		}()
	}
	select {}
}

func TestBudgetMetersAllThreadsAndCancelsStoppedJob(t *testing.T) {
	g := New(16)
	g.allowance = 0.75
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	go g.runBudget(ctx)
	ticket, err := g.Acquire(ctx, PriorityLow)
	if err != nil {
		t.Fatal(err)
	}
	defer ticket.Release()
	job, stop := context.WithCancel(WithTicket(ctx, ticket))
	defer stop()
	cmd := exec.CommandContext(job, os.Args[0], "-test.run=^TestBudgetHelperProcess$")
	cmd.Env = append(os.Environ(), "ALREADYON_BUDGET_TEST_HELPER=1")
	done := make(chan error, 1)
	start := time.Now()
	go func() { done <- RunCommand(job, cmd) }()
	time.Sleep(4 * time.Second)
	// Read under the governor lock; cmd.Process is initialized by the worker.
	waitFor(t, "budget stops the CPU-heavy process", func() bool {
		g.mu.Lock()
		defer g.mu.Unlock()
		return ticket.pgid != 0 && state(ticket.pgid) == "T"
	})
	stop()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("cancellation did not kill stopped process")
	}
	elapsed := time.Since(start)
	used := cmd.ProcessState.UserTime() + cmd.ProcessState.SystemTime()
	ratio := float64(used) / float64(elapsed)
	t.Logf("4-thread analysis: CPU=%s wall=%s logical CPUs=%.3f", used, elapsed, ratio)
	if ratio > 0.75+0.15 || used < 100*time.Millisecond {
		t.Fatalf("budget did not bound progressing work: %.3f CPUs", ratio)
	}
}

func TestProcessCPUParsing(t *testing.T) {
	fields := strings.Fields("S 1 2 3 4 5 6 7 8 9 10 150 25")
	got, err := parseProcessCPU("123 (name with ) parentheses) "+strings.Join(fields, " "), 100)
	if err != nil || got != 1750*time.Millisecond {
		t.Fatalf("%s %v", got, err)
	}
	for _, raw := range []string{"", "1 (short) S", "1 (bad) " + strings.Repeat("x ", 13)} {
		if _, err := parseProcessCPU(raw, 100); err == nil {
			t.Fatal(fmt.Sprintf("accepted %q", raw))
		}
	}
}

func TestForegroundCommandDoesNotConsumeBudget(t *testing.T) {
	g := New(1)
	g.allowance = 0.75
	g.throttled = true
	claim, err := g.Acquire(t.Context(), PriorityLow)
	if err != nil {
		t.Fatal(err)
	}
	defer claim.Release()
	bg, stop := context.WithCancel(WithTicket(t.Context(), claim))
	defer stop()
	background := exec.CommandContext(bg, os.Args[0], "-test.run=^TestBudgetHelperProcess$")
	background.Env = append(os.Environ(), "ALREADYON_BUDGET_TEST_HELPER=1")
	done := make(chan error, 1)
	go func() { done <- RunCommand(bg, background) }()
	waitFor(t, "background is stopped", func() bool {
		g.mu.Lock()
		defer g.mu.Unlock()
		return claim.pgid != 0 && state(claim.pgid) == "T"
	})
	ctx, cancel := context.WithTimeout(t.Context(), 400*time.Millisecond)
	defer cancel()
	foreground := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestBudgetHelperProcess$")
	foreground.Env = append(os.Environ(), "ALREADYON_BUDGET_TEST_HELPER=1")
	_ = RunCommand(ctx, foreground)
	if foreground.ProcessState == nil || foreground.ProcessState.UserTime() < 10*time.Millisecond {
		t.Fatal("foreground did not progress while analysis was stopped")
	}
	stop()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("background failed to cancel")
	}
}

func TestRealFFmpegCompletesWithinBackgroundBudget(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip(err)
	}
	g := New(1)
	g.allowance = 0.75
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()
	go g.runBudget(ctx)
	ticket, err := g.Acquire(ctx, PriorityLow)
	if err != nil {
		t.Fatal(err)
	}
	defer ticket.Release()
	job := WithTicket(ctx, ticket)
	cmd := exec.CommandContext(job, "ffmpeg", "-v", "error", "-nostdin", "-threads", "1", "-filter_threads", "1", "-f", "lavfi", "-i", "sine=duration=120:sample_rate=48000", "-af", "loudnorm=I=-16:TP=-1.5:LRA=50", "-f", "null", "-")
	start := time.Now()
	if err := RunCommand(job, cmd); err != nil {
		t.Fatal(err)
	}
	elapsed := time.Since(start)
	used := cmd.ProcessState.UserTime() + cmd.ProcessState.SystemTime()
	t.Logf("120s synthetic loudness: CPU=%s wall=%s logical CPUs=%.3f", used, elapsed, float64(used)/float64(elapsed))
	if used > time.Duration(float64(elapsed)*(0.75+0.15))+50*time.Millisecond {
		t.Fatal("FFmpeg exceeded the budget tolerance")
	}
}

// Shared runners can supply less CPU than the allowance. The real process must
// progress and remain bounded; deterministic credit tests check multicore refill.
func TestMulticoreBudgetMakesParallelProgress(t *testing.T) {
	g := New(4)
	g.allowance = 2
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	go g.runBudget(ctx)
	ticket, err := g.Acquire(ctx, PriorityLow)
	if err != nil {
		t.Fatal(err)
	}
	defer ticket.Release()
	job, stop := context.WithTimeout(WithTicket(ctx, ticket), 3*time.Second)
	defer stop()
	cmd := exec.CommandContext(job, os.Args[0], "-test.run=^TestBudgetHelperProcess$")
	cmd.Env = append(os.Environ(), "ALREADYON_BUDGET_TEST_HELPER=1")
	start := time.Now()
	_ = RunCommand(job, cmd)
	if cmd.ProcessState == nil {
		t.Fatal("helper did not start")
	}
	used := cmd.ProcessState.UserTime() + cmd.ProcessState.SystemTime()
	ratio := float64(used) / float64(time.Since(start))
	t.Logf("two-CPU allowance: %.3f logical CPUs", ratio)
	if used < 100*time.Millisecond || ratio > 2.2 {
		t.Fatalf("parallel CPU budget not respected: %.3f", ratio)
	}
}
