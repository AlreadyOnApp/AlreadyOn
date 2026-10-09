package status

import (
	"errors"
	"testing"
	"time"
)

type fakeClock struct{ t time.Time }

func (c *fakeClock) now() time.Time          { return c.t }
func (c *fakeClock) advance(d time.Duration) { c.t = c.t.Add(d) }

func newTestTracker() (*Tracker, *fakeClock) {
	c := &fakeClock{t: time.Date(2026, 9, 29, 9, 0, 0, 0, time.UTC)}
	return New(c.now), c
}

func TestTracker_StartsPending(t *testing.T) {
	tr, _ := newTestTracker()
	s := tr.Snapshot()
	if s.Phase != PhasePending || s.Active || s.StartupComplete || s.StartedAt != nil {
		t.Errorf("initial snapshot = %+v, want pending/inactive/incomplete", s)
	}
}

func TestTracker_CleanRunLifecycle(t *testing.T) {
	tr, clock := newTestTracker()
	tr.Begin(TriggerStartup, PhaseScanning)
	s := tr.Snapshot()
	if !s.Active || s.Phase != PhaseScanning || s.Trigger != TriggerStartup || s.StartedAt == nil || s.FinishedAt != nil {
		t.Fatalf("after Begin = %+v", s)
	}

	clock.advance(90 * time.Second)
	tr.SetPhase(PhaseClassifying)
	if got := tr.Snapshot().Phase; got != PhaseClassifying {
		t.Errorf("phase = %s, want classifying", got)
	}

	clock.advance(10 * time.Second)
	tr.End()
	s = tr.Snapshot()
	if s.Active || s.Phase != PhaseReady || !s.StartupComplete || s.LastError != "" {
		t.Fatalf("after End = %+v, want ready/complete", s)
	}
	if got := s.FinishedAt.Sub(*s.StartedAt); got != 100*time.Second {
		t.Errorf("duration = %s, want 100s", got)
	}
}

func TestTracker_RecordedErrorEndsFailedButKeepsRunning(t *testing.T) {
	tr, _ := newTestTracker()
	tr.Begin(TriggerStartup, PhaseScanning)
	tr.RecordError(errors.New("scan exploded"))
	if s := tr.Snapshot(); !s.Active || s.Phase != PhaseScanning || s.LastError != "scan exploded" {
		t.Fatalf("mid-run = %+v, want still active/scanning with error recorded", s)
	}
	tr.End()
	s := tr.Snapshot()
	if s.Phase != PhaseFailed || s.LastError != "scan exploded" || !s.StartupComplete {
		t.Errorf("after End = %+v, want failed with error, startup complete", s)
	}
}

func TestTracker_NextRunClearsPreviousError(t *testing.T) {
	tr, _ := newTestTracker()
	tr.Begin(TriggerStartup, PhaseScanning)
	tr.RecordError(errors.New("boom"))
	tr.End()

	tr.Begin(TriggerRefresh, PhaseClassifying)
	tr.End()
	s := tr.Snapshot()
	if s.Phase != PhaseReady || s.LastError != "" {
		t.Errorf("second run = %+v, want ready with no error", s)
	}
}

func TestTracker_RefreshDoesNotMarkStartupComplete(t *testing.T) {
	tr, _ := newTestTracker()
	tr.Begin(TriggerRefresh, PhaseClassifying)
	tr.End()
	if tr.Snapshot().StartupComplete {
		t.Error("a refresh run marked StartupComplete")
	}
}

func TestTracker_InactiveCallsAreNoOps(t *testing.T) {
	tr, _ := newTestTracker()
	tr.SetPhase(PhaseScheduling)
	tr.RecordError(errors.New("ignored"))
	tr.End()
	s := tr.Snapshot()
	if s.Phase != PhasePending || s.LastError != "" || s.FinishedAt != nil {
		t.Errorf("snapshot = %+v, want untouched pending state", s)
	}
}

func TestTracker_SnapshotIsACopy(t *testing.T) {
	tr, _ := newTestTracker()
	tr.Begin(TriggerStartup, PhaseScanning)
	s := tr.Snapshot()
	*s.StartedAt = time.Time{}
	if tr.Snapshot().StartedAt.IsZero() {
		t.Error("mutating a Snapshot changed the tracker's state")
	}
}
