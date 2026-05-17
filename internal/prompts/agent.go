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

// RAGSearcher is an interface for RAG-based semantic search.
// This allows the agent to use RAG without a direct dependency on the rag package.
type RAGSearcher interface {
	SearchMarkets(ctx context.Context, query string, topK int) ([]RAGMarketResult, error)
	SearchEvents(ctx context.Context, query string, topK int) ([]RAGEventResult, error)
}

// RAGMarketResult represents a market search result from RAG.
type RAGMarketResult struct {
	ID          string  `json:"id"`
	Score       float32 `json:"score"`
	Question    string  `json:"question"`
	Description string  `json:"description"`
	Outcomes    string  `json:"outcomes"`
	Liquidity   string  `json:"liquidity"`
	EndDate     string  `json:"end_date"`
}

// RAGEventResult represents an event search result from RAG.
type RAGEventResult struct {
	ID          string  `json:"id"`
	Score       float32 `json:"score"`
	Title       string  `json:"title"`
	Description string  `json:"description"`
	Slug        string  `json:"slug"`
	Liquidity   string  `json:"liquidity"`
	Volume      string  `json:"volume"`
}

// FilterEventsWithRAG uses semantic search to filter events based on a query.
// It first searches for semantically similar events, then uses the LLM to refine the selection.
func (a *Agent) FilterEventsWithRAG(ctx context.Context, rag RAGSearcher, query string, topK int, events []EventInfo) ([]EventInfo, error) {
	// Step 1: Use RAG to find semantically similar events
	ragResults, err := rag.SearchEvents(ctx, query, topK*2) // Get more results for LLM to refine
	if err != nil {
		return nil, fmt.Errorf("RAG search failed: %w", err)
	}

	// Build a set of RAG-matched IDs with scores
	ragMatches := make(map[string]float32)
	for _, r := range ragResults {
		ragMatches[r.ID] = r.Score
	}

	// Filter events to only those that match RAG results
	var candidates []EventInfo
	for _, e := range events {
		if _, ok := ragMatches[e.ID]; ok {
			candidates = append(candidates, e)
		}
	}

	if len(candidates) == 0 {
		return nil, nil
	}

	// Step 2: Use LLM to further filter based on trading criteria
	return a.FilterEvents(ctx, candidates)
}

// FilterMarketsWithRAG uses semantic search to filter markets based on a query.
// It first searches for semantically similar markets, then uses the LLM to refine the selection.
func (a *Agent) FilterMarketsWithRAG(ctx context.Context, rag RAGSearcher, query string, topK int, markets []MarketInfo) ([]MarketInfo, error) {
	// Step 1: Use RAG to find semantically similar markets
	ragResults, err := rag.SearchMarkets(ctx, query, topK*2)
	if err != nil {
		return nil, fmt.Errorf("RAG search failed: %w", err)
	}

	// Build a set of RAG-matched IDs with scores
	ragMatches := make(map[string]float32)
	for _, r := range ragResults {
		ragMatches[r.ID] = r.Score
	}

	// Filter markets to only those that match RAG results
	var candidates []MarketInfo
	for _, m := range markets {
		if _, ok := ragMatches[m.ID]; ok {
			candidates = append(candidates, m)
		}
	}

	if len(candidates) == 0 {
		return nil, nil
	}

	// Step 2: Use LLM to further filter based on trading criteria
	return a.FilterMarkets(ctx, candidates)
}

// MapEventsToMarkets maps events to their associated markets.
// Returns a map of event ID to list of markets.
func MapEventsToMarkets(events []EventInfo, markets []MarketInfo) map[string][]MarketInfo {
	// Build a map of market ID to market
	marketByID := make(map[string]MarketInfo)
	for _, m := range markets {
		marketByID[m.ID] = m
	}

	// Map events to their markets
	result := make(map[string][]MarketInfo)
	for _, e := range events {
		var eventMarkets []MarketInfo
		for _, mid := range e.MarketIDs {
			if m, ok := marketByID[mid]; ok {
				eventMarkets = append(eventMarkets, m)
			}
		}
		if len(eventMarkets) > 0 {
			result[e.ID] = eventMarkets
		}
	}

	return result
}

// ProcessDataChunks processes data in chunks to avoid context limits.
// It calls the processor function for each chunk and aggregates results.
func (a *Agent) ProcessDataChunks(ctx context.Context, data []MarketInfo, processor func(ctx context.Context, chunk []MarketInfo) ([]MarketInfo, error)) ([]MarketInfo, error) {
	// Estimate tokens per market (rough estimate)
	tokensPerMarket := 200 // ~200 tokens per market JSON representation
	maxMarketsPerChunk := a.tokenLimit / tokensPerMarket

	if maxMarketsPerChunk < 1 {
		maxMarketsPerChunk = 1
	}

	// Divide into chunks
	numChunks := (len(data) + maxMarketsPerChunk - 1) / maxMarketsPerChunk
	chunks := DivideList(data, numChunks)

	var allResults []MarketInfo
	for _, chunk := range chunks {
		results, err := processor(ctx, chunk)
		if err != nil {
			return nil, fmt.Errorf("processing chunk: %w", err)
		}
		allResults = append(allResults, results...)
	}

	return allResults, nil
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
