# ADR-0005: Use omnillm-core for LLM Abstraction

## Status

Accepted

## Context

The trading agent uses LLMs for:

- Superforecaster probability estimation
- Market analysis and filtering
- Trade recommendations
- Risk assessment

We need to support multiple LLM providers (Anthropic, OpenAI, etc.) without tight coupling.

## Decision

Use [plexusone/omnillm-core](https://github.com/plexusone/omnillm-core) as the LLM abstraction layer.

omnillm-core provides:

- **Provider interface** - Unified API for all LLM providers
- **Multiple backends** - Anthropic, OpenAI, AWS Bedrock, etc.
- **Streaming support** - For real-time responses
- **Tool use** - Function calling support

## Consequences

**Positive:**

- Easy to switch LLM providers
- Consistent API regardless of backend
- Provider-specific features abstracted away
- Supports both chat and completion modes

**Negative:**

- May lag behind provider-specific SDK updates
- Additional dependency

**Implementation:**

- `internal/prompts/agent.go` - Uses `provider.Provider` interface
- Default model: `claude-sonnet-4-20250514`
- Configurable via `--model` flag
