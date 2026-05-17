# ADR-0004: Use omniserp for News and Web Search

## Status

Accepted

## Context

Market research requires access to current news and web information to:

- Find news relevant to prediction markets
- Research topics for informed forecasting
- Gather context for market analysis

Options considered:

1. Direct integration with Serper API
2. Direct integration with SerpAPI
3. Use omniserp (multi-provider abstraction)

## Decision

Use [plexusone/omniserp](https://github.com/plexusone/omniserp) for all search operations.

omniserp provides:

- **Multi-provider support** - Serper, SerpAPI with automatic fallback
- **Normalized responses** - Consistent data structures across providers
- **News search** - Dedicated news endpoint with article metadata
- **Web search** - General search with answer boxes
- **Web scraping** - Extract content from URLs

## Consequences

**Positive:**

- Provider-agnostic code
- Automatic fallback if one provider fails
- Normalized response format simplifies parsing
- Easy to add new providers

**Negative:**

- Additional abstraction layer
- May not expose all provider-specific features

**Implementation:**

- `internal/news/search.go` - News and web search wrapper
- CLI commands: `news`, `search`
