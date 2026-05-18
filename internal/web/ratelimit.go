package web

import (
	"math"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"
)

// rateBucket tracks available tokens for one client+tier.
type rateBucket struct {
	tokens     float64
	lastRefill time.Time
}

// rateLimiter implements a simple per-IP token bucket rate limiter.
// Zero external dependencies; uses sync.Mutex for concurrent access.
type rateLimiter struct {
	mu      sync.Mutex
	buckets map[string]*rateBucket // key: "host:tier"
	limits  map[string]int         // tier -> max burst / requests per window
	window  time.Duration
}

// newRateLimiter creates a rate limiter with the given window duration.
func newRateLimiter(window time.Duration) *rateLimiter {
	rl := &rateLimiter{
		buckets: make(map[string]*rateBucket),
		limits:  make(map[string]int),
		window:  window,
	}

	// Background cleanup of stale buckets (every 5 minutes).
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			rl.cleanup()
		}
	}()

	return rl
}

// setLimit configures the max burst size / refill budget per window for a tier.
func (rl *rateLimiter) setLimit(tier string, maxRequests int) {
	rl.limits[tier] = maxRequests
}

// allow checks if a request from the given client host in the given tier is allowed.
// Returns (allowed bool, retryAfterSec int).
func (rl *rateLimiter) allow(host, tier string) (bool, int) {
	key := host + ":" + tier
	limit, ok := rl.limits[tier]
	if !ok {
		return true, 0
	}

	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	bucket, exists := rl.buckets[key]
	if !exists {
		rl.buckets[key] = &rateBucket{
			tokens:     float64(limit - 1),
			lastRefill: now,
		}
		return true, 0
	}

	refillRate := float64(limit) / rl.window.Seconds()
	elapsed := now.Sub(bucket.lastRefill)
	bucket.lastRefill = now
	bucket.tokens = math.Min(float64(limit), bucket.tokens+(elapsed.Seconds()*refillRate))

	if bucket.tokens < 1 {
		missing := 1 - bucket.tokens
		retryAfter := int(math.Ceil(missing / refillRate))
		if retryAfter < 1 {
			retryAfter = 1
		}
		return false, retryAfter
	}

	bucket.tokens--
	return true, 0
}

// cleanup removes buckets that have been idle long enough to fully refill twice.
func (rl *rateLimiter) cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	maxIdle := rl.window * 2
	for key, bucket := range rl.buckets {
		if now.Sub(bucket.lastRefill) > maxIdle {
			delete(rl.buckets, key)
		}
	}
}

func clientHost(remoteAddr string) string {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err == nil {
		return host
	}
	return remoteAddr
}

// rateMiddleware returns HTTP middleware that enforces rate limiting for the given tier.
func (rl *rateLimiter) rateMiddleware(tier string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		host := clientHost(r.RemoteAddr)
		allowed, retryAfter := rl.allow(host, tier)
		if !allowed {
			w.Header().Set("Retry-After", strconv.Itoa(retryAfter))
			jsonError(w, "rate limit exceeded, retry after "+strconv.Itoa(retryAfter)+"s", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	}
}
