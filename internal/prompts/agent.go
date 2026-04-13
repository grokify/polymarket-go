// Package prompts provides prompt templates and agent execution for Polymarket trading.
package prompts

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"

	"github.com/plexusone/omnillm-core/provider"
)

// Agent executes prompts using an LLM provider.
type Agent struct {
	llm        provider.Provider
	model      string
	prompter   *Prompter
	tokenLimit int
}

// AgentConfig holds configuration for the agent.
type AgentConfig struct {
	LLM        provider.Provider
	Model      string
	TokenLimit int // Max tokens for context, default 15000
}

// NewAgent creates a new Agent.
func NewAgent(cfg AgentConfig) *Agent {
	if cfg.TokenLimit == 0 {
		cfg.TokenLimit = 15000
	}
	if cfg.Model == "" {
		cfg.Model = "claude-sonnet-4-20250514"
	}
	return &Agent{
		llm:        cfg.LLM,
		model:      cfg.Model,
		prompter:   NewPrompter(),
		tokenLimit: cfg.TokenLimit,
	}
}

// GetSuperforecast generates a superforecaster prediction.
func (a *Agent) GetSuperforecast(ctx context.Context, eventTitle, marketQuestion, outcome string) (string, error) {
	prompt := a.prompter.Superforecaster(marketQuestion, eventTitle, outcome)

	resp, err := a.llm.CreateChatCompletion(ctx, &provider.ChatCompletionRequest{
		Model: a.model,
		Messages: []provider.Message{
			{Role: provider.RoleUser, Content: prompt},
		},
	})
	if err != nil {
		return "", fmt.Errorf("LLM call failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no response from LLM")
	}

	return resp.Choices[0].Message.Content, nil
}

// GetMarketAnalysis generates market analysis for a user query.
func (a *Agent) GetMarketAnalysis(ctx context.Context, userInput string) (string, error) {
	systemPrompt := a.prompter.MarketAnalyst()

	resp, err := a.llm.CreateChatCompletion(ctx, &provider.ChatCompletionRequest{
		Model: a.model,
		Messages: []provider.Message{
			{Role: provider.RoleSystem, Content: systemPrompt},
			{Role: provider.RoleUser, Content: userInput},
		},
	})
	if err != nil {
		return "", fmt.Errorf("LLM call failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no response from LLM")
	}

	return resp.Choices[0].Message.Content, nil
}

