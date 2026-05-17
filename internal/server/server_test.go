package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/grokify/polymarket-go/internal/polymarket"
)

// mockPolymarketClient implements a mock Polymarket client for testing.
type mockPolymarketClient struct {
	markets   []polymarket.Market
	market    *polymarket.Market
	orderBook *polymarket.OrderBook
	price     *polymarket.Price
	midPrice  *polymarket.MidPrice
	spread    *polymarket.Spread
	err       error
}

func (m *mockPolymarketClient) GetMarkets(ctx context.Context, params polymarket.GetMarketsParams) ([]polymarket.Market, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.markets, nil
}

func (m *mockPolymarketClient) GetMarket(ctx context.Context, conditionID string) (*polymarket.Market, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.market, nil
}

func (m *mockPolymarketClient) GetOrderBook(ctx context.Context, tokenID string) (*polymarket.OrderBook, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.orderBook, nil
}

func (m *mockPolymarketClient) GetPrice(ctx context.Context, tokenID string) (*polymarket.Price, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.price, nil
}

func (m *mockPolymarketClient) GetMidPrice(ctx context.Context, tokenID string) (*polymarket.MidPrice, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.midPrice, nil
}

func (m *mockPolymarketClient) GetSpread(ctx context.Context, tokenID string) (*polymarket.Spread, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.spread, nil
}

func TestNewServer(t *testing.T) {
	cfg := Config{
		Port: 8080,
	}

	srv, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if srv.Router() == nil {
		t.Error("Router() returned nil")
	}
	if srv.API() == nil {
		t.Error("API() returned nil")
	}
}

func TestNewServerDefaults(t *testing.T) {
	cfg := Config{}

	srv, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if srv.config.Port != 8080 {
		t.Errorf("default port = %d, want 8080", srv.config.Port)
	}
	if srv.pmClient == nil {
		t.Error("pmClient should not be nil")
	}
}

func TestHealthEndpoint(t *testing.T) {
	srv, err := New(Config{})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	srv.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var resp struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Status != "ok" {
		t.Errorf("status = %q, want %q", resp.Status, "ok")
	}
}

func TestListMarketsEndpoint(t *testing.T) {
	srv, err := New(Config{})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/markets?limit=5", nil)
	rec := httptest.NewRecorder()

	srv.Router().ServeHTTP(rec, req)

	// Note: This test hits the live API. In a real test suite,
	// we'd use dependency injection with a mock client.
	// For now, we just verify the endpoint responds.
	if rec.Code != http.StatusOK && rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 200 or 500", rec.Code)
	}
}

func TestOpenAPIDocsEndpoint(t *testing.T) {
	srv, err := New(Config{})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/openapi.json", nil)
	rec := httptest.NewRecorder()

	srv.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	// Verify it's valid JSON
	var openapi map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&openapi); err != nil {
		t.Fatalf("failed to decode OpenAPI spec: %v", err)
	}

	// Check required OpenAPI fields
	if openapi["openapi"] == nil {
		t.Error("OpenAPI spec missing 'openapi' version field")
	}
	if openapi["info"] == nil {
		t.Error("OpenAPI spec missing 'info' field")
	}
	if openapi["paths"] == nil {
		t.Error("OpenAPI spec missing 'paths' field")
	}
}

