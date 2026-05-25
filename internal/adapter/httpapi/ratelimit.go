package httpapi

import (
	"sync"
	"time"
)

type authRateLimiter struct {
	mu      sync.Mutex
	window  time.Duration
	limit   int
	entries map[string][]time.Time
}

func newAuthRateLimiter(perMinute int) *authRateLimiter {
	return &authRateLimiter{
		window:  time.Minute,
		limit:   perMinute,
		entries: map[string][]time.Time{},
	}
}

func (l *authRateLimiter) Allow(key string) bool {
	now := time.Now()
	cutoff := now.Add(-l.window)

	l.mu.Lock()
	defer l.mu.Unlock()

	hits := l.entries[key]
	next := hits[:0]
	for _, at := range hits {
		if at.After(cutoff) {
			next = append(next, at)
		}
	}
	if len(next) >= l.limit {
		l.entries[key] = next
		return false
	}
	next = append(next, now)
	l.entries[key] = next
	return true
}
