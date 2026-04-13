// Package tools provides Polymarket-specific tools for agents.
package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/grokify/polymarket-go/internal/polymarket"
)

// MarketTool provides market discovery and analysis capabilities.
type MarketTool struct {
	client *polymarket.Client
}

// NewMarketTool creates a new MarketTool.
func NewMarketTool(client *polymarket.Client) *MarketTool {
	return &MarketTool{client: client}
}

// Name returns the tool name.
func (t *MarketTool) Name() string {
	return "get_markets"
}

// Description returns the tool description.
func (t *MarketTool) Description() string {
	return "Fetch active prediction markets from Polymarket with optional filters for liquidity, resolution date, and categories"
}

// Parameters returns the JSON schema for tool parameters.
func (t *MarketTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"min_liquidity": map[string]any{
				"type":        "number",
				"description": "Minimum liquidity in USD",
			},
			"max_days_to_resolution": map[string]any{
				"type":        "integer",
				"description": "Maximum days until market resolution",
			},
			"category": map[string]any{
				"type":        "string",
				"description": "Market category filter (e.g., politics, sports, crypto)",
			},
			"limit": map[string]any{
				"type":        "integer",
				"description": "Maximum number of markets to return",
				"default":     10,
			},
		},
	}
}

// Call executes the tool with the given input.
func (t *MarketTool) Call(ctx context.Context, input string) (string, error) {
	var params struct {
		MinLiquidity        float64 `json:"min_liquidity"`
		MaxDaysToResolution int     `json:"max_days_to_resolution"`
		Category            string  `json:"category"`
		Limit               int     `json:"limit"`
		TextQuery           string  `json:"text_query"`
	}

	if err := json.Unmarshal([]byte(input), &params); err != nil {
		return "", fmt.Errorf("invalid parameters: %w", err)
	}

	if params.Limit == 0 {
		params.Limit = 10
	}

	active := true
	marketsData, err := t.client.GetMarkets(ctx, polymarket.GetMarketsParams{
		Active:    &active,
		Limit:     params.Limit,
		LiqMin:    params.MinLiquidity,
		TagSlug:   params.Category,
		TextQuery: params.TextQuery,
		Order:     "liquidity",
	})
	if err != nil {
		return "", fmt.Errorf("fetching markets: %w", err)
	}

	// Format markets for LLM consumption
	markets := make([]map[string]any, 0, len(marketsData))
	for _, m := range marketsData {
		markets = append(markets, map[string]any{
			"id":          m.ID,
			"question":    m.Question,
			"description": m.Description,
			"liquidity":   m.LiquidityNum,
			"volume_24h":  m.Volume24hr,
			"end_date":    m.EndDateISO,
			"slug":        m.Slug,
			"best_bid":    m.BestBid,
			"best_ask":    m.BestAsk,
			"spread":      m.Spread,
			"last_price":  m.LastTradePrice,
		})
	}

	result := map[string]any{
		"markets": markets,
		"count":   len(markets),
	}

	output, _ := json.MarshalIndent(result, "", "  ")
	return string(output), nil
}

// OrderBookTool provides order book data for markets.
type OrderBookTool struct {
	client *polymarket.Client
}

// NewOrderBookTool creates a new OrderBookTool.
func NewOrderBookTool(client *polymarket.Client) *OrderBookTool {
	return &OrderBookTool{client: client}
}

// Name returns the tool name.
func (t *OrderBookTool) Name() string {
	return "get_orderbook"
}

// Description returns the tool description.
func (t *OrderBookTool) Description() string {
	return "Fetch the order book for a specific market token, showing current bids and asks"
}

// Parameters returns the JSON schema for tool parameters.
func (t *OrderBookTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"token_id": map[string]any{
				"type":        "string",
				"description": "The token ID of the market outcome",
			},
		},
		"required": []string{"token_id"},
	}
}

// Call executes the tool with the given input.
func (t *OrderBookTool) Call(ctx context.Context, input string) (string, error) {
	var params struct {
		TokenID string `json:"token_id"`
	}

	if err := json.Unmarshal([]byte(input), &params); err != nil {
		return "", fmt.Errorf("invalid parameters: %w", err)
	}

	if params.TokenID == "" {
		return "", fmt.Errorf("token_id is required")
	}

	book, err := t.client.GetOrderBook(ctx, params.TokenID)
	if err != nil {
		return "", fmt.Errorf("fetching order book: %w", err)
	}

	// Also get mid price and spread for context
	mid, _ := t.client.GetMidPrice(ctx, params.TokenID)
	spread, _ := t.client.GetSpread(ctx, params.TokenID)

	result := map[string]any{
		"token_id": params.TokenID,
		"bids":     book.Bids,
		"asks":     book.Asks,
	}
	if mid != nil {
		result["mid_price"] = mid.Mid
	}
	if spread != nil {
		result["spread"] = spread.Spread
	}

	output, _ := json.MarshalIndent(result, "", "  ")
	return string(output), nil
}

// PlaceOrderTool places orders on Polymarket.
// Note: Order placement requires authentication with a private key.
type PlaceOrderTool struct {
	client *polymarket.Client
	// TODO: Add signer for authenticated requests
}

// NewPlaceOrderTool creates a new PlaceOrderTool.
func NewPlaceOrderTool(client *polymarket.Client) *PlaceOrderTool {
	return &PlaceOrderTool{client: client}
}

// Name returns the tool name.
func (t *PlaceOrderTool) Name() string {
	return "place_order"
}

// Description returns the tool description.
func (t *PlaceOrderTool) Description() string {
	return "Place a limit order on Polymarket for a specific market outcome"
}

// Parameters returns the JSON schema for tool parameters.
func (t *PlaceOrderTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"token_id": map[string]any{
				"type":        "string",
				"description": "The token ID of the market outcome",
			},
			"side": map[string]any{
				"type":        "string",
				"enum":        []string{"buy", "sell"},
				"description": "Order side: buy or sell",
			},
			"price": map[string]any{
				"type":        "number",
				"description": "Limit price (0.01 to 0.99)",
				"minimum":     0.01,
				"maximum":     0.99,
			},
			"size": map[string]any{
				"type":        "number",
				"description": "Order size in shares",
				"minimum":     0.01,
			},
		},
		"required": []string{"token_id", "side", "price", "size"},
	}
}

// Call executes the tool with the given input.
func (t *PlaceOrderTool) Call(ctx context.Context, input string) (string, error) {
	var params struct {
		TokenID string  `json:"token_id"`
		Side    string  `json:"side"`
		Price   float64 `json:"price"`
		Size    float64 `json:"size"`
	}

	if err := json.Unmarshal([]byte(input), &params); err != nil {
		return "", fmt.Errorf("invalid parameters: %w", err)
	}

	// Validate parameters
	if params.TokenID == "" {
		return "", fmt.Errorf("token_id is required")
	}
	if params.Side != "buy" && params.Side != "sell" {
		return "", fmt.Errorf("side must be 'buy' or 'sell'")
	}
	if params.Price < 0.01 || params.Price > 0.99 {
		return "", fmt.Errorf("price must be between 0.01 and 0.99")
	}
	if params.Size < 0.01 {
		return "", fmt.Errorf("size must be at least 0.01")
	}

	// TODO: Implement actual order placement via polymarket-kit
	result := map[string]any{
		"status":  "not_implemented",
		"message": "Order placement pending polymarket-kit integration",
		"order": map[string]any{
			"token_id": params.TokenID,
			"side":     params.Side,
			"price":    params.Price,
			"size":     params.Size,
		},
	}

	output, _ := json.MarshalIndent(result, "", "  ")
	return string(output), nil
}
