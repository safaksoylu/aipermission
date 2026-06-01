package console

import (
	"testing"
	"time"
)

func TestParsePositiveInt(t *testing.T) {
	if got := parsePositiveInt("24", 80); got != 24 {
		t.Fatalf("expected parsed value, got %d", got)
	}
	if got := parsePositiveInt("0", 80); got != 80 {
		t.Fatalf("expected fallback for zero, got %d", got)
	}
	if got := parsePositiveInt("bad", 80); got != 80 {
		t.Fatalf("expected fallback for bad input, got %d", got)
	}
}

func TestConsoleIntervalLimiter(t *testing.T) {
	limiter := newConsoleIntervalLimiter(50 * time.Millisecond)
	if !limiter.allow() {
		t.Fatalf("first call should be allowed")
	}
	if limiter.allow() {
		t.Fatalf("second immediate call should be denied")
	}
	time.Sleep(60 * time.Millisecond)
	if !limiter.allow() {
		t.Fatalf("call after interval should be allowed")
	}
}
