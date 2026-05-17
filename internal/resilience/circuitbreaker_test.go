package resilience

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

func TestCircuitStateString(t *testing.T) {
	tests := []struct {
		state CircuitState
		want  string
	}{
		{StateClosed, "closed"},
		{StateOpen, "open"},
		{StateHalfOpen, "half-open"},
		{CircuitState(99), "unknown"},
	}

	for _, tt := range tests {
		if got := tt.state.String(); got != tt.want {
			t.Errorf("CircuitState(%d).String() = %q, want %q", tt.state, got, tt.want)
		}
	}
}

func TestDefaultCircuitBreakerConfig(t *testing.T) {
	cfg := DefaultCircuitBreakerConfig()

	if cfg.FailureThreshold != 5 {
		t.Errorf("FailureThreshold = %d, want 5", cfg.FailureThreshold)
	}
	if cfg.SuccessThreshold != 2 {
		t.Errorf("SuccessThreshold = %d, want 2", cfg.SuccessThreshold)
	}
	if cfg.Timeout != 30*time.Second {
		t.Errorf("Timeout = %v, want 30s", cfg.Timeout)
	}
}

func TestNewCircuitBreakerDefaults(t *testing.T) {
	// Empty config should use defaults
	cb := NewCircuitBreaker(CircuitBreakerConfig{})

	if cb.config.FailureThreshold != 5 {
		t.Errorf("FailureThreshold = %d, want 5", cb.config.FailureThreshold)
	}
	if cb.config.SuccessThreshold != 2 {
		t.Errorf("SuccessThreshold = %d, want 2", cb.config.SuccessThreshold)
	}
	if cb.config.Timeout != 30*time.Second {
		t.Errorf("Timeout = %v, want 30s", cb.config.Timeout)
	}
	if cb.State() != StateClosed {
		t.Errorf("initial state = %v, want closed", cb.State())
	}
}

func TestCircuitBreakerClosedState(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		FailureThreshold: 3,
	})

	ctx := context.Background()

	// Successful calls should work
	err := cb.Execute(ctx, func(ctx context.Context) error {
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cb.State() != StateClosed {
		t.Errorf("state = %v, want closed", cb.State())
	}

	failures, successes := cb.Counts()
	if failures != 0 {
		t.Errorf("failures = %d, want 0", failures)
	}
	if successes != 0 {
		// Note: success count is only tracked in half-open state
		t.Logf("successes = %d (only tracked in half-open)", successes)
	}
}

func TestCircuitBreakerOpensOnFailures(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		FailureThreshold: 3,
		Timeout:          1 * time.Second,
	})

	ctx := context.Background()
	testErr := errors.New("test error")

	// Fail 3 times to open the circuit
	for i := 0; i < 3; i++ {
		err := cb.Execute(ctx, func(ctx context.Context) error {
			return testErr
		})
		if err != testErr {
			t.Errorf("iteration %d: error = %v, want %v", i, err, testErr)
		}
	}

	if cb.State() != StateOpen {
		t.Errorf("state = %v, want open", cb.State())
	}

	// Next call should be rejected
	err := cb.Execute(ctx, func(ctx context.Context) error {
		t.Error("function should not be called when circuit is open")
		return nil
	})
	if !errors.Is(err, ErrCircuitOpen) {
		t.Errorf("error = %v, want ErrCircuitOpen", err)
	}
}

func TestCircuitBreakerResetsOnSuccess(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		FailureThreshold: 3,
	})

	ctx := context.Background()
	testErr := errors.New("test error")

	// Fail twice
	for i := 0; i < 2; i++ {
		_ = cb.Execute(ctx, func(ctx context.Context) error {
			return testErr
		})
	}

	failures, _ := cb.Counts()
	if failures != 2 {
		t.Errorf("failures = %d, want 2", failures)
	}

	// Success should reset failure count
	_ = cb.Execute(ctx, func(ctx context.Context) error {
		return nil
	})

	failures, _ = cb.Counts()
	if failures != 0 {
		t.Errorf("failures after success = %d, want 0", failures)
	}
	if cb.State() != StateClosed {
		t.Errorf("state = %v, want closed", cb.State())
	}
}

func TestCircuitBreakerHalfOpenState(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		FailureThreshold: 2,
		SuccessThreshold: 2,
		Timeout:          50 * time.Millisecond,
	})

	ctx := context.Background()
	testErr := errors.New("test error")

	// Open the circuit
	for i := 0; i < 2; i++ {
		_ = cb.Execute(ctx, func(ctx context.Context) error {
			return testErr
		})
	}

	if cb.State() != StateOpen {
		t.Fatalf("state = %v, want open", cb.State())
	}

	// Wait for timeout to transition to half-open
	time.Sleep(60 * time.Millisecond)

	// Next call should be allowed (half-open)
	err := cb.Execute(ctx, func(ctx context.Context) error {
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error in half-open: %v", err)
	}

	if cb.State() != StateHalfOpen {
		t.Errorf("state = %v, want half-open", cb.State())
	}
}

