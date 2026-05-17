# REST API Server

The REST API server provides HTTP access to Polymarket data with automatic OpenAPI specification generation.

## Starting the Server

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
# Start on default port
polymarket-agent serve

# Start with all features
polymarket-agent serve --port=3000 --with-news
```

## OpenAPI Documentation

The server automatically generates an OpenAPI specification. Access the interactive documentation at:

```
http://localhost:8080/docs
```

## Endpoints

### Health Check

```
GET /health
```

**Response:**

```json
{
  "status": "ok"
}
```

### List Markets

```
GET /markets
```

**Query Parameters:**

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `limit` | int | 10 | Maximum results (1-100) |
| `active` | bool | | Filter by active status |
| `closed` | bool | | Filter by closed status |
| `min_liquidity` | float | | Minimum liquidity in USD |
| `max_liquidity` | float | | Maximum liquidity in USD |
| `q` | string | | Text search query |
| `tag` | string | | Filter by tag slug |
| `order` | string | liquidity | Sort field (liquidity, volume, end_date) |
| `ascending` | bool | false | Sort ascending |
| `cursor` | string | | Pagination cursor |

**Response:**

```json
{
  "markets": [
    {
      "id": "0x...",
      "question": "Will Bitcoin reach $100k in 2025?",
      "slug": "bitcoin-100k-2025",
      "description": "...",
      "active": true,
      "closed": false,
      "liquidity": 150000.00,
      "volume_24hr": 25000.00,
      "outcomes": "[\"Yes\",\"No\"]",
      "outcome_prices": "[0.65,0.35]",
      "best_bid": 0.64,
      "best_ask": 0.66,
      "spread": 0.02,
      "last_trade_price": 0.65,
      "end_date": "2025-12-31T23:59:59Z"
    }
  ],
  "count": 1
}
```

### Get Single Market

```
GET /markets/{conditionId}
```

**Path Parameters:**

| Parameter | Type | Description |
|-----------|------|-------------|
| `conditionId` | string | Market condition ID |

**Response:** Single `MarketResponse` object.

### Get Order Book

```
GET /markets/{tokenId}/orderbook
```

**Path Parameters:**

| Parameter | Type | Description |
|-----------|------|-------------|
| `tokenId` | string | Market token ID |

**Response:**

```json
{
  "market": "0x...",
  "asset_id": "...",
  "hash": "...",
  "bids": [
    {"price": "0.64", "size": "1000"}
  ],
  "asks": [
    {"price": "0.66", "size": "500"}
  ]
}
```

### Get Token Price

```
GET /markets/{tokenId}/price
```

**Response:**

```json
{
  "token_id": "...",
  "price": 0.65,
  "mid_price": 0.65,
  "spread": 0.02
}
```

### Search News

Requires `--with-news` flag and `SERPER_API_KEY` or `SERPAPI_API_KEY`.

```
GET /news
```

**Query Parameters:**

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `q` | string | **required** | Search query |
| `limit` | int | 10 | Number of results (1-100) |
| `lang` | string | | Language code (e.g., "en") |
| `country` | string | | Country code (e.g., "us") |

**Response:**

```json
{
  "articles": [
    {
      "title": "Bitcoin ETF Sees Record Inflows",
      "link": "https://...",
      "source": "Reuters",
      "date": "2025-05-17",
      "snippet": "...",
      "thumbnail": "https://..."
    }
  ],
  "count": 10
}
```

### Web Search

Requires `--with-news` flag and search API key.

```
GET /search
```

**Query Parameters:**

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `q` | string | **required** | Search query |
| `limit` | int | 10 | Number of results |
| `lang` | string | | Language code |
| `country` | string | | Country code |

**Response:**

```json
{
  "results": [
    {
      "position": 1,
      "title": "Polymarket - Prediction Markets",
      "link": "https://polymarket.com",
      "snippet": "...",
      "domain": "polymarket.com"
    }
  ],
  "answer_box": {
    "title": "...",
    "answer": "...",
    "snippet": "...",
    "link": "..."
  },
  "count": 10
}
```

### RAG Semantic Search - Markets

Requires RAG store configuration.

```
POST /rag/markets/search
```

**Request Body:**

```json
{
  "query": "cryptocurrency regulation",
  "top_k": 10
}
```

**Response:**

```json
{
  "results": [
    {
      "id": "0x...",
      "score": 0.89,
      "question": "Will SEC approve spot Bitcoin ETF?",
      "description": "...",
      "outcomes": "[\"Yes\",\"No\"]",
      "liquidity": "150000.00",
      "end_date": "2025-06-30"
    }
  ],
  "count": 5
}
```

### RAG Semantic Search - Events

```
POST /rag/events/search
```

**Request Body:**

```json
{
  "query": "2024 election",
  "top_k": 10
}
```

### Index Markets for RAG

```
POST /rag/markets/index
```

**Request Body:**

```json
{
  "limit": 100,
  "min_liquidity": 10000,
  "active_only": true
}
```

**Response:**

```json
{
  "indexed": 100,
  "message": "Successfully indexed 100 markets"
}
```

## Error Handling

All errors return a JSON response with details:

```json
{
  "status": 500,
  "title": "Internal Server Error",
  "detail": "fetching markets: connection refused"
}
```

**HTTP Status Codes:**

| Code | Description |
|------|-------------|
| 200 | Success |
| 400 | Bad Request (invalid parameters) |
| 404 | Not Found |
| 500 | Internal Server Error |

## Programmatic Usage

```go
import "github.com/grokify/polymarket-go/internal/server"

cfg := server.Config{
    Port:             8080,
    Logger:           slog.Default(),
    PolymarketClient: polymarket.NewClient(),
    NewsSearcher:     newsSearcher, // optional
    RAGStore:         ragStore,     // optional
}

srv, err := server.New(cfg)
if err != nil {
    log.Fatal(err)
}

// Start server
srv.ListenAndServe()

// Or with graceful shutdown
srv.ListenAndServeContext(ctx)
```

## Middleware

The server includes the following middleware by default:

- **RequestID** - Adds unique request ID header
- **RealIP** - Extracts real client IP
- **Logger** - Request/response logging
- **Recoverer** - Panic recovery
- **Timeout** - 60-second request timeout

## Security

The server configures secure timeouts to prevent Slowloris attacks:

| Timeout | Value | Description |
|---------|-------|-------------|
| ReadTimeout | 15s | Maximum time to read request |
| ReadHeaderTimeout | 10s | Maximum time to read headers |
| WriteTimeout | 60s | Maximum time to write response |
| IdleTimeout | 120s | Maximum idle connection time |
