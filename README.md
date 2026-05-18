# Polymarket SDK for Go

[![Go CI][go-ci-svg]][go-ci-url]
[![Go Lint][go-lint-svg]][go-lint-url]
[![Go SAST][go-sast-svg]][go-sast-url]
[![Go Report Card][goreport-svg]][goreport-url]
[![Docs][docs-godoc-svg]][docs-godoc-url]
[![Visualization][viz-svg]][viz-url]
[![License][license-svg]][license-url]

 [go-ci-svg]: https://github.com/grokify/polymarket-go/actions/workflows/go-ci.yaml/badge.svg?branch=main
 [go-ci-url]: https://github.com/grokify/polymarket-go/actions/workflows/go-ci.yaml
 [go-lint-svg]: https://github.com/grokify/polymarket-go/actions/workflows/go-lint.yaml/badge.svg?branch=main
 [go-lint-url]: https://github.com/grokify/polymarket-go/actions/workflows/go-lint.yaml
 [go-sast-svg]: https://github.com/grokify/polymarket-go/actions/workflows/go-sast-codeql.yaml/badge.svg?branch=main
 [go-sast-url]: https://github.com/grokify/polymarket-go/actions/workflows/go-sast-codeql.yaml
 [goreport-svg]: https://goreportcard.com/badge/github.com/grokify/polymarket-go
 [goreport-url]: https://goreportcard.com/report/github.com/grokify/polymarket-go
 [docs-godoc-svg]: https://pkg.go.dev/badge/github.com/grokify/polymarket-go
 [docs-godoc-url]: https://pkg.go.dev/github.com/grokify/polymarket-go
 [viz-svg]: https://img.shields.io/badge/visualizaton-Go-blue.svg
 [viz-url]: https://mango-dune-07a8b7110.1.azurestaticapps.net/?repo=grokify%2Fpolymarket-go
 [loc-svg]: https://tokei.rs/b1/github/grokify/polymarket-go
 [repo-url]: https://github.com/grokify/polymarket-go
 [license-svg]: https://img.shields.io/badge/license-MIT-blue.svg
 [license-url]: https://github.com/grokify/polymarket-go/blob/main/LICENSE