func TestCircuitBreakerHalfOpenToClosedOnSuccess(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		FailureThreshold: 2,
		SuccessThreshold: 2,
		Timeout:          10 * time.Millisecond,
	})

	ctx := context.Background()

	// Open the circuit
	for i := 0; i < 2; i++ {
		_ = cb.Execute(ctx, func(ctx context.Context) error {
			return errors.New("fail")
		})
	}

	// Wait for timeout
	time.Sleep(20 * time.Millisecond)

	// Two successes should close the circuit
	for i := 0; i < 2; i++ {
		err := cb.Execute(ctx, func(ctx context.Context) error {
			return nil
		})
		if err != nil {
			t.Fatalf("iteration %d: unexpected error: %v", i, err)
		}
	}

	if cb.State() != StateClosed {
		t.Errorf("state = %v, want closed", cb.State())
	}
}

func TestCircuitBreakerHalfOpenToOpenOnFailure(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		FailureThreshold: 2,
		SuccessThreshold: 3,
		Timeout:          10 * time.Millisecond,
	})

	ctx := context.Background()

	// Open the circuit
	for i := 0; i < 2; i++ {
		_ = cb.Execute(ctx, func(ctx context.Context) error {
			return errors.New("fail")
		})
	}

	// Wait for timeout
	time.Sleep(20 * time.Millisecond)

	// One success
	_ = cb.Execute(ctx, func(ctx context.Context) error {
		return nil
	})

	if cb.State() != StateHalfOpen {
		t.Fatalf("state = %v, want half-open", cb.State())
	}

	// Failure in half-open should go back to open
	_ = cb.Execute(ctx, func(ctx context.Context) error {
		return errors.New("fail again")
	})

	if cb.State() != StateOpen {
		t.Errorf("state = %v, want open", cb.State())
	}
}

func TestCircuitBreakerReset(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		FailureThreshold: 2,
	})

	ctx := context.Background()

	// Open the circuit
	for i := 0; i < 2; i++ {
		_ = cb.Execute(ctx, func(ctx context.Context) error {
			return errors.New("fail")
		})
	}

	if cb.State() != StateOpen {
		t.Fatalf("state = %v, want open", cb.State())
	}

	// Reset should close the circuit
	cb.Reset()

	if cb.State() != StateClosed {
		t.Errorf("state after reset = %v, want closed", cb.State())
	}

	failures, successes := cb.Counts()
	if failures != 0 || successes != 0 {
		t.Errorf("counts after reset = (%d, %d), want (0, 0)", failures, successes)
	}
}

func TestCircuitBreakerOnStateChange(t *testing.T) {
	var mu sync.Mutex
	var transitions []struct{ from, to CircuitState }

	cb := NewCircuitBreaker(CircuitBreakerConfig{
		FailureThreshold: 2,
		SuccessThreshold: 1,
		Timeout:          10 * time.Millisecond,
		OnStateChange: func(from, to CircuitState) {
			mu.Lock()
			transitions = append(transitions, struct{ from, to CircuitState }{from, to})
			mu.Unlock()
		},
	})

	ctx := context.Background()

	// Open the circuit
	for i := 0; i < 2; i++ {
		_ = cb.Execute(ctx, func(ctx context.Context) error {
			return errors.New("fail")
		})
	}

	// Wait for callback
	time.Sleep(10 * time.Millisecond)

	mu.Lock()
	if len(transitions) != 1 {
		t.Errorf("transitions count = %d, want 1", len(transitions))
	} else {
		if transitions[0].from != StateClosed || transitions[0].to != StateOpen {
			t.Errorf("transition = %v->%v, want closed->open", transitions[0].from, transitions[0].to)
		}
	}
	mu.Unlock()
}

func TestExecuteWithResult(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		FailureThreshold: 5,
	})

	ctx := context.Background()

	// Test success
	result, err := ExecuteWithResult(cb, ctx, func(ctx context.Context) (int, error) {
		return 42, nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != 42 {
		t.Errorf("result = %d, want 42", result)
	}

	// Test failure
	_, err = ExecuteWithResult(cb, ctx, func(ctx context.Context) (int, error) {
		return 0, errors.New("fail")
	})
	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestExecuteWithResultCircuitOpen(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		FailureThreshold: 1,
		Timeout:          1 * time.Hour,
	})

	ctx := context.Background()

	// Open the circuit
	_, _ = ExecuteWithResult(cb, ctx, func(ctx context.Context) (string, error) {
		return "", errors.New("fail")
	})

	// Should be rejected
	result, err := ExecuteWithResult(cb, ctx, func(ctx context.Context) (string, error) {
		t.Error("should not be called")
		return "bad", nil
	})

	if !errors.Is(err, ErrCircuitOpen) {
		t.Errorf("error = %v, want ErrCircuitOpen", err)
	}
	if result != "" {
		t.Errorf("result = %q, want empty string", result)
	}
}

func TestCircuitBreakerConcurrency(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		FailureThreshold: 100,
		SuccessThreshold: 10,
		Timeout:          1 * time.Second,
	})

	ctx := context.Background()
	var wg sync.WaitGroup

	// Run concurrent requests
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			_ = cb.Execute(ctx, func(ctx context.Context) error {
				if n%2 == 0 {
					return errors.New("fail")
				}
				return nil
			})
		}(i)
	}

	wg.Wait()

	// Should not panic and state should be valid
	state := cb.State()
	if state != StateClosed && state != StateOpen && state != StateHalfOpen {
		t.Errorf("invalid state: %v", state)
	}
}
