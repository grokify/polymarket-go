package errors

import (
	"errors"
	"fmt"
	"net/http"
	"testing"
	"time"
)

func TestSentinelErrors(t *testing.T) {
	tests := []struct {
		err  error
		want string
	}{
		{ErrNotFound, "not found"},
		{ErrUnauthorized, "unauthorized"},
		{ErrForbidden, "forbidden"},
		{ErrRateLimited, "rate limited"},
		{ErrCircuitOpen, "circuit breaker open"},
		{ErrRetryExhausted, "retry attempts exhausted"},
		{ErrTimeout, "operation timed out"},
	}

	for _, tt := range tests {
		if tt.err.Error() != tt.want {
			t.Errorf("%v.Error() = %q, want %q", tt.err, tt.err.Error(), tt.want)
		}
	}
}

func TestAPIError(t *testing.T) {
	err := &APIError{
		Service:    "Polymarket",
		Operation:  "GetMarkets",
		StatusCode: 500,
		Body:       "internal error",
	}

	t.Run("Error message", func(t *testing.T) {
		want := "Polymarket API error in GetMarkets: status 500: internal error"
		if err.Error() != want {
			t.Errorf("Error() = %q, want %q", err.Error(), want)
		}
	})

	t.Run("Error message without body", func(t *testing.T) {
		e := &APIError{Service: "Test", Operation: "Op", StatusCode: 404}
		want := "Test API error in Op: status 404"
		if e.Error() != want {
			t.Errorf("Error() = %q, want %q", e.Error(), want)
		}
	})

	t.Run("IsRetryable for 5xx", func(t *testing.T) {
		if !err.IsRetryable() {
			t.Error("500 error should be retryable")
		}
	})

	t.Run("IsRetryable for 429", func(t *testing.T) {
		e := &APIError{StatusCode: 429}
		if !e.IsRetryable() {
			t.Error("429 error should be retryable")
		}
	})

	t.Run("Not retryable for 400", func(t *testing.T) {
		e := &APIError{StatusCode: 400}
		if e.IsRetryable() {
			t.Error("400 error should not be retryable")
		}
	})

	t.Run("HTTPStatusCode", func(t *testing.T) {
		tests := []struct {
			code int
			want int
		}{
			{500, http.StatusBadGateway},
			{503, http.StatusBadGateway},
			{429, http.StatusTooManyRequests},
			{404, http.StatusNotFound},
			{401, http.StatusUnauthorized},
			{400, http.StatusBadGateway},
		}
		for _, tt := range tests {
			e := &APIError{StatusCode: tt.code}
			if got := e.HTTPStatusCode(); got != tt.want {
				t.Errorf("HTTPStatusCode() for %d = %d, want %d", tt.code, got, tt.want)
			}
		}
	})

	t.Run("Is ErrNotFound", func(t *testing.T) {
		e := &APIError{StatusCode: 404}
		if !errors.Is(e, ErrNotFound) {
			t.Error("404 APIError should be ErrNotFound")
		}
	})

	t.Run("Is ErrRateLimited", func(t *testing.T) {
		e := &APIError{StatusCode: 429}
		if !errors.Is(e, ErrRateLimited) {
			t.Error("429 APIError should be ErrRateLimited")
		}
	})

	t.Run("Unwrap", func(t *testing.T) {
		underlying := errors.New("underlying")
		e := &APIError{Err: underlying}
		if !errors.Is(e, underlying) {
			t.Error("should unwrap to underlying error")
		}
	})
}

func TestNetworkError(t *testing.T) {
	underlying := errors.New("connection refused")
	err := &NetworkError{
		Operation: "GetMarkets",
		Host:      "api.polymarket.com",
		Err:       underlying,
	}

	t.Run("Error message", func(t *testing.T) {
		want := "network error in GetMarkets to api.polymarket.com: connection refused"
		if err.Error() != want {
			t.Errorf("Error() = %q, want %q", err.Error(), want)
		}
	})

	t.Run("Error message without host", func(t *testing.T) {
		e := &NetworkError{Operation: "Op", Err: underlying}
		want := "network error in Op: connection refused"
		if e.Error() != want {
			t.Errorf("Error() = %q, want %q", e.Error(), want)
		}
	})

	t.Run("IsRetryable", func(t *testing.T) {
		if !err.IsRetryable() {
			t.Error("network errors should be retryable")
		}
	})

	t.Run("HTTPStatusCode", func(t *testing.T) {
		if err.HTTPStatusCode() != http.StatusBadGateway {
			t.Errorf("HTTPStatusCode() = %d, want %d", err.HTTPStatusCode(), http.StatusBadGateway)
		}
	})

	t.Run("Unwrap", func(t *testing.T) {
		if !errors.Is(err, underlying) {
			t.Error("should unwrap to underlying error")
		}
	})
}

