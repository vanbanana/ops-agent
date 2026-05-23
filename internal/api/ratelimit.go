package api

import (
	"net/http"
	"sync"
	"time"
)

// RateLimiter implements a token-bucket style limiter per key (token/IP).
// Window: 1 minute, Limit: 60 requests.
type RateLimiter struct {
	mu      sync.Mutex
	buckets map[string]*bucket
	limit   int
	window  time.Duration
}

type bucket struct {
	count    int
	resetAt  time.Time
}

// NewRateLimiter creates a rate limiter with specified requests per window.
func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		buckets: make(map[string]*bucket),
		limit:   limit,
		window:  window,
	}
}

// Allow checks if a request from the given key is allowed.
// Returns true if allowed, false if rate-limited.
func (rl *RateLimiter) Allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	b, ok := rl.buckets[key]
	if !ok || now.After(b.resetAt) {
		// New window
		rl.buckets[key] = &bucket{count: 1, resetAt: now.Add(rl.window)}
		return true
	}

	if b.count >= rl.limit {
		return false
	}
	b.count++
	return true
}

// Remaining returns how many requests are left in the current window.
func (rl *RateLimiter) Remaining(key string) int {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	b, ok := rl.buckets[key]
	if !ok || now.After(b.resetAt) {
		return rl.limit
	}
	return rl.limit - b.count
}

// Middleware returns an HTTP middleware that rate-limits by Authorization header or IP.
func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Key: prefer token from Authorization header, fallback to IP
		key := r.Header.Get("Authorization")
		if key == "" {
			key = r.RemoteAddr
		}

		if !rl.Allow(key) {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Retry-After", "60")
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte(`{"code":429,"error":"rate limit exceeded: 60 req/min","error_code":"RATE_LIMIT_001"}`))
			return
		}

		w.Header().Set("X-RateLimit-Remaining", string(rune('0'+rl.Remaining(key))))
		next.ServeHTTP(w, r)
	})
}
