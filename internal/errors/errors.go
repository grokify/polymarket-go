// Package errors provides structured error types for the polymarket-go project.
//
// These error types enable:
//   - Programmatic error handling with errors.Is() and errors.As()
//   - Rich context for logging and debugging
//   - Appropriate HTTP status code mapping
//   - Retry/circuit breaker integration via IsRetryable()
package errors

import (
	"errors"
	"fmt"
	"net/http"
	"time"
)

// Sentinel errors for common conditions.
var (
	// ErrNotFound indicates a requested resource was not found.
	ErrNotFound = errors.New("not found")

	// ErrUnauthorized indicates missing or invalid authentication.
	ErrUnauthorized = errors.New("unauthorized")

	// ErrForbidden indicates the request is not allowed.
	ErrForbidden = errors.New("forbidden")

	// ErrRateLimited indicates too many requests.
	ErrRateLimited = errors.New("rate limited")

	// ErrCircuitOpen indicates the circuit breaker is open.
	ErrCircuitOpen = errors.New("circuit breaker open")

	// ErrRetryExhausted indicates all retry attempts failed.
	ErrRetryExhausted = errors.New("retry attempts exhausted")

	// ErrTimeout indicates an operation timed out.
	ErrTimeout = errors.New("operation timed out")
)

// Retryable is an interface for errors that know if they should be retried.
type Retryable interface {
	IsRetryable() bool
}

// IsRetryable returns true if the error can be retried.
func IsRetryable(err error) bool {
	var r Retryable
	if errors.As(err, &r) {
		return r.IsRetryable()
	}
	return false
}

// HTTPStatusCoder is an interface for errors that map to HTTP status codes.
type HTTPStatusCoder interface {
	HTTPStatusCode() int
}

// HTTPStatusCode returns the appropriate HTTP status code for an error.
// Returns 500 if the error doesn't implement HTTPStatusCoder.
func HTTPStatusCode(err error) int {
	var h HTTPStatusCoder
	if errors.As(err, &h) {
		return h.HTTPStatusCode()
	}
	return http.StatusInternalServerError
}

// APIError represents an error from an external API.
type APIError struct {
	// Service is the name of the API service (e.g., "Polymarket", "OpenAI").
	Service string
	// Operation is the API operation that failed (e.g., "GetMarkets").
	Operation string
	// StatusCode is the HTTP status code returned.
	StatusCode int
	// Body is the response body (truncated for large responses).
	Body string
	// Err is the underlying error, if any.
	Err error
}

func (e *APIError) Error() string {
	if e.Body != "" {
		return fmt.Sprintf("%s API error in %s: status %d: %s", e.Service, e.Operation, e.StatusCode, e.Body)
	}
	return fmt.Sprintf("%s API error in %s: status %d", e.Service, e.Operation, e.StatusCode)
}

func (e *APIError) Unwrap() error {
	return e.Err
}

func (e *APIError) IsRetryable() bool {
	// 5xx errors and 429 (rate limit) are generally retryable
	return e.StatusCode >= 500 || e.StatusCode == http.StatusTooManyRequests
}

func (e *APIError) HTTPStatusCode() int {
	// Map external API errors to appropriate client-facing codes
	switch {
	case e.StatusCode == http.StatusTooManyRequests:
		return http.StatusTooManyRequests
	case e.StatusCode >= 500:
		return http.StatusBadGateway // Upstream server error
	case e.StatusCode == http.StatusNotFound:
		return http.StatusNotFound
	case e.StatusCode == http.StatusUnauthorized:
		return http.StatusUnauthorized
	default:
		return http.StatusBadGateway
	}
}

// Is implements errors.Is for APIError.
func (e *APIError) Is(target error) bool {
	switch {
	case e.StatusCode == http.StatusNotFound && errors.Is(target, ErrNotFound):
		return true
	case e.StatusCode == http.StatusUnauthorized && errors.Is(target, ErrUnauthorized):
		return true
	case e.StatusCode == http.StatusForbidden && errors.Is(target, ErrForbidden):
		return true
	case e.StatusCode == http.StatusTooManyRequests && errors.Is(target, ErrRateLimited):
		return true
	}
	return false
}

// NetworkError represents a network-level failure.
type NetworkError struct {
	// Operation describes what was being attempted.
	Operation string
	// Host is the target host, if known.
	Host string
	// Err is the underlying error.
	Err error
}

func (e *NetworkError) Error() string {
	if e.Host != "" {
		return fmt.Sprintf("network error in %s to %s: %v", e.Operation, e.Host, e.Err)
	}
	return fmt.Sprintf("network error in %s: %v", e.Operation, e.Err)
}