// SourceBestTrade analyzes a market and recommends a trade.
func (a *Agent) SourceBestTrade(ctx context.Context, market MarketInfo) (*TradeRecommendation, error) {
	// Step 1: Get superforecast
	forecast, err := a.GetSuperforecast(ctx, market.Description, market.Question, strings.Join(market.Outcomes, ", "))
	if err != nil {
		return nil, fmt.Errorf("getting forecast: %w", err)
	}

	// Step 2: Get trade recommendation
	outcomePricesJSON, _ := json.Marshal(market.OutcomePrices)
	tradePrompt := a.prompter.OneBestTrade(forecast, market.Outcomes, string(outcomePricesJSON))

	resp, err := a.llm.CreateChatCompletion(ctx, &provider.ChatCompletionRequest{
		Model: a.model,
		Messages: []provider.Message{
			{Role: provider.RoleUser, Content: tradePrompt},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("LLM call failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no response from LLM")
	}

	// Step 3: Parse the response
	tradeResponse := resp.Choices[0].Message.Content
	price, size, side, err := ParseTradeResponse(tradeResponse)
	if err != nil {
		return nil, fmt.Errorf("parsing trade response: %w", err)
	}

	return &TradeRecommendation{
		MarketID:    market.ID,
		Question:    market.Question,
		Forecast:    forecast,
		Price:       price,
		Size:        size,
		Side:        side,
		RawResponse: tradeResponse,
	}, nil
}

// filterByIDs calls the LLM with a prompt and extracts a JSON array of IDs from the response.
func (a *Agent) filterByIDs(ctx context.Context, prompt string) (map[string]bool, error) {
	resp, err := a.llm.CreateChatCompletion(ctx, &provider.ChatCompletionRequest{
		Model: a.model,
		Messages: []provider.Message{
			{Role: provider.RoleUser, Content: prompt},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("LLM call failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no response from LLM")
	}

	content := resp.Choices[0].Message.Content
	var selectedIDs []string
	if err := json.Unmarshal([]byte(extractJSON(content)), &selectedIDs); err != nil {
		return nil, nil // Return nil to signal "keep all" on parse failure
	}

	idSet := make(map[string]bool)
	for _, id := range selectedIDs {
		idSet[id] = true
	}
	return idSet, nil
}

// FilterEvents uses the LLM to filter events for trading opportunities.
func (a *Agent) FilterEvents(ctx context.Context, events []EventInfo) ([]EventInfo, error) {
	eventsJSON, err := json.MarshalIndent(events, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshaling events: %w", err)
	}

	prompt := a.prompter.FilterEvents() + "\n\nEvents:\n" + string(eventsJSON) + "\n\nReturn a JSON array of event IDs to trade on."

	idSet, err := a.filterByIDs(ctx, prompt)
	if err != nil {
		return nil, err
	}
	if idSet == nil {
		return events, nil // Conservative: return all on parse failure
	}

	var filtered []EventInfo
	for _, e := range events {
		if idSet[e.ID] {
			filtered = append(filtered, e)
		}
	}
	return filtered, nil
}

// FilterMarkets uses the LLM to filter markets for trading opportunities.
func (a *Agent) FilterMarkets(ctx context.Context, markets []MarketInfo) ([]MarketInfo, error) {
	marketsJSON, err := json.MarshalIndent(markets, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshaling markets: %w", err)
	}

	prompt := a.prompter.FilterMarkets() + "\n\nMarkets:\n" + string(marketsJSON) + "\n\nReturn a JSON array of market IDs to trade on."

	idSet, err := a.filterByIDs(ctx, prompt)
	if err != nil {
		return nil, err
	}
	if idSet == nil {
		return markets, nil // Conservative: return all on parse failure
	}

	var filtered []MarketInfo
	for _, m := range markets {
		if idSet[m.ID] {
			filtered = append(filtered, m)
		}
	}
	return filtered, nil
}

// AssessRisk evaluates the risk of a proposed trade.
func (a *Agent) AssessRisk(ctx context.Context, market, position string, exposure float64) (*RiskAssessment, error) {
	prompt := a.prompter.RiskAssessment(market, position, exposure)

	resp, err := a.llm.CreateChatCompletion(ctx, &provider.ChatCompletionRequest{
		Model: a.model,
		Messages: []provider.Message{
			{Role: provider.RoleUser, Content: prompt},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("LLM call failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no response from LLM")
	}

	return &RiskAssessment{
		Market:   market,
		Position: position,
		Exposure: exposure,
		Analysis: resp.Choices[0].Message.Content,
	}, nil
}

// EstimateTokens estimates the number of tokens in a string.
// This is a rough estimate assuming ~4 characters per token.
func EstimateTokens(text string) int {
	return len(text) / 4
}

// DivideList splits a list into n roughly equal parts.
func DivideList[T any](list []T, n int) [][]T {
	if n <= 0 {
		return nil
	}
	size := int(math.Ceil(float64(len(list)) / float64(n)))
	var result [][]T
	for i := 0; i < len(list); i += size {
		end := i + size
		if end > len(list) {
			end = len(list)
		}
		result = append(result, list[i:end])
	}
	return result
}

// extractJSON attempts to extract a JSON array from a string.
func extractJSON(s string) string {
	start := strings.Index(s, "[")
	end := strings.LastIndex(s, "]")
	if start >= 0 && end > start {
		return s[start : end+1]
	}
	return "[]"
}

// MarketInfo holds market information for analysis.
type MarketInfo struct {
	ID            string    `json:"id"`
	Question      string    `json:"question"`
	Description   string    `json:"description"`
	Outcomes      []string  `json:"outcomes"`
	OutcomePrices []float64 `json:"outcome_prices"`
	Liquidity     float64   `json:"liquidity"`
	Volume        float64   `json:"volume"`
	EndDate       string    `json:"end_date,omitempty"`
}

// EventInfo holds event information for analysis.
type EventInfo struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Description string   `json:"description,omitempty"`
	MarketIDs   []string `json:"market_ids,omitempty"`
	Liquidity   float64  `json:"liquidity"`
	Volume      float64  `json:"volume"`
}

// TradeRecommendation holds a trade recommendation from the agent.
type TradeRecommendation struct {
	MarketID    string  `json:"market_id"`
	Question    string  `json:"question"`
	Forecast    string  `json:"forecast"`
	Price       float64 `json:"price"`
	Size        float64 `json:"size"` // As percentage of bankroll
	Side        string  `json:"side"` // "BUY" or "SELL"
	RawResponse string  `json:"raw_response,omitempty"`
}

// RiskAssessment holds a risk assessment for a trade.
type RiskAssessment struct {
	Market   string  `json:"market"`
	Position string  `json:"position"`
	Exposure float64 `json:"exposure"`
	Analysis string  `json:"analysis"`
}
