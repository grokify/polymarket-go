package prompts

import (
	"context"
	"errors"
	"testing"

	"github.com/plexusone/omnillm-core/provider"
)

// mockLLM implements provider.Provider for testing.
type mockLLM struct {
	response string
	err      error
}

func (m *mockLLM) CreateChatCompletion(ctx context.Context, req *provider.ChatCompletionRequest) (*provider.ChatCompletionResponse, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &provider.ChatCompletionResponse{
		Choices: []provider.ChatCompletionChoice{
			{
				Message: provider.Message{
					Role:    provider.RoleAssistant,
					Content: m.response,
				},
			},
		},
	}, nil
}

func (m *mockLLM) CreateChatCompletionStream(ctx context.Context, req *provider.ChatCompletionRequest) (provider.ChatCompletionStream, error) {
	return nil, errors.New("not implemented")
}

func (m *mockLLM) Close() error {
	return nil
}

func (m *mockLLM) Name() string {
	return "mock"
}

// mockRAGSearcher implements RAGSearcher for testing.
type mockRAGSearcher struct {
	marketResults []RAGMarketResult
	eventResults  []RAGEventResult
	err           error
}

func (m *mockRAGSearcher) SearchMarkets(ctx context.Context, query string, topK int) ([]RAGMarketResult, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.marketResults, nil
}

func (m *mockRAGSearcher) SearchEvents(ctx context.Context, query string, topK int) ([]RAGEventResult, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.eventResults, nil
}

func TestNewAgent(t *testing.T) {
	agent := NewAgent(AgentConfig{
		LLM: &mockLLM{},
	})

	if agent == nil {
		t.Fatal("NewAgent() should not return nil")
	}
	if agent.tokenLimit != 15000 {
		t.Errorf("default tokenLimit = %d, want 15000", agent.tokenLimit)
	}
	if agent.model != "claude-sonnet-4-20250514" {
		t.Errorf("default model = %q, want claude-sonnet-4-20250514", agent.model)
	}
}

func TestNewAgentCustomConfig(t *testing.T) {
	agent := NewAgent(AgentConfig{
		LLM:        &mockLLM{},
		Model:      "gpt-4",
		TokenLimit: 8000,
	})

	if agent.tokenLimit != 8000 {
		t.Errorf("tokenLimit = %d, want 8000", agent.tokenLimit)
	}
	if agent.model != "gpt-4" {
		t.Errorf("model = %q, want gpt-4", agent.model)
	}
}

func TestGetSuperforecast(t *testing.T) {
	mock := &mockLLM{
		response: "I believe this market has a likelihood 0.65 for outcome of Yes.",
	}
	agent := NewAgent(AgentConfig{LLM: mock})

	result, err := agent.GetSuperforecast(context.Background(), "Bitcoin ETF", "Will BTC ETF be approved?", "Yes")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != mock.response {
		t.Errorf("result = %q, want %q", result, mock.response)
	}
}