func TestValidationError(t *testing.T) {
	err := &ValidationError{
		Field:  "tokenID",
		Value:  "",
		Reason: "cannot be empty",
	}

	t.Run("Error message with value", func(t *testing.T) {
		e := &ValidationError{Field: "price", Value: "-1", Reason: "must be positive"}
		want := `validation error for price="-1": must be positive`
		if e.Error() != want {
			t.Errorf("Error() = %q, want %q", e.Error(), want)
		}
	})

	t.Run("Error message without value", func(t *testing.T) {
		e := &ValidationError{Field: "tokenID", Reason: "is required"}
		want := "validation error for tokenID: is required"
		if e.Error() != want {
			t.Errorf("Error() = %q, want %q", e.Error(), want)
		}
	})

	t.Run("Not retryable", func(t *testing.T) {
		if err.IsRetryable() {
			t.Error("validation errors should not be retryable")
		}
	})

	t.Run("HTTPStatusCode", func(t *testing.T) {
		if err.HTTPStatusCode() != http.StatusBadRequest {
			t.Errorf("HTTPStatusCode() = %d, want %d", err.HTTPStatusCode(), http.StatusBadRequest)
		}
	})
}

func TestConfigurationError(t *testing.T) {
	err := &ConfigurationError{
		Component: "RAGStore",
		Setting:   "vectorIndex",
		Reason:    "is required",
	}

	t.Run("Error message with setting", func(t *testing.T) {
		want := "configuration error in RAGStore for vectorIndex: is required"
		if err.Error() != want {
			t.Errorf("Error() = %q, want %q", err.Error(), want)
		}
	})

	t.Run("Error message without setting", func(t *testing.T) {
		e := &ConfigurationError{Component: "LLM", Reason: "missing API key"}
		want := "configuration error in LLM: missing API key"
		if e.Error() != want {
			t.Errorf("Error() = %q, want %q", e.Error(), want)
		}
	})

	t.Run("Not retryable", func(t *testing.T) {
		if err.IsRetryable() {
			t.Error("configuration errors should not be retryable")
		}
	})

	t.Run("HTTPStatusCode", func(t *testing.T) {
		if err.HTTPStatusCode() != http.StatusInternalServerError {
			t.Errorf("HTTPStatusCode() = %d, want %d", err.HTTPStatusCode(), http.StatusInternalServerError)
		}
	})
}

func TestEmbeddingError(t *testing.T) {
	err := &EmbeddingError{
		Provider:    "OpenAI",
		Operation:   "EmbedBatch",
		Reason:      "quota exceeded",
		RateLimited: true,
	}

	t.Run("Error message", func(t *testing.T) {
		want := "OpenAI embedding error in EmbedBatch: quota exceeded"
		if err.Error() != want {
			t.Errorf("Error() = %q, want %q", err.Error(), want)
		}
	})

	t.Run("IsRetryable when rate limited", func(t *testing.T) {
		if !err.IsRetryable() {
			t.Error("rate limited embedding errors should be retryable")
		}
	})

	t.Run("Not retryable when not rate limited", func(t *testing.T) {
		e := &EmbeddingError{RateLimited: false}
		if e.IsRetryable() {
			t.Error("non-rate-limited embedding errors should not be retryable")
		}
	})

	t.Run("HTTPStatusCode when rate limited", func(t *testing.T) {
		if err.HTTPStatusCode() != http.StatusTooManyRequests {
			t.Errorf("HTTPStatusCode() = %d, want %d", err.HTTPStatusCode(), http.StatusTooManyRequests)
		}
	})

	t.Run("HTTPStatusCode when not rate limited", func(t *testing.T) {
		e := &EmbeddingError{RateLimited: false}
		if e.HTTPStatusCode() != http.StatusInternalServerError {
			t.Errorf("HTTPStatusCode() = %d, want %d", e.HTTPStatusCode(), http.StatusInternalServerError)
		}
	})

	t.Run("Is ErrRateLimited", func(t *testing.T) {
		if !errors.Is(err, ErrRateLimited) {
			t.Error("rate limited EmbeddingError should be ErrRateLimited")
		}
	})
}

func TestIndexError(t *testing.T) {
	err := &IndexError{
		Operation: "upsert",
		NodeID:    "market-123",
		Reason:    "vector dimension mismatch",
	}

	t.Run("Error message with node ID", func(t *testing.T) {
		want := "index error in upsert for node market-123: vector dimension mismatch"
		if err.Error() != want {
			t.Errorf("Error() = %q, want %q", err.Error(), want)
		}
	})

	t.Run("Error message without node ID", func(t *testing.T) {
		e := &IndexError{Operation: "search", Reason: "index not initialized"}
		want := "index error in search: index not initialized"
		if e.Error() != want {
			t.Errorf("Error() = %q, want %q", e.Error(), want)
		}
	})

	t.Run("IsRetryable", func(t *testing.T) {
		if !err.IsRetryable() {
			t.Error("index errors should be retryable")
		}
	})
}