func (e *NetworkError) Unwrap() error {
	return e.Err
}

func (e *NetworkError) IsRetryable() bool {
	return true // Network errors are generally retryable
}

func (e *NetworkError) HTTPStatusCode() int {
	return http.StatusBadGateway
}

// ValidationError represents an input validation failure.
type ValidationError struct {
	// Field is the name of the invalid field.
	Field string
	// Value is the invalid value (as string for logging).
	Value string
	// Reason explains why the value is invalid.
	Reason string
	// Err is the underlying error, if any.
	Err error
}

func (e *ValidationError) Error() string {
	if e.Value != "" {
		return fmt.Sprintf("validation error for %s=%q: %s", e.Field, e.Value, e.Reason)
	}
	return fmt.Sprintf("validation error for %s: %s", e.Field, e.Reason)
}

func (e *ValidationError) Unwrap() error {
	return e.Err
}

func (e *ValidationError) IsRetryable() bool {
	return false // Validation errors won't succeed on retry
}

func (e *ValidationError) HTTPStatusCode() int {
	return http.StatusBadRequest
}

// ConfigurationError represents a configuration or initialization failure.
type ConfigurationError struct {
	// Component is the component being configured.
	Component string
	// Setting is the specific setting that failed.
	Setting string
	// Reason explains what's wrong.
	Reason string
	// Err is the underlying error, if any.
	Err error
}

func (e *ConfigurationError) Error() string {
	if e.Setting != "" {
		return fmt.Sprintf("configuration error in %s for %s: %s", e.Component, e.Setting, e.Reason)
	}
	return fmt.Sprintf("configuration error in %s: %s", e.Component, e.Reason)
}

func (e *ConfigurationError) Unwrap() error {
	return e.Err
}

func (e *ConfigurationError) IsRetryable() bool {
	return false // Config errors require code/config changes
}

func (e *ConfigurationError) HTTPStatusCode() int {
	return http.StatusInternalServerError
}

// EmbeddingError represents a failure in text embedding.
type EmbeddingError struct {
	// Provider is the embedding provider (e.g., "OpenAI").
	Provider string
	// Operation describes what was being embedded.
	Operation string
	// Reason explains the failure.
	Reason string
	// IsRateLimited indicates if this is a rate limit error.
	RateLimited bool
	// Err is the underlying error, if any.
	Err error
}

func (e *EmbeddingError) Error() string {
	return fmt.Sprintf("%s embedding error in %s: %s", e.Provider, e.Operation, e.Reason)
}

func (e *EmbeddingError) Unwrap() error {
	return e.Err
}

func (e *EmbeddingError) IsRetryable() bool {
	return e.RateLimited // Rate limited errors can be retried after backoff
}

func (e *EmbeddingError) HTTPStatusCode() int {
	if e.RateLimited {
		return http.StatusTooManyRequests
	}
	return http.StatusInternalServerError
}

func (e *EmbeddingError) Is(target error) bool {
	return e.RateLimited && errors.Is(target, ErrRateLimited)
}

// IndexError represents a vector index operation failure.
type IndexError struct {
	// Operation is the index operation (e.g., "upsert", "search").
	Operation string
	// NodeID is the ID of the affected node, if applicable.
	NodeID string
	// Reason explains the failure.
	Reason string
	// Err is the underlying error, if any.
	Err error
}

func (e *IndexError) Error() string {
	if e.NodeID != "" {
		return fmt.Sprintf("index error in %s for node %s: %s", e.Operation, e.NodeID, e.Reason)
	}
	return fmt.Sprintf("index error in %s: %s", e.Operation, e.Reason)
}

func (e *IndexError) Unwrap() error {
	return e.Err
}

func (e *IndexError) IsRetryable() bool {
	return true // Index operations can often be retried
}

func (e *IndexError) HTTPStatusCode() int {
	return http.StatusInternalServerError
}

// LLMError represents a failure from an LLM provider.
type LLMError struct {
	// Provider is the LLM provider (e.g., "Anthropic", "OpenAI").
	Provider string
	// Model is the model that was used.
	Model string
	// Operation describes what was attempted.
	Operation string
	// Reason explains the failure.
	Reason string
	// RateLimited indicates if this is a rate limit error.
	RateLimited bool
	// Err is the underlying error, if any.
	Err error
}

func (e *LLMError) Error() string {
	if e.Model != "" {
		return fmt.Sprintf("%s LLM error with %s in %s: %s", e.Provider, e.Model, e.Operation, e.Reason)
	}
	return fmt.Sprintf("%s LLM error in %s: %s", e.Provider, e.Operation, e.Reason)
}

func (e *LLMError) Unwrap() error {
	return e.Err
}

