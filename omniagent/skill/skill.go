// Package skill provides an omniagent skill wrapper for polymarket-go.
//
// This package adapts the polymarket-go SDK to the omniagent skill interface,
// allowing prediction market analysis capabilities to be compiled into omniagent-based agents.
package skill

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/grokify/polymarket-go/internal/polymarket"
	"github.com/plexusone/omniagent/skills/compiled"
	"github.com/plexusone/omniskill/skill"
)

// Verify interface compliance at compile time.
var _ compiled.Skill = (*Skill)(nil)

// Skill implements omniagent compiled.Skill for prediction market analysis.
type Skill struct {
	client *polymarket.Client
	config Config
}

// Config configures the predictions skill.
type Config struct {
	// CacheMarkets enables caching of market data (future).
	CacheMarkets bool

	// CacheTTL is the cache duration (default: 5m).
	CacheTTL string
}

// New creates a new prediction market skill.
func New(cfg Config) *Skill {
	client := polymarket.NewClient()

	return &Skill{
		client: client,
		config: cfg,
	}
}

// Name returns the skill name.
func (s *Skill) Name() string {
	return "predictions"
}

// Description returns the skill description.
func (s *Skill) Description() string {
	return "Prediction market analysis tools for Polymarket"
}

// Tools returns the tools provided by this skill.
func (s *Skill) Tools() []skill.Tool {
	return []skill.Tool{
		skill.NewTool(
			"get_markets",
			"Fetch active prediction markets from Polymarket. Returns markets with pricing, liquidity, and resolution dates. Use filters to narrow results.",
			map[string]skill.Parameter{
				"min_liquidity": {
					Type:        "number",
					Description: "Minimum liquidity in USD (e.g., 10000 for $10k minimum)",
					Required:    false,
				},
				"category": {
					Type:        "string",
					Description: "Market category filter (e.g., politics, sports, crypto, entertainment)",
					Required:    false,
				},
				"limit": {
					Type:        "integer",
					Description: "Maximum number of markets to return (default: 10, max: 100)",
					Required:    false,
					Default:     10,
				},
				"text_query": {
					Type:        "string",
					Description: "Search query to filter markets by question text",
					Required:    false,
				},
			},
			s.getMarkets,
		),
		skill.NewTool(
			"get_orderbook",
			"Fetch the order book for a specific market token, showing current bids and asks with mid price and spread.",
			map[string]skill.Parameter{
				"token_id": {
					Type:        "string",
					Description: "The token ID of the market outcome (from get_markets clobTokenIds field)",
					Required:    true,
				},
			},
			s.getOrderBook,
		),
	}
}

// Init initializes the skill.
func (s *Skill) Init(ctx context.Context) error {
	return nil
}

// Close releases resources.
func (s *Skill) Close() error {
	return nil
}

// getMarkets fetches active prediction markets.
func (s *Skill) getMarkets(ctx context.Context, params map[string]any) (any, error) {
	var minLiquidity float64
	var category string
	var limit int = 10
	var textQuery string

	if v, ok := params["min_liquidity"].(float64); ok {
		minLiquidity = v
	}
	if v, ok := params["category"].(string); ok {
		category = v
	}
	if v, ok := params["limit"].(float64); ok {
		limit = int(v)
	} else if v, ok := params["limit"].(int); ok {
		limit = v
	}
	if v, ok := params["text_query"].(string); ok {
		textQuery = v
	}

	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	active := true
	marketsData, err := s.client.GetMarkets(ctx, polymarket.GetMarketsParams{
		Active:    &active,
		Limit:     limit,
		LiqMin:    minLiquidity,
		TagSlug:   category,
		TextQuery: textQuery,
		Order:     "liquidity",
	})
	if err != nil {
		return nil, fmt.Errorf("fetching markets: %w", err)
	}

	// Format markets for LLM consumption
	markets := make([]map[string]any, 0, len(marketsData))
	for _, m := range marketsData {
		market := map[string]any{
			"id":             m.ID,
			"question":       m.Question,
			"description":    m.Description,
			"liquidity":      m.LiquidityNum,
			"volume_24h":     m.Volume24hr,
			"end_date":       m.EndDateISO,
			"slug":           m.Slug,
			"best_bid":       m.BestBid,
			"best_ask":       m.BestAsk,
			"spread":         m.Spread,
			"last_price":     m.LastTradePrice,
			"clob_token_ids": m.ClobTokenIDs,
		}
		markets = append(markets, market)
	}

	return map[string]any{
		"markets": markets,
		"count":   len(markets),
	}, nil
}

// getOrderBook fetches the order book for a market token.
func (s *Skill) getOrderBook(ctx context.Context, params map[string]any) (any, error) {
	tokenID, ok := params["token_id"].(string)
	if !ok || tokenID == "" {
		return nil, fmt.Errorf("token_id is required")
	}

	book, err := s.client.GetOrderBook(ctx, tokenID)
	if err != nil {
		return nil, fmt.Errorf("fetching order book: %w", err)
	}

	// Convert order book levels to more readable format
	bids := make([]map[string]string, 0, len(book.Bids))
	for _, b := range book.Bids {
		bids = append(bids, map[string]string{
			"price": b.Price,
			"size":  b.Size,
		})
	}

	asks := make([]map[string]string, 0, len(book.Asks))
	for _, a := range book.Asks {
		asks = append(asks, map[string]string{
			"price": a.Price,
			"size":  a.Size,
		})
	}

	result := map[string]any{
		"token_id": tokenID,
		"bids":     bids,
		"asks":     asks,
	}

	// Also get mid price and spread for context
	if mid, err := s.client.GetMidPrice(ctx, tokenID); err == nil && mid != nil {
		result["mid_price"] = mid.Mid
	}
	if spread, err := s.client.GetSpread(ctx, tokenID); err == nil && spread != nil {
		result["spread"] = spread.Spread
	}

	return result, nil
}

// MarshalJSON provides JSON output for skill metadata.
func (s *Skill) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]any{
		"name":        s.Name(),
		"description": s.Description(),
		"tools_count": len(s.Tools()),
	})
}
