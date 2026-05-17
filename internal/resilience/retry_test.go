package resilience

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestDefaultRetryConfig(t *testing.T) {
	cfg := DefaultRetryConfig()

	if cfg.MaxAttempts != 3 {
		t.Errorf("MaxAttempts = %d, want 3", cfg.MaxAttempts)
	}
	if cfg.InitialBackoff != 100*time.Millisecond {
		t.Errorf("InitialBackoff = %v, want 100ms", cfg.InitialBackoff)
	}
	if cfg.MaxBackoff != 30*time.Second {
		t.Errorf("MaxBackoff = %v, want 30s", cfg.MaxBackoff)
	}
	if cfg.BackoffMultiplier != 2.0 {
		t.Errorf("BackoffMultiplier = %f, want 2.0", cfg.BackoffMultiplier)
	}
	if !cfg.Jitter {
		t.Error("Jitter = false, want true")
	}
}

func TestRetrySuccess(t *testing.T) {
	ctx := context.Background()
	cfg := RetryConfig{
		MaxAttempts:    3,
		InitialBackoff: 1 * time.Millisecond,
	}

	attempts := 0
	result, err := Retry(ctx, cfg, func(ctx context.Context) (string, error) {
		attempts++
		return "success", nil
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "success" {
		t.Errorf("result = %q, want %q", result, "success")
	}
	if attempts != 1 {
		t.Errorf("attempts = %d, want 1", attempts)
	}
}

func TestRetryEventualSuccess(t *testing.T) {
	ctx := context.Background()
	cfg := RetryConfig{
		MaxAttempts:    5,
		InitialBackoff: 1 * time.Millisecond,
		Jitter:         false,
	}

	attempts := 0
	result, err := Retry(ctx, cfg, func(ctx context.Context) (int, error) {
		attempts++
		if attempts < 3 {
			return 0, errors.New("transient error")
		}
		return 42, nil
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != 42 {
		t.Errorf("result = %d, want 42", result)
	}
	if attempts != 3 {
		t.Errorf("attempts = %d, want 3", attempts)
	}
}

func TestRetryAllFail(t *testing.T) {
	ctx := context.Background()
	cfg := RetryConfig{
		MaxAttempts:    3,
		InitialBackoff: 1 * time.Millisecond,
		Jitter:         false,
	}

	expectedErr := errors.New("persistent error")
	attempts := 0
	_, err := Retry(ctx, cfg, func(ctx context.Context) (string, error) {
		attempts++
		return "", expectedErr
	})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != expectedErr.Error() {
		t.Errorf("error = %v, want %v", err, expectedErr)
	}
	if attempts != 3 {
		t.Errorf("attempts = %d, want 3", attempts)
	}
}

func TestRetryNonRetryableError(t *testing.T) {
	ctx := context.Background()
	nonRetryableErr := errors.New("non-retryable")
	retryableErr := errors.New("retryable")

	cfg := RetryConfig{
		MaxAttempts:    5,
		InitialBackoff: 1 * time.Millisecond,
		RetryableFunc: func(err error) bool {
			return err.Error() != "non-retryable"
		},
	}

	// Test non-retryable error stops immediately
	attempts := 0
	_, err := Retry(ctx, cfg, func(ctx context.Context) (string, error) {
		attempts++
		return "", nonRetryableErr
	})

	if err != nonRetryableErr {
		t.Errorf("error = %v, want %v", err, nonRetryableErr)
	}
	if attempts != 1 {
		t.Errorf("attempts = %d, want 1 (should not retry)", attempts)
	}

	// Test retryable error retries
	attempts = 0
	_, err = Retry(ctx, cfg, func(ctx context.Context) (string, error) {
		attempts++
		return "", retryableErr
	})

	if attempts != 5 {
		t.Errorf("attempts = %d, want 5 (should retry)", attempts)
	}
}

func TestRetryContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	cfg := RetryConfig{
		MaxAttempts:    10,
		InitialBackoff: 100 * time.Millisecond,
		Jitter:         false,
	}

	attempts := 0
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	_, err := Retry(ctx, cfg, func(ctx context.Context) (string, error) {
		attempts++
		return "", errors.New("keep failing")
	})

	if !errors.Is(err, context.Canceled) {
		t.Errorf("error = %v, want context.Canceled", err)
	}
	// Should have made at least 1 attempt before cancellation
	if attempts < 1 {
		t.Errorf("attempts = %d, want >= 1", attempts)
	}
}

func TestRetryDefaultsApplied(t *testing.T) {
	ctx := context.Background()
	// Empty config should use defaults
	cfg := RetryConfig{}

	attempts := 0
	_, _ = Retry(ctx, cfg, func(ctx context.Context) (string, error) {
		attempts++
		return "", errors.New("fail")
	})

	// Default MaxAttempts is 3
	if attempts != 3 {
		t.Errorf("attempts = %d, want 3 (default)", attempts)
	}
}

func TestCalculateBackoff(t *testing.T) {
	tests := []struct {
		name           string
		attempt        int
		initialBackoff time.Duration
		multiplier     float64
		maxBackoff     time.Duration
		want           time.Duration
	}{
		{
			name:           "first attempt",
			attempt:        0,
			initialBackoff: 100 * time.Millisecond,
			multiplier:     2.0,
			maxBackoff:     10 * time.Second,
			want:           100 * time.Millisecond,
		},
		{
			name:           "second attempt",
			attempt:        1,
			initialBackoff: 100 * time.Millisecond,
			multiplier:     2.0,
			maxBackoff:     10 * time.Second,
			want:           200 * time.Millisecond,
		},
		{
			name:           "third attempt",
			attempt:        2,
			initialBackoff: 100 * time.Millisecond,
			multiplier:     2.0,
			maxBackoff:     10 * time.Second,
			want:           400 * time.Millisecond,
		},
		{
			name:           "capped at max",
			attempt:        10,
			initialBackoff: 100 * time.Millisecond,
			multiplier:     2.0,
			maxBackoff:     1 * time.Second,
			want:           1 * time.Second,
		},
		{
			name:           "multiplier 3x",
			attempt:        2,
			initialBackoff: 50 * time.Millisecond,
			multiplier:     3.0,
			maxBackoff:     10 * time.Second,
			want:           450 * time.Millisecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalculateBackoff(tt.attempt, tt.initialBackoff, tt.multiplier, tt.maxBackoff)
			if got != tt.want {
				t.Errorf("CalculateBackoff() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRetryWithExponentialBackoff(t *testing.T) {
	ctx := context.Background()

	attempts := 0
	result, err := RetryWithExponentialBackoff(ctx, 2, func(ctx context.Context) (string, error) {
		attempts++
		if attempts < 2 {
			return "", errors.New("fail")
		}
		return "ok", nil
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "ok" {
		t.Errorf("result = %q, want %q", result, "ok")
	}
	if attempts != 2 {
		t.Errorf("attempts = %d, want 2", attempts)
	}
}
