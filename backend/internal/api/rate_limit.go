package api

import (
	"context"
	"net"
	"net/http"
	"sync"
	"time"
)

const maxAuthRateLimitEntries = 1024
const authRateLimitLockoutFailures = 8

type authRateLimiter struct {
	mu      sync.Mutex
	entries map[string]authRateLimitEntry
}

type authRateLimitEntry struct {
	failures    int
	lastSeen    time.Time
	lockedUntil time.Time
}

func newAuthRateLimiter() *authRateLimiter {
	return &authRateLimiter{entries: map[string]authRateLimitEntry{}}
}

func (l *authRateLimiter) wait(ctx context.Context, key string) error {
	delay := l.delay(key)
	if delay <= 0 {
		return nil
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func (l *authRateLimiter) recordFailure(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.pruneLocked(time.Now())
	entry := l.entries[key]
	entry.failures++
	now := time.Now()
	entry.lastSeen = now
	if entry.failures >= authRateLimitLockoutFailures {
		entry.lockedUntil = now.Add(1 * time.Minute)
	}
	l.entries[key] = entry
}

func (l *authRateLimiter) recordSuccess(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.entries, key)
}

func (l *authRateLimiter) delay(key string) time.Duration {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.pruneLocked(time.Now())
	entry := l.entries[key]
	if entry.lockedUntil.After(time.Now()) {
		return time.Until(entry.lockedUntil)
	}
	if entry.failures == 0 || time.Since(entry.lastSeen) > 10*time.Minute {
		return 0
	}
	shift := entry.failures - 1
	if shift > 4 {
		shift = 4
	}
	delay := time.Duration(1<<shift) * 500 * time.Millisecond
	if delay > 8*time.Second {
		return 8 * time.Second
	}
	return delay
}

func (l *authRateLimiter) pruneLocked(now time.Time) {
	for key, entry := range l.entries {
		if now.Sub(entry.lastSeen) > 10*time.Minute {
			delete(l.entries, key)
		}
	}
	for len(l.entries) > maxAuthRateLimitEntries {
		var oldestKey string
		var oldest time.Time
		for key, entry := range l.entries {
			if oldestKey == "" || entry.lastSeen.Before(oldest) {
				oldestKey = key
				oldest = entry.lastSeen
			}
		}
		delete(l.entries, oldestKey)
	}
}

func authRateLimitKey(r *http.Request, scope string) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil || host == "" {
		host = r.RemoteAddr
	}
	return scope + ":" + host
}
