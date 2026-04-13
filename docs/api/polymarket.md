# Polymarket Client

The Polymarket client provides access to both the Gamma (markets) and CLOB (trading) APIs.

## Import

```go
import "github.com/grokify/polymarket-go/internal/polymarket"
```

## Creating a Client

```go
// Default client
client := polymarket.NewClient()

// With options
client := polymarket.NewClient(
    polymarket.WithCLOBURL("https://clob.polymarket.com"),
    polymarket.WithGammaURL("https://gamma-api.polymarket.com"),
    polymarket.WithHTTPClient(customHTTPClient),
)
```

## Markets API (Gamma)

### GetMarkets

Fetch active prediction markets with optional filters.

```go
func (c *Client) GetMarkets(ctx context.Context, params GetMarketsParams) ([]Market, error)
```

**Parameters:**

| Field | Type | Description |
|-------|------|-------------|
| `Active` | *bool | Filter by active status |
| `Closed` | *bool | Filter by closed status |
| `Limit` | int | Maximum results to return |
| `Cursor` | string | Pagination cursor |
| `TagSlug` | string | Category filter |
| `Order` | string | Sort field (e.g., "liquidity") |
| `Ascending` | bool | Sort direction |
| `LiqMin` | float64 | Minimum liquidity filter |
| `LiqMax` | float64 | Maximum liquidity filter |
| `TextQuery` | string | Search query |

**Example:**

```go
active := true
markets, err := client.GetMarkets(ctx, polymarket.GetMarketsParams{
    Active: &active,
    Limit:  10,
    Order:  "liquidity",
    LiqMin: 50000,
})
if err != nil {
    log.Fatal(err)
}

for _, m := range markets {
    fmt.Printf("%s - $%.0f liquidity\n", m.Question, m.LiquidityNum)
}
```

### GetMarket

Fetch a single market by condition ID.

```go
func (c *Client) GetMarket(ctx context.Context, conditionID string) (*Market, error)
```

### Market Type

```go
type Market struct {
    ID              string  `json:"id"`
    Question        string  `json:"question"`
    Slug            string  `json:"slug"`
    Description     string  `json:"description"`
    ConditionID     string  `json:"conditionId"`
    QuestionID      string  `json:"questionID"`
    EndDateISO      string  `json:"endDateIso"`
    Active          bool    `json:"active"`
    Closed          bool    `json:"closed"`
    LiquidityNum    float64 `json:"liquidityNum"`
    VolumeNum       float64 `json:"volumeNum"`
    Volume24hr      float64 `json:"volume24hr"`
    Outcomes        string  `json:"outcomes"`        // JSON array
    OutcomePrices   string  `json:"outcomePrices"`   // JSON array
    ClobTokenIDs    string  `json:"clobTokenIds"`    // JSON array
    BestBid         float64 `json:"bestBid"`
    BestAsk         float64 `json:"bestAsk"`
    Spread          float64 `json:"spread"`
    LastTradePrice  float64 `json:"lastTradePrice"`
}
```

## Order Book API (CLOB)

### GetOrderBook

Fetch the order book for a token.

```go
func (c *Client) GetOrderBook(ctx context.Context, tokenID string) (*OrderBook, error)
```

**Example:**

```go
book, err := client.GetOrderBook(ctx, tokenID)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Bids: %d, Asks: %d\n", len(book.Bids), len(book.Asks))
if len(book.Bids) > 0 {
    fmt.Printf("Best bid: %s @ %s\n", book.Bids[0].Size, book.Bids[0].Price)
}
```

### OrderBook Type

```go
type OrderBook struct {
    Market string           `json:"market"`
    Asset  string           `json:"asset_id"`
    Hash   string           `json:"hash"`
    Bids   []OrderBookLevel `json:"bids"`
    Asks   []OrderBookLevel `json:"asks"`
}

type OrderBookLevel struct {
    Price string `json:"price"`
    Size  string `json:"size"`
}
```

## Price APIs

### GetPrice

```go
func (c *Client) GetPrice(ctx context.Context, tokenID string) (*Price, error)
```

### GetMidPrice

```go
func (c *Client) GetMidPrice(ctx context.Context, tokenID string) (*MidPrice, error)
```

### GetSpread

```go
func (c *Client) GetSpread(ctx context.Context, tokenID string) (*Spread, error)
```

**Example:**

```go
mid, _ := client.GetMidPrice(ctx, tokenID)
spread, _ := client.GetSpread(ctx, tokenID)

fmt.Printf("Mid: %.4f, Spread: %.4f\n", mid.Mid, spread.Spread)
```

## Constants

```go
const (
    DefaultCLOBURL  = "https://clob.polymarket.com"
    DefaultGammaURL = "https://gamma-api.polymarket.com"
)
```

## Error Handling

All methods return errors for:

- Network failures
- Non-200 HTTP responses
- JSON decoding failures

```go
markets, err := client.GetMarkets(ctx, params)
if err != nil {
    // Handle error - check for context cancellation, network issues, etc.
    log.Printf("Failed to fetch markets: %v", err)
}
```
