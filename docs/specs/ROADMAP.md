# Tasks

Feature parity roadmap for polymarket-go vs [Polymarket/agents](https://github.com/Polymarket/agents).

## Next Priorities

| Priority | Task | Rationale |
|----------|------|-----------|
| 1 | **Boost Test Coverage** | news (20%), server (27%) need integration tests |
| 2 | **Market Creation Agent** | Novel AI feature for suggesting new prediction markets |
| 3 | **Position Maintenance** | Needed for fully autonomous trading |
| 4 | **Persistent RAG Store** | pgvector for production RAG persistence |
| 5 | **Trade Execution** | Implement actual order placement (currently stubbed) |

## Phase 1: Trading Infrastructure ✅

Required for real trading on Polymarket. **Completed via [polymarket-go-sdk](https://github.com/GoPolymarket/polymarket-go-sdk) integration.**

- [x] **Order Signing** - EIP712 signature implementation for Polymarket orders
  - [x] Implement EIP712 typed data signing (via `auth.NewPrivateKeySigner`)
  - [x] Port `OrderBuilder` functionality (via `clob.NewOrderBuilder`)
  - [x] Port `Signer` functionality for private key signing
  - [x] Support order nonce and expiration handling

- [x] **Wallet Management** - Private key and address handling
  - [x] Derive wallet address from private key
  - [x] Secure private key loading from environment variables
  - [x] Web3/Polygon RPC connection

- [x] **CLOB Trading** - Order placement and management
  - [x] `PlaceOrder()` - Place limit order
  - [x] `PlaceMarketOrder()` - Place market order (FOK)
  - [x] `CancelOrder()` - Cancel existing order
  - [x] `GetOrders()` - List orders
  - [x] Support BUY and SELL sides
  - [x] Configurable fees (basis points)

- [x] **Token Approvals** - Via SDK's CTF client
  - [x] USDC approval for exchange
  - [x] CTF (Conditional Token Framework) approval
  - [x] Neg Risk Exchange approval

- [x] **Balance Queries** - Wallet and position management
  - [x] `GetBalanceAllowance()` - Get balance and allowance
  - [x] `GetTrades()` - Get trade history

- [x] **API Authentication** - CLOB API credentials
  - [x] Key-based authentication for CLOB API
  - [x] API key, secret, passphrase support

- [x] **WebSocket Streaming** - Real-time data
  - [x] `SubscribePrices()` - Price updates
  - [x] `SubscribeOrderbook()` - Order book updates
  - [x] `SubscribeMidpoints()` - Midpoint updates
  - [x] `SubscribeUserOrders()` - User order updates
  - [x] `SubscribeUserTrades()` - User trade updates

## Phase 2: Data Enrichment

External data sources for market research and analysis.

**Using [omniretrieve](https://github.com/plexusone/omniretrieve) for RAG and [omniserp](https://github.com/plexusone/omniserp) for search.**

- [x] **RAG Integration** - Via omniretrieve with pgvector
  - [x] Add omniretrieve dependency
  - [x] Store configuration (`internal/rag/store.go`)
  - [x] Embedding integration (OpenAI text-embedding-3-small)
  - [x] `IndexMarkets()` - Build market vector index
  - [x] `SearchMarkets()` - Semantic market search
  - [x] `IndexEvents()` - Build event vector index
  - [x] `SearchEvents()` - Semantic event search
  - [x] CLI: `rag index` and `rag search` commands
  - [ ] pgvector persistent store (currently in-memory)

- [x] **News & Web Search** - Via omniserp (Serper/SerpAPI)
  - [x] Add omniserp dependency
  - [x] `SearchNews()` - News search with normalized results
  - [x] `SearchWeb()` - Web search with answer boxes
  - [x] `GetNewsForMarket()` - News relevant to a market question
  - [x] `GetNewsForKeywords()` - News by keywords
  - [x] CLI: `news` and `search` commands

- [x] **GraphRAG** - Relationship-aware retrieval (via omniretrieve)
  - [x] Event → Market graph edges (`EdgeTypeHasMarket`)
  - [x] Topic/tag clustering (`EdgeTypeTopicRelatesTo`)
  - [x] Correlated market detection (`EdgeTypeCorrelatedWith`, `EdgeTypeSameEvent`)
  - [x] In-memory graph store (Neo4j planned)
  - [x] Hybrid vector+graph search
  - [x] CLI: `graphrag index`, `graphrag related`, `graphrag topic`, `graphrag hybrid`

- [x] **Rich Data Models** - Via polymarket-go-sdk
  - [x] `clobtypes.Trade` - Full trade model
  - [x] `clobtypes.Market` - Rich market model (50+ fields)
  - [x] `gamma.Event` - Event with markets, tags, metrics
  - [x] `clobtypes.OrderResponse` - Order response model

## Phase 3: Agent Intelligence

Enhanced agent capabilities and workflows.

- [x] **Superforecaster Prompts** - Calibrated probability estimation
  - [x] Port superforecaster prompt template
  - [x] Base rate analysis prompts (included in superforecaster)
  - [x] Evidence weighting prompts (included in superforecaster)
  - [ ] Calibration check prompts

- [x] **Market Analyst Prompts** - Trading opportunity discovery
  - [x] Market filtering prompt (`FilterMarkets()`)
  - [x] Edge calculation prompt (`EdgeCalculation()`)
  - [x] Risk assessment prompt (`RiskAssessment()`)

- [x] **Trade Recommendation Pipeline** - Full workflow
  - [x] `FilterEventsWithRAG()` - Semantic event filtering with RAG
  - [x] `MapEventsToMarkets()` - Event to market mapping
  - [x] `FilterMarketsWithRAG()` - Semantic market filtering with RAG
  - [x] `SourceBestTrade()` - Analyze and recommend trade
  - [x] `FormatTradeForExecution()` - Extract trade parameters (`ParseTradeResponse()`)

- [x] **Token Management** - LLM context optimization
  - [x] `EstimateTokens()` - Token count estimation
  - [x] `DivideList()` - Chunk data for long contexts
  - [x] `ProcessDataChunks()` - Process chunks sequentially with aggregation

- [x] **Autonomous Trading Loop** - Scheduled execution
  - [x] Interval-based scheduling (`trade auto --interval`)
  - [x] `OneBestTrade()` - Single best trade workflow (`trade recommend`)
  - [x] Recursive retry with backoff (`internal/resilience/retry.go`)
  - [x] Circuit breaker for failures (`internal/resilience/circuitbreaker.go`)
  - [ ] Position maintenance logic

- [ ] **Market Creation Agent** - Suggest new markets
  - [ ] `SourceBestMarketToCreate()` - Market suggestions
  - [ ] Market description generation
  - [ ] Resolution criteria generation

## Phase 4: Infrastructure

Production readiness and tooling.

- [x] **CLI Enhancements** - Feature-rich command interface (Cobra)
  - [x] `demo` - Demo mode with live market data
  - [x] `markets list` - List markets with filters (--json, --min-liquidity)
  - [x] `markets analyze` - Superforecaster market analysis
  - [x] `events list` - List events with filters (--json)
  - [x] `trade auto` - Run autonomous trading loop (--interval, --execute)
  - [x] `trade recommend` - Get single trade recommendation
  - [x] `news` - Search news by keywords (via omniserp)
  - [x] `search` - Web search with answer boxes
  - [x] `rag index` - Build local RAG index (markets/events)
  - [x] `rag search` - Query RAG index semantically
  - [x] `ask` - Ad-hoc LLM queries with streaming support

- [x] **REST Server** - HTTP API (Huma + Chi)
  - [x] Huma v2 for automatic OpenAPI spec generation
  - [x] Chi v5 router with middleware (logging, recovery, timeouts)
  - [x] `/health` - Health check endpoint
  - [x] `/markets` - List markets with filters
  - [x] `/markets/{conditionId}` - Get single market
  - [x] `/markets/{tokenId}/orderbook` - Order book endpoint
  - [x] `/markets/{tokenId}/price` - Price endpoint
  - [x] `/news` - News search (optional, via omniserp)
  - [x] `/search` - Web search (optional, via omniserp)
  - [x] `/rag/markets/search` - Semantic market search (optional)
  - [x] `/rag/events/search` - Semantic event search (optional)
  - [x] `/rag/markets/index` - Index markets for RAG (optional)
  - [x] CLI: `serve` command with --port, --with-news flags

- [ ] **Testing** - Comprehensive test suite
  - [x] Unit tests for resilience (98%), llm (96%), loader (82%), errors (80%), prompts (75%), rag (67%), executor (66%), tools (59%), polymarket (39%)
  - [ ] Unit tests for news (20%), server (27%) - require external service mocks
  - [ ] Integration tests for API clients
  - [ ] Agent behavior tests
  - [x] Mock LLM responses for testing (`internal/prompts/agent_test.go`)

- [x] **Error Handling** - Resilience patterns
  - [x] Built-in retries via polymarket-go-sdk
  - [x] Retry with exponential backoff (`internal/resilience/retry.go`)
  - [x] Circuit breakers for external services (`internal/resilience/circuitbreaker.go`)
  - [x] Structured error types (`internal/errors/`) with Retryable and HTTPStatusCoder interfaces
  - [ ] Graceful degradation
  - [ ] Integrate structured errors into all packages

## SDK Integration

Using [GoPolymarket/polymarket-go-sdk](https://github.com/GoPolymarket/polymarket-go-sdk) v1.1.0 for:

- Full CLOB REST API
- WebSocket streaming with auto-reconnect
- EIP-712 order signing
- Order builder with fluent API
- Gamma API for metadata
- High-precision decimals
- AWS KMS signer support

### SDKClient Usage

```go
import "github.com/grokify/polymarket-go/internal/polymarket"

// Create authenticated client
client, err := polymarket.NewSDKClient(polymarket.SDKConfig{
    // Reads from env: POLYGON_WALLET_PRIVATE_KEY, POLYMARKET_API_KEY, etc.
})

// Place a limit order
resp, err := client.PlaceOrder(ctx, polymarket.PlaceOrderParams{
    TokenID:   "TOKEN_ID",
    Side:      "BUY",
    Price:     decimal.NewFromFloat(0.55),
    Size:      decimal.NewFromFloat(100),
    OrderType: clobtypes.OrderTypeGTC,
})

// Subscribe to price updates
prices, err := client.SubscribePrices(ctx, []string{"TOKEN_ID"})
for price := range prices {
    fmt.Printf("Price: %v\n", price)
}
```

## References

- [GoPolymarket/polymarket-go-sdk](https://github.com/GoPolymarket/polymarket-go-sdk) - Go SDK (integrated)
- [Polymarket/agents](https://github.com/Polymarket/agents) - Official Python SDK
- [py-clob-client](https://github.com/Polymarket/py-clob-client) - Python CLOB client
- [Polymarket CLOB API Docs](https://docs.polymarket.com/)
