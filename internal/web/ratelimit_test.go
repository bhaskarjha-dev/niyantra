package web

import (
	"testing"
	"time"
)

func TestRateLimiter_AllowsUpToLimit(t *testing.T) {
	rl := &rateLimiter{
		buckets: make(map[string]*rateBucket),
		limits:  map[string]int{"test": 3},
		window:  time.Minute,
	}

	for i := 0; i < 3; i++ {
		allowed, _ := rl.allow("127.0.0.1", "test")
		if !allowed {
			t.Fatalf("request %d should be allowed", i+1)
		}
	}

	allowed, retryAfter := rl.allow("127.0.0.1", "test")
	if allowed {
		t.Fatal("4th request should be denied")
	}
	if retryAfter <= 0 {
		t.Fatal("retryAfter should be positive")
	}
}

func TestRateLimiter_ResetsAfterWindow(t *testing.T) {
	rl := &rateLimiter{
		buckets: make(map[string]*rateBucket),
		limits:  map[string]int{"test": 1},
		window:  100 * time.Millisecond,
	}

	allowed, _ := rl.allow("127.0.0.1", "test")
	if !allowed {
		t.Fatal("first request should be allowed")
	}

	allowed, _ = rl.allow("127.0.0.1", "test")
	if allowed {
		t.Fatal("second request should be denied")
	}

	time.Sleep(150 * time.Millisecond)

	allowed, _ = rl.allow("127.0.0.1", "test")
	if !allowed {
		t.Fatal("request after refill should be allowed")
	}
}

func TestRateLimiter_PerIPIsolation(t *testing.T) {
	rl := &rateLimiter{
		buckets: make(map[string]*rateBucket),
		limits:  map[string]int{"test": 1},
		window:  time.Minute,
	}

	allowed, _ := rl.allow("10.0.0.1", "test")
	if !allowed {
		t.Fatal("IP1 first request should be allowed")
	}

	allowed, _ = rl.allow("10.0.0.2", "test")
	if !allowed {
		t.Fatal("IP2 first request should be allowed (different IP)")
	}

	allowed, _ = rl.allow("10.0.0.1", "test")
	if allowed {
		t.Fatal("IP1 second request should be denied")
	}
}

func TestRateLimiter_PortChurnDoesNotBypass(t *testing.T) {
	rl := &rateLimiter{
		buckets: make(map[string]*rateBucket),
		limits:  map[string]int{"test": 1},
		window:  time.Minute,
	}

	if allowed, _ := rl.allow(clientHost("127.0.0.1:1234"), "test"); !allowed {
		t.Fatal("first request should be allowed")
	}
	if allowed, _ := rl.allow(clientHost("127.0.0.1:5678"), "test"); allowed {
		t.Fatal("changing source ports should not bypass the limiter")
	}
}

func TestClientHost_StripsPort(t *testing.T) {
	tests := map[string]string{
		"127.0.0.1:1234": "127.0.0.1",
		"[::1]:9222":     "::1",
		"10.0.0.8":       "10.0.0.8",
	}

	for input, want := range tests {
		if got := clientHost(input); got != want {
			t.Fatalf("clientHost(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestRateLimiter_UnconfiguredTierAllows(t *testing.T) {
	rl := &rateLimiter{
		buckets: make(map[string]*rateBucket),
		limits:  map[string]int{},
		window:  time.Minute,
	}

	for i := 0; i < 100; i++ {
		allowed, _ := rl.allow("127.0.0.1", "unknown_tier")
		if !allowed {
			t.Fatal("unconfigured tier should always allow")
		}
	}
}

func TestRateLimiter_Cleanup(t *testing.T) {
	rl := &rateLimiter{
		buckets: make(map[string]*rateBucket),
		limits:  map[string]int{"test": 1},
		window:  50 * time.Millisecond,
	}

	rl.allow("127.0.0.1", "test")
	if len(rl.buckets) != 1 {
		t.Fatal("should have 1 bucket")
	}

	time.Sleep(70 * time.Millisecond)
	rl.cleanup()

	if len(rl.buckets) != 1 {
		t.Fatal("bucket should remain until idle for more than 2 windows")
	}

	time.Sleep(70 * time.Millisecond)
	rl.cleanup()

	if len(rl.buckets) != 0 {
		t.Fatal("cleanup should remove stale buckets")
	}
}
