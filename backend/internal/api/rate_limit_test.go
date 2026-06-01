package api

import (
	"fmt"
	"testing"
	"time"
)

func TestAuthRateLimiterPrunesStaleEntriesAndCapsSize(t *testing.T) {
	limiter := newAuthRateLimiter()
	now := time.Now()
	limiter.entries["stale"] = authRateLimitEntry{failures: 1, lastSeen: now.Add(-11 * time.Minute)}
	for i := range maxAuthRateLimitEntries + 20 {
		limiter.entries[fmt.Sprintf("fresh-%04d", i)] = authRateLimitEntry{failures: 1, lastSeen: now.Add(time.Duration(i) * time.Millisecond)}
	}

	limiter.pruneLocked(now)

	if _, ok := limiter.entries["stale"]; ok {
		t.Fatalf("expected stale entry to be pruned")
	}
	if len(limiter.entries) > maxAuthRateLimitEntries {
		t.Fatalf("expected at most %d entries, got %d", maxAuthRateLimitEntries, len(limiter.entries))
	}
}

func TestAuthRateLimiterBackoffAndLockout(t *testing.T) {
	limiter := newAuthRateLimiter()
	key := "unlock:127.0.0.1"
	if delay := limiter.delay(key); delay != 0 {
		t.Fatalf("new key should not be delayed, got %s", delay)
	}
	limiter.recordFailure(key)
	if delay := limiter.delay(key); delay < 500*time.Millisecond {
		t.Fatalf("expected first backoff delay, got %s", delay)
	}
	for range authRateLimitLockoutFailures - 1 {
		limiter.recordFailure(key)
	}
	if delay := limiter.delay(key); delay < 50*time.Second {
		t.Fatalf("expected temporary lockout delay, got %s", delay)
	}
	limiter.recordSuccess(key)
	if delay := limiter.delay(key); delay != 0 {
		t.Fatalf("successful auth should clear limiter, got %s", delay)
	}
}
