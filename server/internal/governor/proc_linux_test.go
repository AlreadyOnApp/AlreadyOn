//go:build linux

package governor

import (
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"testing"
)

func TestProcStatParsing(t *testing.T) {
	s, err := parseProcStat("cpu  100 20 30 400 50 5 6 7 0 0\n")
	if err != nil || s.Busy != 100+20+30+50+5+6+7 || s.Total != s.Busy+400 {
		t.Fatalf("parse = %+v %v", s, err)
	}
	for _, bad := range []string{"", "cpu0 1 2 3 4 5 6 7 8", "cpu 1 2 3", "cpu a b c d e f g h"} {
		if _, err := parseProcStat(bad); err == nil {
			t.Errorf("accepted %q", bad)
		}
	}
}

func sleeper(t *testing.T) *exec.Cmd {
	t.Helper()
	cmd := exec.Command("sleep", "60")
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		t.Skip(err)
	}
	t.Cleanup(func() { _ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL); _ = cmd.Wait() })
	return cmd
}

func TestJobsBelowTheTargetArePausedAndResumed(t *testing.T) {
	if _, err := os.Stat("/proc/self/stat"); err != nil {
		t.Skip("needs /proc")
	}
	g := newGovernor(4, 4)
	g.target = 2
	first, _ := g.Acquire(t.Context(), PriorityLow)
	second, _ := g.Acquire(t.Context(), PriorityLow)
	a, b := sleeper(t), sleeper(t)
	first.Attach(a.Process.Pid)
	second.Attach(b.Process.Pid)
	g.mu.Lock()
	g.target = 1
	g.applyLocked()
	g.mu.Unlock()
	waitFor(t, "the later job is paused", func() bool { return state(b.Process.Pid) == "T" })
	if state(a.Process.Pid) == "T" {
		t.Fatal("the earlier job should keep running")
	}
	first.Release()
	waitFor(t, "the paused job resumes when a slot frees", func() bool { return state(b.Process.Pid) != "T" })
}

func TestHighPriorityJobsOutrankLowOnes(t *testing.T) {
	if _, err := os.Stat("/proc/self/stat"); err != nil {
		t.Skip("needs /proc")
	}
	g := newGovernor(4, 4)
	g.target = 2
	low, _ := g.Acquire(t.Context(), PriorityLow)
	high, _ := g.Acquire(t.Context(), PriorityHigh)
	l, h := sleeper(t), sleeper(t)
	low.Attach(l.Process.Pid)
	high.Attach(h.Process.Pid)
	g.mu.Lock()
	g.target = 1
	g.applyLocked()
	g.mu.Unlock()
	waitFor(t, "the low-priority job is paused", func() bool { return state(l.Process.Pid) == "T" })
	if state(h.Process.Pid) == "T" {
		t.Fatal("the high-priority job should keep running")
	}
}

func state(pid int) string {
	raw, err := os.ReadFile("/proc/" + strconv.Itoa(pid) + "/stat")
	if err != nil {
		return ""
	}
	rest := string(raw[strings.LastIndex(string(raw), ")")+2:])
	return rest[:1]
}
