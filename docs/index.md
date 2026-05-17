# polymarket-go

Go SDK for building AI trading agents on [Polymarket](https://polymarket.com) prediction markets.

## Features

- **Polymarket API Client** - Full client for Gamma (markets) and CLOB (trading) APIs
- **REST API Server** - HTTP API with automatic OpenAPI spec generation (Huma + Chi)
- **RAG & GraphRAG** - Semantic search over markets with knowledge graph traversal
- **News & Web Search** - Real-time news and web search via omniserp
- **Multi-Agent Workflows** - Define agent teams using [multi-agent-spec](https://github.com/plexusone/multi-agent-spec) format
- **LLM Integration** - Works with any LLM via [omnillm](https://github.com/plexusone/omnillm) + [LangChainGo](https://github.com/tmc/langchaingo)
- **Resilience Patterns** - Retry with backoff, circuit breakers for external services

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│  polymarket-go                                                   │
├─────────────────────────────────────────────────────────────────┤
│  cmd/polymarket-agent/     CLI and REST server                   │
├─────────────────────────────────────────────────────────────────┤
│  internal/                                                       │
│  ├── server/               REST API (Huma + Chi)                 │
│  ├── polymarket/           Polymarket API client                 │
│  ├── rag/                  RAG & GraphRAG retrieval              │
│  ├── news/                 News & web search (omniserp)          │
│  ├── prompts/              LLM prompts (superforecaster, etc.)   │
│  ├── resilience/           Retry, circuit breaker patterns       │
│  ├── executor/             Workflow execution engine             │
│  └── tools/                Agent tools for Polymarket            │
├─────────────────────────────────────────────────────────────────┤
│  omniagent/skill/          Compiled skill for omniagent          │
├─────────────────────────────────────────────────────────────────┤
│  agents/specs/             Multi-agent-spec definitions          │
└─────────────────────────────────────────────────────────────────┘
```

## Agent Team

The default trading team consists of three agents in a graph workflow:

| Agent | Role | Model |
|-------|------|-------|
| [**market-analyst**](agents/market-analyst.md) | Discovers trading opportunities | sonnet |
| [**superforecaster**](agents/superforecaster.md) | Generates probability estimates | sonnet |
| [**trader**](agents/trader.md) | Executes trades based on analysis | haiku |

**Workflow:** `discover → forecast → execute`

## Quick Links

- [Installation](getting-started/installation.md)
- [Quick Start](getting-started/quickstart.md)
- [CLI Reference](getting-started/cli.md)
- [REST API Reference](api/rest-server.md)
- [Polymarket Client](api/polymarket.md)
- [Architecture Decisions](specs/adr/index.md)
- [Roadmap](specs/ROADMAP.md)
- [Changelog](releases/CHANGELOG.md)

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