func TestGetSuperforecastError(t *testing.T) {
	mock := &mockLLM{err: errors.New("API error")}
	agent := NewAgent(AgentConfig{LLM: mock})

	_, err := agent.GetSuperforecast(context.Background(), "Event", "Question", "Outcome")
	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestGetMarketAnalysis(t *testing.T) {
	mock := &mockLLM{
		response: "Based on the analysis, I estimate a 70% probability.",
	}
	agent := NewAgent(AgentConfig{LLM: mock})

	result, err := agent.GetMarketAnalysis(context.Background(), "Analyze BTC market")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != mock.response {
		t.Errorf("result = %q, want %q", result, mock.response)
	}
}

func TestGetMarketAnalysisError(t *testing.T) {
	mock := &mockLLM{err: errors.New("API error")}
	agent := NewAgent(AgentConfig{LLM: mock})

	_, err := agent.GetMarketAnalysis(context.Background(), "Query")
	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestAssessRisk(t *testing.T) {
	mock := &mockLLM{
		response: "Risk score: 4/10. Moderate risk with good liquidity.",
	}
	agent := NewAgent(AgentConfig{LLM: mock})

	result, err := agent.AssessRisk(context.Background(), "BTC Market", "BUY 100", 15.5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("result should not be nil")
	}
	if result.Market != "BTC Market" {
		t.Errorf("Market = %q, want %q", result.Market, "BTC Market")
	}
	if result.Position != "BUY 100" {
		t.Errorf("Position = %q, want %q", result.Position, "BUY 100")
	}
	if result.Exposure != 15.5 {
		t.Errorf("Exposure = %f, want 15.5", result.Exposure)
	}
	if result.Analysis != mock.response {
		t.Errorf("Analysis = %q, want %q", result.Analysis, mock.response)
	}
}

func TestFilterEvents(t *testing.T) {
	mock := &mockLLM{
		response: `Based on trading potential, I recommend: ["event1", "event3"]`,
	}
	agent := NewAgent(AgentConfig{LLM: mock})

	events := []EventInfo{
		{ID: "event1", Title: "Election"},
		{ID: "event2", Title: "Sports"},
		{ID: "event3", Title: "Crypto"},
	}

	filtered, err := agent.FilterEvents(context.Background(), events)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(filtered) != 2 {
		t.Errorf("filtered count = %d, want 2", len(filtered))
	}
}

func TestFilterEventsNoJSONArray(t *testing.T) {
	// When response has no JSON array, extractJSON returns [] which parses to empty slice
	// This results in no matches (not the "conservative" case which requires actual parse error)
	mock := &mockLLM{
		response: "I think events 1 and 3 are good.", // No JSON array
	}
	agent := NewAgent(AgentConfig{LLM: mock})

	events := []EventInfo{
		{ID: "event1", Title: "Event 1"},
		{ID: "event2", Title: "Event 2"},
	}

	filtered, err := agent.FilterEvents(context.Background(), events)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// extractJSON returns [] -> empty slice -> no matches
	if len(filtered) != 0 {
		t.Errorf("filtered count = %d, want 0 (no JSON array found)", len(filtered))
	}
}

func TestFilterMarkets(t *testing.T) {
	mock := &mockLLM{
		response: `["market1"]`,
	}
	agent := NewAgent(AgentConfig{LLM: mock})

	markets := []MarketInfo{
		{ID: "market1", Question: "BTC?"},
		{ID: "market2", Question: "ETH?"},
	}

	filtered, err := agent.FilterMarkets(context.Background(), markets)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(filtered) != 1 {
		t.Errorf("filtered count = %d, want 1", len(filtered))
	}
	if filtered[0].ID != "market1" {
		t.Errorf("filtered[0].ID = %q, want market1", filtered[0].ID)
	}
}

func TestFilterEventsWithRAG(t *testing.T) {
	mock := &mockLLM{
		response: `["event1"]`,
	}
	agent := NewAgent(AgentConfig{LLM: mock})

	ragSearcher := &mockRAGSearcher{
		eventResults: []RAGEventResult{
			{ID: "event1", Score: 0.9},
			{ID: "event2", Score: 0.8},
		},
	}

	events := []EventInfo{
		{ID: "event1", Title: "Matching"},
		{ID: "event2", Title: "Also matching"},
		{ID: "event3", Title: "Not in RAG"},
	}

	filtered, err := agent.FilterEventsWithRAG(context.Background(), ragSearcher, "query", 10, events)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(filtered) != 1 {
		t.Errorf("filtered count = %d, want 1", len(filtered))
	}
}

func TestFilterEventsWithRAGNoMatches(t *testing.T) {
	mock := &mockLLM{}
	agent := NewAgent(AgentConfig{LLM: mock})

	ragSearcher := &mockRAGSearcher{
		eventResults: []RAGEventResult{}, // No matches
	}

	events := []EventInfo{
		{ID: "event1", Title: "Event"},
	}

	filtered, err := agent.FilterEventsWithRAG(context.Background(), ragSearcher, "query", 10, events)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if filtered != nil {
		t.Errorf("filtered should be nil when no RAG matches")
	}
}

func TestFilterMarketsWithRAG(t *testing.T) {
	mock := &mockLLM{
		response: `["market1", "market2"]`,
	}
	agent := NewAgent(AgentConfig{LLM: mock})

	ragSearcher := &mockRAGSearcher{
		marketResults: []RAGMarketResult{
			{ID: "market1", Score: 0.95},
			{ID: "market2", Score: 0.85},
		},
	}

	markets := []MarketInfo{
		{ID: "market1", Question: "Q1"},
		{ID: "market2", Question: "Q2"},
		{ID: "market3", Question: "Q3"},
	}

	filtered, err := agent.FilterMarketsWithRAG(context.Background(), ragSearcher, "query", 10, markets)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(filtered) != 2 {
		t.Errorf("filtered count = %d, want 2", len(filtered))
	}
}

func TestMapEventsToMarkets(t *testing.T) {
	events := []EventInfo{
		{ID: "e1", MarketIDs: []string{"m1", "m2"}},
		{ID: "e2", MarketIDs: []string{"m3"}},
		{ID: "e3", MarketIDs: []string{"m99"}}, // m99 doesn't exist
	}

	markets := []MarketInfo{
		{ID: "m1", Question: "Q1"},
		{ID: "m2", Question: "Q2"},
		{ID: "m3", Question: "Q3"},
	}

	result := MapEventsToMarkets(events, markets)

	if len(result) != 2 {
		t.Errorf("result count = %d, want 2", len(result))
	}
	if len(result["e1"]) != 2 {
		t.Errorf("e1 markets count = %d, want 2", len(result["e1"]))
	}
	if len(result["e2"]) != 1 {
		t.Errorf("e2 markets count = %d, want 1", len(result["e2"]))
	}
	if result["e3"] != nil {
		t.Error("e3 should not have markets (m99 doesn't exist)")
	}
}

func TestEstimateTokens(t *testing.T) {
	tests := []struct {
		text string
		want int
	}{
		{"", 0},
		{"test", 1},
		{"hello world", 2}, // 11 chars / 4 = 2
		{"This is a longer string with more characters", 11},
	}

	for _, tt := range tests {
		got := EstimateTokens(tt.text)
		if got != tt.want {
			t.Errorf("EstimateTokens(%q) = %d, want %d", tt.text, got, tt.want)
		}
	}
}

func TestDivideList(t *testing.T) {
	tests := []struct {
		name  string
		list  []int
		n     int
		want  int // number of chunks
		sizes []int
	}{
		{"empty list", []int{}, 3, 0, nil},
		{"n=0", []int{1, 2, 3}, 0, 0, nil},
		{"n=1", []int{1, 2, 3}, 1, 1, []int{3}},
		{"n=2", []int{1, 2, 3, 4}, 2, 2, []int{2, 2}},
		{"n=3 with remainder", []int{1, 2, 3, 4, 5}, 3, 3, []int{2, 2, 1}},
		{"n > len", []int{1, 2}, 5, 2, []int{1, 1}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DivideList(tt.list, tt.n)
			if len(result) != tt.want {
				t.Errorf("chunk count = %d, want %d", len(result), tt.want)
			}
			if tt.sizes != nil {
				for i, expectedSize := range tt.sizes {
					if i < len(result) && len(result[i]) != expectedSize {
						t.Errorf("chunk[%d] size = %d, want %d", i, len(result[i]), expectedSize)
					}
				}
			}
		})
	}
}

func TestExtractJSON(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{`["id1", "id2"]`, `["id1", "id2"]`},
		{`Here are the IDs: ["a", "b"] hope that helps`, `["a", "b"]`},
		{`no json here`, `[]`},
		{`[incomplete`, `[]`},
		{`incomplete]`, `[]`},
		{``, `[]`},
	}

	for _, tt := range tests {
		got := extractJSON(tt.input)
		if got != tt.want {
			t.Errorf("extractJSON(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestMarketInfoStructure(t *testing.T) {
	m := MarketInfo{
		ID:            "test-id",
		Question:      "Test question?",
		Description:   "Description",
		Outcomes:      []string{"Yes", "No"},
		OutcomePrices: []float64{0.6, 0.4},
		Liquidity:     50000,
		Volume:        10000,
		EndDate:       "2025-12-31",
	}

	if m.ID != "test-id" {
		t.Errorf("ID = %q, want test-id", m.ID)
	}
	if len(m.Outcomes) != 2 {
		t.Errorf("Outcomes count = %d, want 2", len(m.Outcomes))
	}
}

func TestEventInfoStructure(t *testing.T) {
	e := EventInfo{
		ID:          "event-id",
		Title:       "Event Title",
		Description: "Description",
		MarketIDs:   []string{"m1", "m2"},
		Liquidity:   100000,
		Volume:      50000,
	}

	if e.ID != "event-id" {
		t.Errorf("ID = %q, want event-id", e.ID)
	}
	if len(e.MarketIDs) != 2 {
		t.Errorf("MarketIDs count = %d, want 2", len(e.MarketIDs))
	}
}

func TestTradeRecommendationStructure(t *testing.T) {
	tr := TradeRecommendation{
		MarketID:    "market-1",
		Question:    "Will X happen?",
		Forecast:    "65% likely",
		Price:       0.65,
		Size:        0.1,
		Side:        "BUY",
		RawResponse: "raw",
	}

	if tr.MarketID != "market-1" {
		t.Errorf("MarketID = %q, want market-1", tr.MarketID)
	}
	if tr.Side != "BUY" {
		t.Errorf("Side = %q, want BUY", tr.Side)
	}
}

func TestRiskAssessmentStructure(t *testing.T) {
	ra := RiskAssessment{
		Market:   "Market",
		Position: "Position",
		Exposure: 10.5,
		Analysis: "Analysis",
	}

	if ra.Market != "Market" {
		t.Errorf("Market = %q, want Market", ra.Market)
	}
	if ra.Exposure != 10.5 {
		t.Errorf("Exposure = %f, want 10.5", ra.Exposure)
	}
}

func TestRAGMarketResultStructure(t *testing.T) {
	r := RAGMarketResult{
		ID:          "id",
		Score:       0.95,
		Question:    "Q",
		Description: "D",
		Outcomes:    "O",
		Liquidity:   "L",
		EndDate:     "E",
	}

	if r.ID != "id" {
		t.Errorf("ID = %q, want id", r.ID)
	}
	if r.Score != 0.95 {
		t.Errorf("Score = %f, want 0.95", r.Score)
	}
}

func TestRAGEventResultStructure(t *testing.T) {
	r := RAGEventResult{
		ID:          "id",
		Score:       0.8,
		Title:       "T",
		Description: "D",
		Slug:        "S",
		Liquidity:   "L",
		Volume:      "V",
	}

	if r.ID != "id" {
		t.Errorf("ID = %q, want id", r.ID)
	}
	if r.Score != 0.8 {
		t.Errorf("Score = %f, want 0.8", r.Score)
	}
}
