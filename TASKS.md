# Tasks

Feature parity roadmap for polymarket-go vs [Polymarket/agents](https://github.com/Polymarket/agents).

## Phase 1: Trading Infrastructure

Required for real trading on Polymarket.

- [ ] **Order Signing** - EIP712 signature implementation for Polymarket orders
  - [ ] Implement EIP712 typed data signing
  - [ ] Port `OrderBuilder` functionality from py_order_utils
  - [ ] Port `Signer` functionality for private key signing
  - [ ] Support order nonce and expiration handling

- [ ] **Wallet Management** - Private key and address handling
  - [ ] Derive wallet address from private key
  - [ ] Secure private key loading from environment variables
  - [ ] Web3/Polygon RPC connection (`https://polygon-rpc.com`)

- [ ] **CLOB Trading** - Order placement and management
  - [ ] `PlaceOrder()` - Place limit order
  - [ ] `PlaceMarketOrder()` - Place market order (FOK)
  - [ ] `CancelOrder()` - Cancel existing order
  - [ ] `GetOpenOrders()` - List open orders
  - [ ] Support BUY and SELL sides
  - [ ] Configurable fees (basis points)

- [ ] **Token Approvals** - USDC and CTF approval transactions
  - [ ] USDC approval for exchange (unlimited)
  - [ ] CTF (Conditional Token Framework) approval
  - [ ] Neg Risk Exchange approval
  - [ ] Neg Risk Adapter approval

- [ ] **Balance Queries** - Wallet and position management
  - [ ] `GetUSDCBalance()` - Get wallet USDC balance
  - [ ] `GetPositions()` - Get current positions
  - [ ] `GetTradeHistory()` - Get historical trades

- [ ] **API Authentication** - CLOB API credentials
  - [ ] Key-based authentication for CLOB API
  - [ ] Derive/create API credentials automatically
  - [ ] Store API key, secret, passphrase securely

## Phase 2: Data Enrichment

External data sources for market research and analysis.

- [ ] **RAG Integration** - Vector database for semantic filtering
  - [ ] Chroma or pgvector integration
  - [ ] OpenAI embeddings (text-embedding-3-small)
  - [ ] `CreateMarketsRAG()` - Build market vector index
  - [ ] `QueryMarketsRAG()` - Semantic market search
  - [ ] `CreateEventsRAG()` - Build event vector index
  - [ ] `QueryEventsRAG()` - Semantic event search

- [ ] **News Connector** - NewsAPI integration
  - [ ] `GetArticlesForKeywords()` - Search news by keywords
  - [ ] `GetTopArticlesForMarket()` - Top news for a market
  - [ ] `GetArticlesForOptions()` - News for market outcomes
  - [ ] `GetCategory()` - Infer news category from market

- [ ] **Web Search** - Real-time information retrieval
  - [ ] Tavily integration (or alternative)
  - [ ] `SearchWeb()` - General web search
  - [ ] `SearchNews()` - News-specific search

- [ ] **Rich Data Models** - Port Pydantic models to Go structs
  - [ ] `Trade` - Full trade model (20+ fields)
  - [ ] `Market` - Rich market model (50+ fields)
  - [ ] `PolymarketEvent` - Event with markets, tags, metrics
  - [ ] `ClobReward` - Reward configuration
  - [ ] `Article` - News article model

## Phase 3: Agent Intelligence

Enhanced agent capabilities and workflows.

- [ ] **Superforecaster Prompts** - Calibrated probability estimation
  - [ ] Port superforecaster prompt template
  - [ ] Base rate analysis prompts
  - [ ] Evidence weighting prompts
  - [ ] Calibration check prompts

- [ ] **Market Analyst Prompts** - Trading opportunity discovery
  - [ ] Market filtering prompt
  - [ ] Edge calculation prompt
  - [ ] Risk assessment prompt

- [ ] **Trade Recommendation Pipeline** - Full workflow
  - [ ] `FilterEventsWithRAG()` - Semantic event filtering
  - [ ] `MapEventsToMarkets()` - Event to market mapping
  - [ ] `FilterMarketsWithRAG()` - Semantic market filtering
  - [ ] `SourceBestTrade()` - Analyze and recommend trade
  - [ ] `FormatTradeForExecution()` - Extract trade parameters

- [ ] **Token Management** - LLM context optimization
  - [ ] `EstimateTokens()` - Token count estimation
  - [ ] `DivideList()` - Chunk data for long contexts
  - [ ] `ProcessDataChunk()` - Process chunks sequentially
  - [ ] Multi-chunk LLM calls with aggregation

- [ ] **Autonomous Trading Loop** - Scheduled execution
  - [ ] Cron-based scheduling (weekly, daily, etc.)
  - [ ] `OneBestTrade()` - Single best trade workflow
  - [ ] Recursive retry with backoff
  - [ ] Circuit breaker for failures
  - [ ] Position maintenance logic

- [ ] **Market Creation Agent** - Suggest new markets
  - [ ] `SourceBestMarketToCreate()` - Market suggestions
  - [ ] Market description generation
  - [ ] Resolution criteria generation

## Phase 4: Infrastructure

Production readiness and tooling.

- [ ] **CLI Enhancements** - Feature-rich command interface
  - [ ] `get-all-markets` - List markets with filters
  - [ ] `get-all-events` - List events with filters
  - [ ] `get-relevant-news` - Search news by keywords
  - [ ] `create-local-markets-rag` - Build local RAG index
  - [ ] `query-local-markets-rag` - Query RAG index
  - [ ] `ask-superforecaster` - Interactive forecasting
  - [ ] `ask-llm` - General LLM queries
  - [ ] `run-autonomous-trader` - Start trading loop

- [ ] **REST Server** - HTTP API
  - [ ] FastAPI-equivalent using Chi or Gin
  - [ ] `/markets` - Market endpoints
  - [ ] `/events` - Event endpoints
  - [ ] `/trades` - Trade endpoints
  - [ ] `/agent` - Agent control endpoints

- [ ] **Testing** - Comprehensive test suite
  - [ ] Unit tests for all packages
  - [ ] Integration tests for API clients
  - [ ] Agent behavior tests
  - [ ] Mock LLM responses for testing

- [ ] **Error Handling** - Resilience patterns
  - [ ] Automatic retries with exponential backoff
  - [ ] Circuit breakers for external services
  - [ ] Graceful degradation
  - [ ] Structured error types

## Constants Reference

From Polymarket/agents for implementation:

```go
const (
    ChainID            = 137 // Polygon
    CLOBURL            = "https://clob.polymarket.com"
    GammaURL           = "https://gamma-api.polymarket.com"
    PolygonRPC         = "https://polygon-rpc.com"
    USDCAddress        = "0x2791Bca1f2de4661ED88A30C99A7a9449Aa84174"
    CTFAddress         = "0x4D97DCd97eC945f40cF65F87097ACe5EA0476045"
    ExchangeAddress    = "0x4bfb41d5b3570defd03c39a9a4d8de6bd8b8982e"
    NegRiskExchange    = "0xC5d563A36AE78145C45a50134d48A1215220f80a"
)
```

## Dependencies to Add

```go
// go.mod additions for full parity
require (
    github.com/ethereum/go-ethereum v1.13.x  // Ethereum/Polygon interaction
    github.com/chroma-core/chroma-go v0.x.x  // Vector database (if available)
    // Or use pgvector with jackc/pgx
)
```

## References

- [Polymarket/agents](https://github.com/Polymarket/agents) - Official Python SDK
- [py-clob-client](https://github.com/Polymarket/py-clob-client) - Python CLOB client
- [py-order-utils](https://github.com/Polymarket/py-order-utils) - Order signing utilities
- [Polymarket CLOB API Docs](https://docs.polymarket.com/)
