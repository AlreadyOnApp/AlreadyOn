package auth

import (
	"sync"
	"time"
)

// Limiter slows repeated failed sign-ins from one client address. After
// freeAttempts failures the address is locked out for a period that doubles
// with each further failure, up to maxLockout. Success clears it. It lives in
// memory, so a restart resets it.
type Limiter struct {
	mu       sync.Mutex
	now      func() time.Time
	clients  map[string]*attempts
	overflow attempts
}

type attempts struct {
	failures int
	until    time.Time
	last     time.Time
}

const (
	freeAttempts = 5
	baseLockout  = 30 * time.Second
	maxLockout   = 15 * time.Minute
	forgetAfter  = time.Hour
	maxTracked   = 4096
)

func NewLimiter(now func() time.Time) *Limiter {
	return &Limiter{now: now, clients: map[string]*attempts{}}
}

// Allow reports whether the client may try now, and how long to wait if not.
func (l *Limiter) Allow(client string) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	a := l.clients[client]
	if a != nil && now.Sub(a.last) >= forgetAfter {
		delete(l.clients, client)
		a = nil
	}
	if a == nil && len(l.clients) >= maxTracked {
		a = &l.overflow
	}
	if a == nil {
		return true, 0
	}
	if wait := a.until.Sub(now); wait > 0 {
		return false, wait
	}
	return true, 0
}

// Fail records a failed attempt.
func (l *Limiter) Fail(client string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	if len(l.clients) >= maxTracked {
		for k, a := range l.clients {
			if now.Sub(a.last) > forgetAfter {
				delete(l.clients, k)
			}
		}
	}
	a := l.clients[client]
	if a == nil {
		// A shared overflow bucket preserves existing lockouts without growing
		// memory or evicting a locked client when many addresses fail at once.
		if len(l.clients) >= maxTracked {
			a = &l.overflow
		} else {
			a = &attempts{}
			l.clients[client] = a
		}
	}
	if now.Sub(a.last) >= forgetAfter {
		*a = attempts{}
	}
	a.failures++
	a.last = now
	if a.failures >= freeAttempts {
		lock := baseLockout
		for i := freeAttempts; i < a.failures && lock < maxLockout; i++ {
			lock *= 2
		}
		a.until = now.Add(min(lock, maxLockout))
	}
}

// Succeed forgets a client's failures.
func (l *Limiter) Succeed(client string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.clients, client)
}
