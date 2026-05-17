package resilience

import (
	"context"
	"errors"
	"sync"
	"time"
)

// CircuitState represents the circuit breaker state.
type CircuitState int

const (
	// StateClosed allows requests to pass through.
	StateClosed CircuitState = iota
	// StateOpen blocks all requests.
	StateOpen
	// StateHalfOpen allows limited requests to test recovery.
	StateHalfOpen
)

// String returns the string representation of the state.
func (s CircuitState) String() string {
	switch s {
	case StateClosed:
		return "closed"
	case StateOpen:
		return "open"
	case StateHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// ErrCircuitOpen is returned when the circuit is open.
var ErrCircuitOpen = errors.New("circuit breaker is open")

// CircuitBreakerConfig configures the circuit breaker.
type CircuitBreakerConfig struct {
	// FailureThreshold is the number of failures to open the circuit (default: 5).
	FailureThreshold int

	// SuccessThreshold is the number of successes in half-open to close (default: 2).
	SuccessThreshold int

	// Timeout is how long to wait before transitioning from open to half-open (default: 30s).
	Timeout time.Duration

	// OnStateChange is called when the state changes.
	OnStateChange func(from, to CircuitState)
}

// DefaultCircuitBreakerConfig returns a default circuit breaker configuration.
func DefaultCircuitBreakerConfig() CircuitBreakerConfig {
	return CircuitBreakerConfig{
		FailureThreshold: 5,
		SuccessThreshold: 2,
		Timeout:          30 * time.Second,
	}
}

// CircuitBreaker implements the circuit breaker pattern.
type CircuitBreaker struct {
	config CircuitBreakerConfig

	mu              sync.RWMutex
	state           CircuitState
	failureCount    int
	successCount    int
	lastFailureTime time.Time
}

// NewCircuitBreaker creates a new circuit breaker.
func NewCircuitBreaker(cfg CircuitBreakerConfig) *CircuitBreaker {
	if cfg.FailureThreshold <= 0 {
		cfg.FailureThreshold = 5
	}
	if cfg.SuccessThreshold <= 0 {
		cfg.SuccessThreshold = 2
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 30 * time.Second
	}

	return &CircuitBreaker{
		config: cfg,
		state:  StateClosed,
	}
}

// State returns the current state of the circuit breaker.
func (cb *CircuitBreaker) State() CircuitState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// Execute runs the function if the circuit allows it.
func (cb *CircuitBreaker) Execute(ctx context.Context, fn func(ctx context.Context) error) error {
	if !cb.allowRequest() {
		return ErrCircuitOpen
	}

	err := fn(ctx)
	cb.recordResult(err)
	return err
}

// ExecuteWithResult runs the function and returns a result if the circuit allows it.
func ExecuteWithResult[T any](cb *CircuitBreaker, ctx context.Context, fn func(ctx context.Context) (T, error)) (T, error) {
	var zero T
	if !cb.allowRequest() {
		return zero, ErrCircuitOpen
	}

	result, err := fn(ctx)
	cb.recordResult(err)
	return result, err
}

// allowRequest checks if a request should be allowed.
func (cb *CircuitBreaker) allowRequest() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case StateClosed:
		return true

	case StateOpen:
		// Check if timeout has elapsed to transition to half-open
		if time.Since(cb.lastFailureTime) >= cb.config.Timeout {
			cb.transitionTo(StateHalfOpen)
			return true
		}
		return false

	case StateHalfOpen:
		// In half-open, allow limited requests
		return true

	default:
		return false
	}
}

// recordResult records the result of a request.
func (cb *CircuitBreaker) recordResult(err error) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case StateClosed:
		if err != nil {
			cb.failureCount++
			cb.lastFailureTime = time.Now()
			if cb.failureCount >= cb.config.FailureThreshold {
				cb.transitionTo(StateOpen)
			}
		} else {
			// Reset failure count on success
			cb.failureCount = 0
		}

	case StateHalfOpen:
		if err != nil {
			// Failure in half-open: back to open
			cb.failureCount = cb.config.FailureThreshold
			cb.lastFailureTime = time.Now()
			cb.transitionTo(StateOpen)
		} else {
			cb.successCount++
			if cb.successCount >= cb.config.SuccessThreshold {
				cb.transitionTo(StateClosed)
			}
		}
	}
}

// transitionTo changes the circuit state.
func (cb *CircuitBreaker) transitionTo(newState CircuitState) {
	oldState := cb.state
	cb.state = newState

	// Reset counters on state change
	switch newState {
	case StateClosed:
		cb.failureCount = 0
		cb.successCount = 0
	case StateHalfOpen:
		cb.successCount = 0
	case StateOpen:
		cb.successCount = 0
	}

	if cb.config.OnStateChange != nil {
		// Call in goroutine to avoid holding lock
		go cb.config.OnStateChange(oldState, newState)
	}
}

// Reset resets the circuit breaker to closed state.
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.state = StateClosed
	cb.failureCount = 0
	cb.successCount = 0
}

// Counts returns the current failure and success counts.
func (cb *CircuitBreaker) Counts() (failures, successes int) {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.failureCount, cb.successCount
}