func TestLLMError(t *testing.T) {
	err := &LLMError{
		Provider:    "Anthropic",
		Model:       "claude-sonnet-4-20250514",
		Operation:   "ChatCompletion",
		Reason:      "rate limit exceeded",
		RateLimited: true,
	}

	t.Run("Error message with model", func(t *testing.T) {
		want := "Anthropic LLM error with claude-sonnet-4-20250514 in ChatCompletion: rate limit exceeded"
		if err.Error() != want {
			t.Errorf("Error() = %q, want %q", err.Error(), want)
		}
	})

	t.Run("Error message without model", func(t *testing.T) {
		e := &LLMError{Provider: "OpenAI", Operation: "Complete", Reason: "failed"}
		want := "OpenAI LLM error in Complete: failed"
		if e.Error() != want {
			t.Errorf("Error() = %q, want %q", e.Error(), want)
		}
	})

	t.Run("Is ErrRateLimited", func(t *testing.T) {
		if !errors.Is(err, ErrRateLimited) {
			t.Error("rate limited LLMError should be ErrRateLimited")
		}
	})
}

func TestCircuitOpenError(t *testing.T) {
	lastFailure := time.Now().Add(-30 * time.Second)
	retryAfter := time.Now().Add(30 * time.Second)
	err := &CircuitOpenError{
		Service:     "Polymarket",
		LastFailure: lastFailure,
		RetryAfter:  retryAfter,
	}

	t.Run("Error message", func(t *testing.T) {
		msg := err.Error()
		if msg == "" {
			t.Error("Error() should not be empty")
		}
		if !containsString(msg, "circuit breaker open for Polymarket") {
			t.Errorf("Error() should mention service: %s", msg)
		}
	})

	t.Run("IsRetryable when time has passed", func(t *testing.T) {
		e := &CircuitOpenError{RetryAfter: time.Now().Add(-1 * time.Second)}
		if !e.IsRetryable() {
			t.Error("should be retryable after RetryAfter time")
		}
	})

	t.Run("Not retryable before RetryAfter", func(t *testing.T) {
		if err.IsRetryable() {
			t.Error("should not be retryable before RetryAfter time")
		}
	})

	t.Run("HTTPStatusCode", func(t *testing.T) {
		if err.HTTPStatusCode() != http.StatusServiceUnavailable {
			t.Errorf("HTTPStatusCode() = %d, want %d", err.HTTPStatusCode(), http.StatusServiceUnavailable)
		}
	})

	t.Run("Is ErrCircuitOpen", func(t *testing.T) {
		if !errors.Is(err, ErrCircuitOpen) {
			t.Error("CircuitOpenError should be ErrCircuitOpen")
		}
	})
}

func TestRetryExhaustedError(t *testing.T) {
	underlying := &APIError{Service: "Test", StatusCode: 503}
	err := &RetryExhaustedError{
		Operation: "GetMarkets",
		Attempts:  3,
		LastErr:   underlying,
	}

	t.Run("Error message", func(t *testing.T) {
		msg := err.Error()
		if !containsString(msg, "retry exhausted") {
			t.Errorf("Error() should mention retry exhausted: %s", msg)
		}
		if !containsString(msg, "3 attempts") {
			t.Errorf("Error() should mention attempts: %s", msg)
		}
	})

	t.Run("Not retryable", func(t *testing.T) {
		if err.IsRetryable() {
			t.Error("retry exhausted should not be retryable")
		}
	})

	t.Run("HTTPStatusCode from underlying", func(t *testing.T) {
		// Underlying is 503 which maps to BadGateway
		if err.HTTPStatusCode() != http.StatusBadGateway {
			t.Errorf("HTTPStatusCode() = %d, want %d", err.HTTPStatusCode(), http.StatusBadGateway)
		}
	})

	t.Run("Is ErrRetryExhausted", func(t *testing.T) {
		if !errors.Is(err, ErrRetryExhausted) {
			t.Error("RetryExhaustedError should be ErrRetryExhausted")
		}
	})

	t.Run("Unwrap to underlying", func(t *testing.T) {
		if !errors.Is(err, underlying) {
			t.Error("should unwrap to underlying error")
		}
	})
}

