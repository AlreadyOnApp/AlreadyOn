package auth

import (
	"fmt"
	"testing"
	"time"
)

// Saturating the address table must preserve existing lockouts and its bound.
func TestLimiterBoundsAndExpiry(t *testing.T) {
	now := time.Unix(10000, 0)
	l := NewLimiter(func() time.Time { return now })
	for i := 0; i < maxTracked+20; i++ {
		for j := 0; j < freeAttempts; j++ {
			l.Fail(fmt.Sprint(i))
		}
	}
	if len(l.clients) != maxTracked {
		t.Fatalf("tracked %d clients", len(l.clients))
	}
	if ok, _ := l.Allow("0"); ok {
		t.Fatal("existing lockout lost")
	}
	if ok, _ := l.Allow("new"); ok {
		t.Fatal("overflow bypassed")
	}
	l.Succeed("0")
	if len(l.clients) != maxTracked-1 {
		t.Fatal("success did not clear failures")
	}
	now = now.Add(forgetAfter)
	if ok, _ := l.Allow("1"); !ok {
		t.Fatal("expired lockout retained")
	}
	l.Fail("1")
	if l.clients["1"].failures != 1 {
		t.Fatal("expired failures retained")
	}
}
