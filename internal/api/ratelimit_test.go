package api

import (
	"testing"
	"time"
)

func TestRateLimiterAllow(t *testing.T) {
	rl := NewRateLimiter(5, 1*time.Minute)

	// First 5 requests should pass
	for i := 0; i < 5; i++ {
		if !rl.Allow("user1") {
			t.Fatalf("request %d should be allowed", i+1)
		}
	}

	// 6th should be blocked
	if rl.Allow("user1") {
		t.Fatal("6th request should be rate-limited")
	}

	// Different key should still be allowed
	if !rl.Allow("user2") {
		t.Fatal("different key should not be affected")
	}

	t.Log("✅ Rate limiter correctly blocks after limit reached")
}

func TestRateLimiterWindowReset(t *testing.T) {
	rl := NewRateLimiter(2, 50*time.Millisecond)

	// Use up limit
	rl.Allow("key")
	rl.Allow("key")
	if rl.Allow("key") {
		t.Fatal("should be blocked")
	}

	// Wait for window to expire
	time.Sleep(60 * time.Millisecond)

	// Should be allowed again
	if !rl.Allow("key") {
		t.Fatal("should be allowed after window reset")
	}

	t.Log("✅ Rate limiter resets after window expires")
}

func TestRateLimiterRemaining(t *testing.T) {
	rl := NewRateLimiter(10, 1*time.Minute)

	if rl.Remaining("fresh") != 10 {
		t.Fatalf("expected 10 remaining for fresh key, got %d", rl.Remaining("fresh"))
	}

	rl.Allow("used")
	rl.Allow("used")
	rl.Allow("used")

	if rl.Remaining("used") != 7 {
		t.Fatalf("expected 7 remaining after 3 uses, got %d", rl.Remaining("used"))
	}
}