func TestTimeoutError(t *testing.T) {
	err := &TimeoutError{
		Operation: "ChatCompletion",
		Timeout:   30 * time.Second,
	}

	t.Run("Error message", func(t *testing.T) {
		want := "timeout in ChatCompletion after 30s"
		if err.Error() != want {
			t.Errorf("Error() = %q, want %q", err.Error(), want)
		}
	})

	t.Run("IsRetryable", func(t *testing.T) {
		if !err.IsRetryable() {
			t.Error("timeout errors should be retryable")
		}
	})

	t.Run("HTTPStatusCode", func(t *testing.T) {
		if err.HTTPStatusCode() != http.StatusGatewayTimeout {
			t.Errorf("HTTPStatusCode() = %d, want %d", err.HTTPStatusCode(), http.StatusGatewayTimeout)
		}
	})

	t.Run("Is ErrTimeout", func(t *testing.T) {
		if !errors.Is(err, ErrTimeout) {
			t.Error("TimeoutError should be ErrTimeout")
		}
	})
}

func TestSearchError(t *testing.T) {
	err := &SearchError{
		Engine:      "Serper",
		Query:       "Bitcoin price",
		Reason:      "quota exceeded",
		RateLimited: true,
	}

	t.Run("Error message", func(t *testing.T) {
		want := `Serper search error for query "Bitcoin price": quota exceeded`
		if err.Error() != want {
			t.Errorf("Error() = %q, want %q", err.Error(), want)
		}
	})

	t.Run("IsRetryable when rate limited", func(t *testing.T) {
		if !err.IsRetryable() {
			t.Error("rate limited search errors should be retryable")
		}
	})

	t.Run("Not retryable when not rate limited", func(t *testing.T) {
		e := &SearchError{RateLimited: false}
		if e.IsRetryable() {
			t.Error("non-rate-limited search errors should not be retryable")
		}
	})

	t.Run("Is ErrRateLimited", func(t *testing.T) {
		if !errors.Is(err, ErrRateLimited) {
			t.Error("rate limited SearchError should be ErrRateLimited")
		}
	})
}

func TestDependencyError(t *testing.T) {
	underlying := &APIError{StatusCode: 500}
	err := &DependencyError{
		Step:       "analyze",
		Dependency: "fetch_markets",
		Reason:     "upstream failure",
		Err:        underlying,
	}

	t.Run("Error message", func(t *testing.T) {
		want := "dependency error in step analyze: fetch_markets failed: upstream failure"
		if err.Error() != want {
			t.Errorf("Error() = %q, want %q", err.Error(), want)
		}
	})

	t.Run("IsRetryable from underlying", func(t *testing.T) {
		if !err.IsRetryable() {
			t.Error("should be retryable when underlying is retryable")
		}
	})

	t.Run("Not retryable when underlying is not", func(t *testing.T) {
		e := &DependencyError{Err: &ValidationError{}}
		if e.IsRetryable() {
			t.Error("should not be retryable when underlying is not")
		}
	})

	t.Run("Unwrap to underlying", func(t *testing.T) {
		if !errors.Is(err, underlying) {
			t.Error("should unwrap to underlying error")
		}
	})
}

func TestIsRetryableHelper(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"APIError 500", &APIError{StatusCode: 500}, true},
		{"APIError 400", &APIError{StatusCode: 400}, false},
		{"NetworkError", &NetworkError{}, true},
		{"ValidationError", &ValidationError{}, false},
		{"ConfigurationError", &ConfigurationError{}, false},
		{"TimeoutError", &TimeoutError{}, true},
		{"plain error", errors.New("plain"), false},
		{"wrapped retryable", fmt.Errorf("wrap: %w", &NetworkError{}), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsRetryable(tt.err); got != tt.want {
				t.Errorf("IsRetryable() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHTTPStatusCodeHelper(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want int
	}{
		{"APIError 404", &APIError{StatusCode: 404}, http.StatusNotFound},
		{"ValidationError", &ValidationError{}, http.StatusBadRequest},
		{"NetworkError", &NetworkError{}, http.StatusBadGateway},
		{"TimeoutError", &TimeoutError{}, http.StatusGatewayTimeout},
		{"plain error", errors.New("plain"), http.StatusInternalServerError},
		{"wrapped", fmt.Errorf("wrap: %w", &ValidationError{}), http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := HTTPStatusCode(tt.err); got != tt.want {
				t.Errorf("HTTPStatusCode() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestErrorsAs(t *testing.T) {
	// Test that errors.As works correctly with our types
	err := fmt.Errorf("wrapped: %w", &APIError{Service: "Test", StatusCode: 404})

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Error("errors.As should find APIError")
	}
	if apiErr.Service != "Test" {
		t.Errorf("apiErr.Service = %q, want Test", apiErr.Service)
	}
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsStringHelper(s, substr))
}

func containsStringHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
