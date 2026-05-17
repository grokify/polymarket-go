// Package resilience provides resilience patterns like retry and circuit breaker.
package resilience

import (
	"context"
	"math"
	"math/rand/v2"
	"time"
)

// RetryConfig configures retry behavior.
type RetryConfig struct {
	// MaxAttempts is the maximum number of attempts (default: 3).
	MaxAttempts int

	// InitialBackoff is the initial backoff duration (default: 100ms).
	InitialBackoff time.Duration

	// MaxBackoff is the maximum backoff duration (default: 30s).
	MaxBackoff time.Duration

	// BackoffMultiplier multiplies the backoff after each attempt (default: 2.0).
	BackoffMultiplier float64

	// Jitter adds randomness to backoff to prevent thundering herd (default: true).
	Jitter bool

	// RetryableFunc determines if an error should be retried.
	// If nil, all errors are retried.
	RetryableFunc func(error) bool
}

// DefaultRetryConfig returns a default retry configuration.
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts:       3,
		InitialBackoff:    100 * time.Millisecond,
		MaxBackoff:        30 * time.Second,
		BackoffMultiplier: 2.0,
		Jitter:            true,
	}
}

// Retry executes a function with retry logic.
func Retry[T any](ctx context.Context, cfg RetryConfig, fn func(ctx context.Context) (T, error)) (T, error) {
	if cfg.MaxAttempts <= 0 {
		cfg.MaxAttempts = 3
	}
	if cfg.InitialBackoff <= 0 {
		cfg.InitialBackoff = 100 * time.Millisecond
	}
	if cfg.MaxBackoff <= 0 {
		cfg.MaxBackoff = 30 * time.Second
	}
	if cfg.BackoffMultiplier <= 0 {
		cfg.BackoffMultiplier = 2.0
	}

	var lastErr error
	var zero T
	backoff := cfg.InitialBackoff

	for attempt := 0; attempt < cfg.MaxAttempts; attempt++ {
		result, err := fn(ctx)
		if err == nil {
			return result, nil
		}

		lastErr = err

		// Check if error is retryable
		if cfg.RetryableFunc != nil && !cfg.RetryableFunc(err) {
			return zero, err
		}

		// Don't sleep after the last attempt
		if attempt == cfg.MaxAttempts-1 {
			break
		}

		// Calculate backoff with jitter
		sleepDuration := backoff
		if cfg.Jitter {
			// #nosec G404 - jitter doesn't require cryptographic randomness
			jitter := time.Duration(rand.Float64() * float64(backoff) * 0.3)
			sleepDuration = backoff + jitter
		}

		// Wait or context cancelled
		select {
		case <-ctx.Done():
			return zero, ctx.Err()
		case <-time.After(sleepDuration):
		}

		// Increase backoff for next attempt
		backoff = time.Duration(float64(backoff) * cfg.BackoffMultiplier)
		if backoff > cfg.MaxBackoff {
			backoff = cfg.MaxBackoff
		}
	}

	return zero, lastErr
}

// RetryWithExponentialBackoff retries with exponential backoff using default config.
func RetryWithExponentialBackoff[T any](ctx context.Context, maxAttempts int, fn func(ctx context.Context) (T, error)) (T, error) {
	cfg := DefaultRetryConfig()
	cfg.MaxAttempts = maxAttempts
	return Retry(ctx, cfg, fn)
}

// CalculateBackoff calculates exponential backoff duration.
func CalculateBackoff(attempt int, initialBackoff time.Duration, multiplier float64, maxBackoff time.Duration) time.Duration {
	backoff := float64(initialBackoff) * math.Pow(multiplier, float64(attempt))
	if time.Duration(backoff) > maxBackoff {
		return maxBackoff
	}
	return time.Duration(backoff)
}