Go SDK for building AI trading agents on [Polymarket](https://polymarket.com) prediction markets.

## Features

- 📊 **Polymarket API Client** - Full client for Gamma (markets) and CLOB (trading) APIs
- 🌐 **REST API Server** - HTTP API with automatic OpenAPI spec generation (Huma + Chi)
- 🔍 **RAG & GraphRAG** - Semantic search over markets with knowledge graph traversal
- 📰 **News & Web Search** - Real-time news and web search via omniserp
- 🤖 **Multi-Agent Workflows** - Define agent teams using [multi-agent-spec](https://github.com/plexusone/multi-agent-spec) format
- 🧠 **LLM Integration** - Works with any LLM via [omnillm](https://github.com/plexusone/omnillm) + [LangChainGo](https://github.com/tmc/langchaingo)
- 🛡️ **Resilience Patterns** - Retry with backoff, circuit breakers for external services
- 🔌 **OmniAgent Integration** - Compiled skill wrapper for embedding in [omniagent](https://github.com/plexusone/omniagent)-based agents

## Installation

```bash
go get github.com/grokify/polymarket-go
```

## Quick Start

### CLI Commands

```bash
# Build the CLI
go build -o polymarket-agent ./cmd/polymarket-agent/

# List markets
polymarket-agent markets list --limit=10 --min-liquidity=50000

# Analyze markets with AI superforecaster
polymarket-agent markets analyze --limit=1

# Search news
polymarket-agent news "bitcoin ETF"

# Semantic search with RAG
polymarket-agent rag index --limit=100
polymarket-agent rag search "cryptocurrency regulation"

# Start REST API server
polymarket-agent serve --port=8080
```

### REST API Server

Start the server and access the auto-generated OpenAPI docs:

```bash
polymarket-agent serve --port=8080 --with-news
```

**Endpoints:**

| Method | Path | Description |
|--------|------|-------------|
| GET | `/health` | Health check |
| GET | `/markets` | List markets with filters |
| GET | `/markets/{conditionId}` | Get single market |
| GET | `/markets/{tokenId}/orderbook` | Order book |
| GET | `/markets/{tokenId}/price` | Token price |
| GET | `/news?q=query` | News search |
| GET | `/search?q=query` | Web search |
| POST | `/rag/markets/search` | Semantic market search |

OpenAPI documentation available at `/docs`.

### Use the Polymarket Client

```go
import "github.com/grokify/polymarket-go/internal/polymarket"

client := polymarket.NewClient()

// Fetch active markets
markets, err := client.GetMarkets(ctx, polymarket.GetMarketsParams{
    Active: &active,
    Limit:  10,
    Order:  "liquidity",
})

// Get order book for a token
book, err := client.GetOrderBook(ctx, tokenID)
```

### RAG Semantic Search

```go
import "github.com/grokify/polymarket-go/internal/rag"

// Create RAG store with embedder
store, _ := rag.NewStore(rag.StoreConfig{
    VectorIndex: vectorIndex,
    Embedder:    embedder,
})

// Index markets
store.IndexMarkets(ctx, markets)

// Semantic search
results, _ := store.SearchMarkets(ctx, "cryptocurrency regulation", 10)
```

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│  polymarket-go                                                  │
├─────────────────────────────────────────────────────────────────┤
│  cmd/polymarket-agent/     CLI and REST server                  │
├─────────────────────────────────────────────────────────────────┤
│  internal/                                                      │
│  ├── server/               REST API (Huma + Chi)                │
│  ├── polymarket/           Polymarket API client                │
│  ├── rag/                  RAG & GraphRAG retrieval             │
│  ├── news/                 News & web search (omniserp)         │
│  ├── prompts/              LLM prompts (superforecaster, etc.)  │
│  ├── resilience/           Retry, circuit breaker patterns      │
│  ├── executor/             Workflow execution engine            │
│  └── tools/                Agent tools for Polymarket           │
├─────────────────────────────────────────────────────────────────┤
│  omniagent/skill/          Compiled skill for omniagent         │
├─────────────────────────────────────────────────────────────────┤
│  agents/specs/             Multi-agent-spec definitions         │
└─────────────────────────────────────────────────────────────────┘
```

## Agent Team

The default trading team consists of three agents in a graph workflow:

| Agent | Role | Model |
|-------|------|-------|
| **market-analyst** | Discovers trading opportunities | sonnet |
| **superforecaster** | Generates probability estimates | sonnet |
| **trader** | Executes trades based on analysis | haiku |

Workflow: `discover → forecast → execute`

## Environment Variables

| Variable | Description |
|----------|-------------|
| `ANTHROPIC_API_KEY` | Anthropic API key for LLM |
| `OPENAI_API_KEY` | OpenAI API key for embeddings |
| `SERPER_API_KEY` | Serper API key for news/web search |
| `SERPAPI_API_KEY` | SerpAPI key (alternative to Serper) |
| `POLYGON_WALLET_PRIVATE_KEY` | Private key for trading |
| `POLYMARKET_API_KEY` | Polymarket CLOB API key |
| `POLYMARKET_API_SECRET` | Polymarket CLOB API secret |
| `POLYMARKET_API_PASSPHRASE` | Polymarket CLOB API passphrase |

## Dependencies

| Package | Purpose |
|---------|---------|
| [polymarket-go-sdk](https://github.com/GoPolymarket/polymarket-go-sdk) | Polymarket trading SDK |
| [huma](https://github.com/danielgtaylor/huma) | REST API with OpenAPI generation |
| [chi](https://github.com/go-chi/chi) | HTTP router |
| [omnillm-core](https://github.com/plexusone/omnillm-core) | LLM provider abstraction |
| [omniretrieve](https://github.com/plexusone/omniretrieve) | RAG & GraphRAG retrieval |
| [omniserp](https://github.com/plexusone/omniserp) | News & web search |
| [langchaingo](https://github.com/tmc/langchaingo) | Go LLM framework |
| [omniagent](https://github.com/plexusone/omniagent) | AI agent framework (optional) |

## Documentation

- [Getting Started](docs/getting-started/quickstart.md)
- [CLI Reference](docs/getting-started/cli.md)
- [REST API Reference](docs/api/rest-server.md)
- [Architecture Decisions](docs/specs/adr/)
- [Roadmap](docs/specs/ROADMAP.md)

## License

MIT License - see [LICENSE](LICENSE) for details.