func (e *LLMError) IsRetryable() bool {
	return e.RateLimited
}

func (e *LLMError) HTTPStatusCode() int {
	if e.RateLimited {
		return http.StatusTooManyRequests
	}
	return http.StatusInternalServerError
}

func (e *LLMError) Is(target error) bool {
	return e.RateLimited && errors.Is(target, ErrRateLimited)
}

// CircuitOpenError indicates a circuit breaker prevented the operation.
type CircuitOpenError struct {
	// Service is the protected service name.
	Service string
	// LastFailure is when the circuit opened.
	LastFailure time.Time
	// RetryAfter is when the circuit may allow requests again.
	RetryAfter time.Time
}

func (e *CircuitOpenError) Error() string {
	return fmt.Sprintf("circuit breaker open for %s since %s, retry after %s",
		e.Service, e.LastFailure.Format(time.RFC3339), e.RetryAfter.Format(time.RFC3339))
}

func (e *CircuitOpenError) IsRetryable() bool {
	return time.Now().After(e.RetryAfter)
}

func (e *CircuitOpenError) HTTPStatusCode() int {
	return http.StatusServiceUnavailable
}

func (e *CircuitOpenError) Is(target error) bool {
	return errors.Is(target, ErrCircuitOpen)
}

// RetryExhaustedError indicates all retry attempts have failed.
type RetryExhaustedError struct {
	// Operation is what was being retried.
	Operation string
	// Attempts is the number of attempts made.
	Attempts int
	// LastErr is the error from the final attempt.
	LastErr error
}

func (e *RetryExhaustedError) Error() string {
	return fmt.Sprintf("retry exhausted for %s after %d attempts: %v", e.Operation, e.Attempts, e.LastErr)
}

func (e *RetryExhaustedError) Unwrap() error {
	return e.LastErr
}

func (e *RetryExhaustedError) IsRetryable() bool {
	return false // Already retried max times
}

func (e *RetryExhaustedError) HTTPStatusCode() int {
	// Use the underlying error's status code if available
	return HTTPStatusCode(e.LastErr)
}

func (e *RetryExhaustedError) Is(target error) bool {
	return errors.Is(target, ErrRetryExhausted)
}

// TimeoutError indicates an operation timed out.
type TimeoutError struct {
	// Operation is what timed out.
	Operation string
	// Timeout is the duration that was exceeded.
	Timeout time.Duration
	// Err is the underlying error (often context.DeadlineExceeded).
	Err error
}

func (e *TimeoutError) Error() string {
	return fmt.Sprintf("timeout in %s after %s", e.Operation, e.Timeout)
}

func (e *TimeoutError) Unwrap() error {
	return e.Err
}

func (e *TimeoutError) IsRetryable() bool {
	return true // Timeouts can often be retried
}

func (e *TimeoutError) HTTPStatusCode() int {
	return http.StatusGatewayTimeout
}

func (e *TimeoutError) Is(target error) bool {
	return errors.Is(target, ErrTimeout)
}

// SearchError represents a search operation failure.
type SearchError struct {
	// Engine is the search engine (e.g., "Serper", "SerpAPI").
	Engine string
	// Query is the search query.
	Query string
	// Reason explains the failure.
	Reason string
	// RateLimited indicates if this is a rate limit error.
	RateLimited bool
	// Err is the underlying error, if any.
	Err error
}

func (e *SearchError) Error() string {
	return fmt.Sprintf("%s search error for query %q: %s", e.Engine, e.Query, e.Reason)
}

func (e *SearchError) Unwrap() error {
	return e.Err
}

func (e *SearchError) IsRetryable() bool {
	return e.RateLimited
}

func (e *SearchError) HTTPStatusCode() int {
	if e.RateLimited {
		return http.StatusTooManyRequests
	}
	return http.StatusBadGateway
}

func (e *SearchError) Is(target error) bool {
	return e.RateLimited && errors.Is(target, ErrRateLimited)
}

// DependencyError represents a workflow dependency failure.
type DependencyError struct {
	// Step is the step that has the dependency.
	Step string
	// Dependency is the name of the failed dependency.
	Dependency string
	// Reason explains the failure.
	Reason string
	// Err is the underlying error, if any.
	Err error
}

func (e *DependencyError) Error() string {
	return fmt.Sprintf("dependency error in step %s: %s failed: %s", e.Step, e.Dependency, e.Reason)
}

func (e *DependencyError) Unwrap() error {
	return e.Err
}

func (e *DependencyError) IsRetryable() bool {
	// Check if the underlying error is retryable
	return IsRetryable(e.Err)
}

func (e *DependencyError) HTTPStatusCode() int {
	return http.StatusInternalServerError
}