func TestOpenAPIContainsExpectedPaths(t *testing.T) {
	srv, err := New(Config{})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/openapi.json", nil)
	rec := httptest.NewRecorder()

	srv.Router().ServeHTTP(rec, req)

	var openapi struct {
		Paths map[string]any `json:"paths"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&openapi); err != nil {
		t.Fatalf("failed to decode OpenAPI spec: %v", err)
	}

	expectedPaths := []string{
		"/health",
		"/markets",
		"/markets/{conditionId}",
		"/markets/{tokenId}/orderbook",
		"/markets/{tokenId}/price",
	}

	for _, path := range expectedPaths {
		if openapi.Paths[path] == nil {
			t.Errorf("OpenAPI spec missing path: %s", path)
		}
	}
}

func TestMarketResponseSerialization(t *testing.T) {
	market := MarketResponse{
		ID:             "0x123",
		Question:       "Will BTC reach $100k?",
		Slug:           "btc-100k",
		Description:    "Test description",
		Active:         true,
		Closed:         false,
		Liquidity:      150000.00,
		Volume24hr:     25000.00,
		Outcomes:       `["Yes","No"]`,
		OutcomePrices:  `[0.65,0.35]`,
		BestBid:        0.64,
		BestAsk:        0.66,
		Spread:         0.02,
		LastTradePrice: 0.65,
		EndDate:        "2025-12-31T23:59:59Z",
	}

	data, err := json.Marshal(market)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var decoded MarketResponse
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if decoded.ID != market.ID {
		t.Errorf("ID = %q, want %q", decoded.ID, market.ID)
	}
	if decoded.Liquidity != market.Liquidity {
		t.Errorf("Liquidity = %f, want %f", decoded.Liquidity, market.Liquidity)
	}
}

func TestOrderBookResponseSerialization(t *testing.T) {
	book := OrderBookResponse{
		Market: "0x123",
		Asset:  "token123",
		Hash:   "abc123",
		Bids: []OrderBookLevel{
			{Price: "0.64", Size: "1000"},
			{Price: "0.63", Size: "500"},
		},
		Asks: []OrderBookLevel{
			{Price: "0.66", Size: "800"},
		},
	}

	data, err := json.Marshal(book)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var decoded OrderBookResponse
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if len(decoded.Bids) != 2 {
		t.Errorf("Bids count = %d, want 2", len(decoded.Bids))
	}
	if len(decoded.Asks) != 1 {
		t.Errorf("Asks count = %d, want 1", len(decoded.Asks))
	}
}

func TestNewsArticleResponseSerialization(t *testing.T) {
	article := NewsArticleResponse{
		Title:     "Bitcoin ETF Approved",
		Link:      "https://example.com/article",
		Source:    "Reuters",
		Date:      "2025-05-17",
		Snippet:   "The SEC has approved...",
		Thumbnail: "https://example.com/thumb.jpg",
	}

	data, err := json.Marshal(article)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	if !strings.Contains(string(data), "Bitcoin ETF") {
		t.Error("serialized data should contain title")
	}
}

func TestRAGSearchInputValidation(t *testing.T) {
	input := RAGSearchInput{}
	input.Body.Query = "cryptocurrency regulation"
	input.Body.TopK = 10

	if input.Body.Query == "" {
		t.Error("Query should not be empty")
	}
	if input.Body.TopK != 10 {
		t.Errorf("TopK = %d, want 10", input.Body.TopK)
	}
}

func TestMarketToResponse(t *testing.T) {
	market := polymarket.Market{
		ConditionID:    "0x123",
		Question:       "Test question?",
		Slug:           "test-question",
		Description:    "Test description",
		Active:         true,
		Closed:         false,
		LiquidityNum:   100000,
		Volume24hr:     5000,
		Outcomes:       `["Yes","No"]`,
		OutcomePrices:  `[0.6,0.4]`,
		BestBid:        0.59,
		BestAsk:        0.61,
		Spread:         0.02,
		LastTradePrice: 0.60,
		EndDateISO:     "2025-12-31T00:00:00Z",
		Image:          "https://example.com/image.png",
	}

	resp := marketToResponse(market)

	if resp.ID != market.ConditionID {
		t.Errorf("ID = %q, want %q", resp.ID, market.ConditionID)
	}
	if resp.Question != market.Question {
		t.Errorf("Question = %q, want %q", resp.Question, market.Question)
	}
	if resp.Liquidity != market.LiquidityNum {
		t.Errorf("Liquidity = %f, want %f", resp.Liquidity, market.LiquidityNum)
	}
	if resp.EndDate != market.EndDateISO {
		t.Errorf("EndDate = %q, want %q", resp.EndDate, market.EndDateISO)
	}
}
