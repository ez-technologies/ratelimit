// Package ratelimit is an in-memory rate limiter with one token bucket per key.
//
// Limits are per process: with several instances each one counts on its own, so the real ceiling is the limit times
// the number of instances.
package ratelimit

import (
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// Limiter holds one token bucket per key. The table is never cleared because of its size: an entry is dropped only
// when it has been idle long enough to be full again, which makes it indistinguishable from a new one. Filling the
// table with fake keys therefore cannot reset a limit that is in use.
type Limiter struct {
	mu        sync.Mutex
	entries   map[string]*entry
	idle      time.Duration
	now       func() time.Time
	lastSweep time.Time
}

type entry struct {
	l    *rate.Limiter
	last time.Time
	keep time.Duration // idle time after which the entry may go: a full refill, and at least the Limiter's idle
}

// New makes a Limiter that forgets keys unused for at least idle (and always for as long as a full refill takes).
func New(idle time.Duration) *Limiter {
	return &Limiter{entries: map[string]*entry{}, idle: idle, now: time.Now}
}

// Allow reports whether key may act now: a burst of burst, then one more every `every`. The first call for a key
// fixes its rate; later calls with other values for the same key use the first ones.
func (l *Limiter) Allow(key string, every time.Duration, burst int) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	l.sweep(now)
	e, ok := l.entries[key]
	if !ok {
		keep := every * time.Duration(burst)
		if keep < l.idle {
			keep = l.idle
		}
		e = &entry{l: rate.NewLimiter(rate.Every(every), burst), keep: keep}
		l.entries[key] = e
	}
	e.last = now
	return e.l.AllowN(now, 1)
}

// Len is the number of keys held.
func (l *Limiter) Len() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.entries)
}

// sweep drops idle entries, at most once a minute (or once per idle period when that is shorter).
func (l *Limiter) sweep(now time.Time) {
	every := time.Minute
	if l.idle < every {
		every = l.idle
	}
	if now.Sub(l.lastSweep) < every {
		return
	}
	l.lastSweep = now
	for k, e := range l.entries {
		if now.Sub(e.last) >= e.keep {
			delete(l.entries, k)
		}
	}
}
