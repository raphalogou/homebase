package auth

import (
	"sync"
	"time"
)

// Limiter counts failed logins per address in a sliding window.
type Limiter struct {
	max    int
	window time.Duration
	now    func() time.Time

	mu       sync.Mutex
	failures map[string][]time.Time
}

// NewLimiter allows max failures per window for each address.
func NewLimiter(max int, window time.Duration, now func() time.Time) *Limiter {
	return &Limiter{max: max, window: window, now: now, failures: map[string][]time.Time{}}
}

// Allow reports whether addr may try again.
func (l *Limiter) Allow(addr string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.recent(addr)) < l.max
}

// Fail records a failed attempt.
func (l *Limiter) Fail(addr string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.failures[addr] = append(l.recent(addr), l.now())
	if len(l.failures) > 1000 {
		l.prune()
	}
}

// Reset forgets addr's failures after a successful login.
func (l *Limiter) Reset(addr string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.failures, addr)
}

func (l *Limiter) recent(addr string) []time.Time {
	cutoff := l.now().Add(-l.window)
	list := l.failures[addr]
	i := 0
	for i < len(list) && !list[i].After(cutoff) {
		i++
	}
	return list[i:]
}

// prune drops addresses with no recent failures, so a scan from many
// addresses cannot grow the map without bound.
func (l *Limiter) prune() {
	for addr := range l.failures {
		if r := l.recent(addr); len(r) == 0 {
			delete(l.failures, addr)
		} else {
			l.failures[addr] = r
		}
	}
}
