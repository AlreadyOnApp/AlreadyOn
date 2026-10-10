package governor

import (
	"testing"
	"time"
)

// Synthetic elapsed intervals catch the old one-CPU ceiling without assuming
// that a shared runner can actually schedule two CPUs of simultaneous work.
func TestMulticoreCreditRefill(t *testing.T) {
	g := New(4)
	g.allowance = 2
	g.refillLocked(budgetInterval)
	if g.credit != 2*budgetInterval {
		t.Fatalf("two-CPU allowance earned %v, want %v", g.credit, 2*budgetInterval)
	}
	g.refillLocked(time.Second)
	if g.credit != 2*budgetInterval {
		t.Fatalf("idle credit grew beyond one interval: %v", g.credit)
	}
	g.credit = -3 * budgetInterval
	g.refillLocked(budgetInterval)
	if g.credit != -budgetInterval {
		t.Fatalf("CPU debt was discarded: %v", g.credit)
	}
}
