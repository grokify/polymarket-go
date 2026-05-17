# CLI Reference

The `polymarket-agent` CLI provides commands for market data, AI analysis, RAG search, and a REST API server.

## Installation

```bash
go build -o polymarket-agent ./cmd/polymarket-agent/
```

## Commands Overview

| Command | Description |
|---------|-------------|
| `markets list` | List markets with filters |
| `markets analyze` | AI superforecaster analysis |
| `events list` | List events with filters |
| `news` | Search news articles |
| `search` | Web search |
| `rag index` | Build RAG vector index |
| `rag search` | Semantic search over markets |
| `graphrag index` | Build knowledge graph |
| `graphrag related` | Find related markets |
| `graphrag hybrid` | Hybrid vector + graph search |
| `trade recommend` | Get trade recommendation |
| `trade auto` | Autonomous trading loop |
| `serve` | Start REST API server |
| `demo` | Demo mode with live data |

## Markets Commands

### List Markets

```bash
polymarket-agent markets list [flags]
```

**Flags:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--limit, -l` | int | 10 | Maximum markets to display |
| `--min-liquidity` | float | 0 | Minimum liquidity filter (USD) |
| `--active` | bool | true | Filter to active markets |
| `--json` | bool | false | Output as JSON |

**Example:**

```bash
# List top 10 markets by liquidity
polymarket-agent markets list --limit=10

# List markets with at least $100k liquidity as JSON
polymarket-agent markets list --min-liquidity=100000 --json
```

### Analyze Markets

Run AI superforecaster analysis on markets.

```bash
polymarket-agent markets analyze [flags]
```

**Flags:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--limit, -l` | int | 1 | Number of markets to analyze |
| `--min-liquidity` | float | 50000 | Minimum liquidity filter |
| `--question, -q` | string | | Filter by question text |
| `--model, -m` | string | claude-sonnet-4-20250514 | LLM model to use |

**Example:**

```bash
# Analyze top market
polymarket-agent markets analyze

# Analyze specific market
polymarket-agent markets analyze -q "bitcoin" --limit=1
```

**Requires:** `ANTHROPIC_API_KEY` environment variable.

## News & Search Commands

### News Search

Search for news articles relevant to prediction markets.

```bash
polymarket-agent news <query> [flags]
```

**Flags:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--limit, -l` | int | 10 | Number of results |
| `--json` | bool | false | Output as JSON |

**Example:**

```bash
polymarket-agent news "bitcoin ETF approval"
polymarket-agent news "election polls" --limit=5 --json
```

**Requires:** `SERPER_API_KEY` or `SERPAPI_API_KEY` environment variable.

### Web Search

Perform general web search with answer boxes.

```bash
polymarket-agent search <query> [flags]
```

**Example:**

```bash
polymarket-agent search "polymarket prediction markets"
```

## RAG Commands

### Index Markets

Build a vector index of markets for semantic search.

```bash
polymarket-agent rag index [flags]
```

**Flags:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--limit, -l` | int | 100 | Number of markets to index |
| `--min-liquidity` | float | 0 | Minimum liquidity filter |

**Example:**

```bash
# Index top 100 markets
polymarket-agent rag index --limit=100
```

### Semantic Search

Search indexed markets using natural language.

```bash
polymarket-agent rag search <query> [flags]
```

**Flags:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--top-k, -k` | int | 10 | Number of results |

**Example:**

```bash
polymarket-agent rag search "cryptocurrency regulation" --top-k=5
```

## GraphRAG Commands

### Build Knowledge Graph

```bash
polymarket-agent graphrag index [flags]
```

### Find Related Markets

```bash
polymarket-agent graphrag related <market-id> [flags]
```

### Hybrid Search

Combine vector similarity with graph traversal.

```bash
polymarket-agent graphrag hybrid <query> [flags]
```

## Trading Commands

### Get Trade Recommendation

```bash
polymarket-agent trade recommend [flags]
```

### Autonomous Trading

Run the autonomous trading loop.

```bash
polymarket-agent trade auto [flags]
```

**Flags:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--interval` | duration | 1h | Time between trading cycles |
| `--execute` | bool | false | Actually execute trades (dry run by default) |

## REST API Server

Start the HTTP API server with automatic OpenAPI spec generation.

```bash
polymarket-agent serve [flags]
```

**Flags:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--port, -p` | int | 8080 | Server port |
| `--with-news` | bool | false | Enable news/search endpoints |
| `--search-engine` | string | | Search engine (serper or serpapi) |

**Example:**

```bash
# Start server on default port
polymarket-agent serve

# Start with news search enabled on custom port
polymarket-agent serve --port=3000 --with-news
```

**Endpoints:**

| Method | Path | Description |
|--------|------|-------------|
| GET | `/health` | Health check |
| GET | `/docs` | OpenAPI documentation |
| GET | `/markets` | List markets |
| GET | `/markets/{conditionId}` | Get market |
| GET | `/markets/{tokenId}/orderbook` | Order book |
| GET | `/markets/{tokenId}/price` | Token price |
| GET | `/news?q=query` | News search (optional) |
| GET | `/search?q=query` | Web search (optional) |
| POST | `/rag/markets/search` | Semantic search |

## Global Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--model, -m` | string | claude-sonnet-4-20250514 | LLM model for AI commands |

## Environment Variables

| Variable | Required For | Description |
|----------|-------------|-------------|
| `ANTHROPIC_API_KEY` | `markets analyze`, `trade` | Anthropic API key |
| `OPENAI_API_KEY` | `rag` | OpenAI API key (embeddings) |
| `SERPER_API_KEY` | `news`, `search`, `serve --with-news` | Serper API key |
| `SERPAPI_API_KEY` | `news`, `search` | SerpAPI key (alternative) |

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | Error (check stderr) |
