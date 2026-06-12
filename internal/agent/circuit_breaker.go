package agent

import (
	"errors"
	"sync"
	"time"
)

// ErrCircuitOpen is returned when the circuit breaker is open (failing fast).
var ErrCircuitOpen = errors.New("circuit breaker open: LLM API unavailable, retry after cooldown")

// CircuitState represents the state of a circuit breaker.
type CircuitState int

const (
	CircuitClosed   CircuitState = iota // Normal: requests pass through
	CircuitOpen                         // Tripped: requests fail fast
	CircuitHalfOpen                     // Testing: one request allowed
)

// CircuitBreaker implements the circuit breaker pattern for LLM calls.
type CircuitBreaker struct {
	mu        sync.Mutex
	state     CircuitState
	failures  int
	threshold int           // consecutive failures to trip (default 3)
	cooldown  time.Duration // how long to stay open (default 30s)
	openUntil time.Time
}

// NewCircuitBreaker creates a circuit breaker with given threshold and cooldown.
func NewCircuitBreaker(threshold int, cooldown time.Duration) *CircuitBreaker {
	if threshold <= 0 {
		threshold = 3
	}
	if cooldown <= 0 {
		cooldown = 30 * time.Second
	}
	return &CircuitBreaker{
		state:     CircuitClosed,
		threshold: threshold,
		cooldown:  cooldown,
	}
}

// Allow returns true if a request should be attempted.
func (cb *CircuitBreaker) Allow() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case CircuitClosed:
		return true
	case CircuitOpen:
		if time.Now().After(cb.openUntil) {
			cb.state = CircuitHalfOpen
			return true // Allow one probe request
		}
		return false
	case CircuitHalfOpen:
		return false // Already probing, don't allow more
	}
	return false
}

// RecordSuccess resets the breaker to closed state.
func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.failures = 0
	cb.state = CircuitClosed
}

// RecordFailure increments failure count and may trip the breaker.
func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.failures++
	if cb.failures >= cb.threshold {
		cb.state = CircuitOpen
		cb.openUntil = time.Now().Add(cb.cooldown)
	}
}

// State returns the current circuit state (for monitoring).
func (cb *CircuitBreaker) State() CircuitState {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	return cb.state
}

// IsOpen returns true if the breaker is open (failing fast).
func (cb *CircuitBreaker) IsOpen() bool {
	return cb.State() == CircuitOpen
}
