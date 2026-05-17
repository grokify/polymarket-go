# ADR-0006: Implement Resilience Patterns

## Status

Accepted

## Context

The trading agent depends on multiple external services:

- Polymarket CLOB API
- Polymarket Gamma API
- LLM providers (Anthropic, OpenAI)
- Search providers (Serper, SerpAPI)
- Vector/Graph stores

These services can experience transient failures, rate limiting, or degraded performance. The agent needs to handle these gracefully.

## Decision

Implement standard resilience patterns in `internal/resilience/`:

### Retry with Exponential Backoff

```go
result, err := resilience.Retry(ctx, cfg, func(ctx context.Context) (T, error) {
    return apiCall(ctx)
})
```

Features:

- Configurable max attempts (default: 3)
- Exponential backoff with jitter
- Configurable initial/max backoff
- Retryable error filtering

### Circuit Breaker

```go
cb := resilience.NewCircuitBreaker(cfg)
err := cb.Execute(ctx, func(ctx context.Context) error {
    return apiCall(ctx)
})
```

Features:

- Three states: Closed, Open, Half-Open
- Configurable failure/success thresholds
- Automatic recovery testing
- State change callbacks

## Consequences

**Positive:**

- Graceful handling of transient failures
- Prevents cascade failures via circuit breaker
- Configurable per-service
- Generic implementations work with any service

**Negative:**

- Added complexity
- Must be explicitly used by callers

**Implementation:**

- `internal/resilience/retry.go` - Retry with backoff
- `internal/resilience/circuitbreaker.go` - Circuit breaker
