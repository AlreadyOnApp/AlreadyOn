package governor

import (
	"context"
	"time"
)

const budgetInterval = 20 * time.Millisecond

// runBudget meters process CPU time (all threads), not elapsed running time.
// Credit cannot accumulate while idle; a burst is repaid before any job resumes.
// This limiter only signals explicitly attached analysis process groups.
func (g *Governor) runBudget(ctx context.Context) {
	ticker := time.NewTicker(budgetInterval)
	defer ticker.Stop()
	previous := time.Now()
	for {
		select {
		case <-ctx.Done():
			return // workers' CommandContext cancellation kills stopped groups too
		case now := <-ticker.C:
			g.mu.Lock()
			elapsed := now.Sub(previous)
			previous = now
			g.refillLocked(elapsed)
			readable := true
			for t := range g.tickets {
				if !g.chargeLocked(t) {
					readable = false
				}
			}
			g.throttled = !readable || g.credit <= 0
			g.applyLocked()
			g.mu.Unlock()
		}
	}
}

func (g *Governor) chargeLocked(t *Ticket) bool {
	if !g.budget || t.pgid == 0 {
		return true
	}
	cpu, err := processCPU(t.pgid)
	if err != nil {
		return false
	}
	if cpu >= t.cpu {
		g.credit -= cpu - t.cpu
		g.charged += cpu - t.cpu
	}
	t.cpu = cpu
	return true
}

type ticketKey struct{}

// WithTicket carries the background claim into probes as well as analysis.
// Foreground playback probes use an ordinary context and remain unthrottled.
func WithTicket(ctx context.Context, t *Ticket) context.Context {
	return context.WithValue(ctx, ticketKey{}, t)
}

// TicketFrom returns the analysis claim, or nil for foreground work.
func TicketFrom(ctx context.Context) *Ticket {
	t, _ := ctx.Value(ticketKey{}).(*Ticket)
	return t
}

// Each interval earns allowance CPU-seconds, including multicore allowances.
// Capping idle credit bounds bursts while negative credit retains unpaid work.
func (g *Governor) refillLocked(elapsed time.Duration) {
	g.credit = min(time.Duration(float64(budgetInterval)*g.allowance), g.credit+time.Duration(float64(elapsed)*g.allowance))
}
