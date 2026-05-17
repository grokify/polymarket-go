# ADR-0001: Use polymarket-go-sdk for Trading Infrastructure

## Status

Accepted

## Context

Building a Polymarket trading agent requires:

- EIP-712 order signing for the CLOB API
- WebSocket connections for real-time data
- Wallet management and token approvals
- Complex order building with proper nonces and expirations

Implementing these from scratch would require significant effort and deep understanding of Polymarket's trading protocol.

## Decision

Use [GoPolymarket/polymarket-go-sdk](https://github.com/GoPolymarket/polymarket-go-sdk) as the foundation for all trading operations.

The SDK provides:

- Full CLOB REST API client
- WebSocket streaming with auto-reconnect
- EIP-712 order signing
- Order builder with fluent API
- Gamma API for market metadata
- High-precision decimal handling
- AWS KMS signer support

## Consequences

**Positive:**

- Significantly reduced development time for Phase 1
- Battle-tested signing and order logic
- Maintained by the community
- Supports both private key and KMS signing

**Negative:**

- External dependency we don't control
- Must adapt to SDK's API design
- Version updates may require code changes

**Mitigations:**

- Wrapped SDK in `internal/polymarket/sdk.go` for abstraction
- Can fork if needed for critical fixes
