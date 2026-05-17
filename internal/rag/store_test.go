package rag

import (
	"context"
	"errors"
	"testing"

	"github.com/grokify/polymarket-go/internal/polymarket"
	"github.com/plexusone/omniretrieve/retrieve"
	"github.com/plexusone/omniretrieve/vector"
)

// mockEmbedder implements Embedder for testing.
type mockEmbedder struct {
	embedding  []float32
	embeddings [][]float32
	err        error
}

func (m *mockEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.embedding, nil
}

func (m *mockEmbedder) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.embeddings != nil {
		return m.embeddings, nil
	}
	// Generate a slice of embeddings for each text
	result := make([][]float32, len(texts))
	for i := range texts {
		result[i] = m.embedding
	}
	return result, nil
}

// mockVectorIndex implements vector.Index for testing.
type mockVectorIndex struct {
	nodes       map[string]vector.Node
	results     []vector.SearchResult
	err         error
	upsertCalls int
}

func newMockVectorIndex() *mockVectorIndex {
	return &mockVectorIndex{
		nodes: make(map[string]vector.Node),
	}
}

func (m *mockVectorIndex) Search(ctx context.Context, embedding []float32, k int, filters map[string]string) ([]vector.SearchResult, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.results, nil
}

func (m *mockVectorIndex) Insert(ctx context.Context, node vector.Node) error {
	if m.err != nil {
		return m.err
	}
	m.nodes[node.ID] = node
	return nil
}

func (m *mockVectorIndex) Upsert(ctx context.Context, node vector.Node) error {
	if m.err != nil {
		return m.err
	}
	m.nodes[node.ID] = node
	m.upsertCalls++
	return nil
}

func (m *mockVectorIndex) Delete(ctx context.Context, id string) error {
	if m.err != nil {
		return m.err
	}
	delete(m.nodes, id)
	return nil
}

func (m *mockVectorIndex) Name() string {
	return "mock-index"
}

func TestNewStore(t *testing.T) {
	tests := []struct {
		name      string
		cfg       StoreConfig
		wantErr   bool
		errSubstr string
	}{
		{
			name:      "missing vector index",
			cfg:       StoreConfig{Embedder: &mockEmbedder{}},
			wantErr:   true,
			errSubstr: "vector index is required",
		},
		{
			name:      "missing embedder",
			cfg:       StoreConfig{VectorIndex: newMockVectorIndex()},
			wantErr:   true,
			errSubstr: "embedder is required",
		},
		{
			name: "valid config",
			cfg: StoreConfig{
				VectorIndex: newMockVectorIndex(),
				Embedder:    &mockEmbedder{},
			},
			wantErr: false,
		},
		{
			name: "custom dimensions",
			cfg: StoreConfig{
				VectorIndex: newMockVectorIndex(),
				Embedder:    &mockEmbedder{},
				Dimensions:  768,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store, err := NewStore(tt.cfg)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				} else if tt.errSubstr != "" && !containsString(err.Error(), tt.errSubstr) {
					t.Errorf("error %q should contain %q", err.Error(), tt.errSubstr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if store == nil {
				t.Error("store should not be nil")
			}
		})
	}
}

func TestNewStoreDefaultDimensions(t *testing.T) {
	store, err := NewStore(StoreConfig{
		VectorIndex: newMockVectorIndex(),
		Embedder:    &mockEmbedder{},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if store.config.Dimensions != 1536 {
		t.Errorf("default dimensions = %d, want 1536", store.config.Dimensions)
	}
}

func TestIndexMarketsEmpty(t *testing.T) {
	idx := newMockVectorIndex()
	store, _ := NewStore(StoreConfig{
		VectorIndex: idx,
		Embedder:    &mockEmbedder{embedding: []float32{0.1, 0.2, 0.3}},
	})

	err := store.IndexMarkets(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if idx.upsertCalls != 0 {
		t.Errorf("upsertCalls = %d, want 0", idx.upsertCalls)
	}
}

func TestIndexMarkets(t *testing.T) {
	idx := newMockVectorIndex()
	store, _ := NewStore(StoreConfig{
		VectorIndex: idx,
		Embedder:    &mockEmbedder{embedding: []float32{0.1, 0.2, 0.3}},
	})

	markets := []polymarket.Market{
		{
			ConditionID:  "market1",
			Question:     "Will Bitcoin reach $100k?",
			Description:  "Test description",
			Outcomes:     `["Yes","No"]`,
			LiquidityNum: 50000,
			Volume24hr:   10000,
			EndDateISO:   "2025-12-31T00:00:00Z",
		},
		{
			ConditionID: "market2",
			Question:    "Will ETH reach $10k?",
			Description: "Another test",
		},
	}

	err := store.IndexMarkets(context.Background(), markets)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if idx.upsertCalls != 2 {
		t.Errorf("upsertCalls = %d, want 2", idx.upsertCalls)
	}

	if len(idx.nodes) != 2 {
		t.Errorf("nodes count = %d, want 2", len(idx.nodes))
	}

	// Check node metadata
	node, ok := idx.nodes["market1"]
	if !ok {
		t.Fatal("market1 not found in index")
	}
	if node.Metadata["type"] != "market" {
		t.Errorf("metadata type = %q, want %q", node.Metadata["type"], "market")
	}
	if node.Metadata["question"] != "Will Bitcoin reach $100k?" {
		t.Errorf("metadata question = %q", node.Metadata["question"])
	}
}

func TestIndexMarketsEmbedError(t *testing.T) {
	embedErr := errors.New("embedding service unavailable")
	store, _ := NewStore(StoreConfig{
		VectorIndex: newMockVectorIndex(),
		Embedder:    &mockEmbedder{err: embedErr},
	})

	markets := []polymarket.Market{{ConditionID: "test"}}
	err := store.IndexMarkets(context.Background(), markets)

	if err == nil {
		t.Error("expected error, got nil")
	}
	if !containsString(err.Error(), "generating embeddings") {
		t.Errorf("error should mention embeddings: %v", err)
	}
}

func TestIndexMarketsUpsertError(t *testing.T) {
	idx := newMockVectorIndex()
	idx.err = errors.New("database connection failed")

	store, _ := NewStore(StoreConfig{
		VectorIndex: idx,
		Embedder:    &mockEmbedder{embedding: []float32{0.1}},
	})

	markets := []polymarket.Market{{ConditionID: "test"}}
	err := store.IndexMarkets(context.Background(), markets)

	if err == nil {
		t.Error("expected error, got nil")
	}
	if !containsString(err.Error(), "indexing market") {
		t.Errorf("error should mention indexing: %v", err)
	}
}

func TestSearchMarkets(t *testing.T) {
	idx := newMockVectorIndex()
	idx.results = []vector.SearchResult{
		{
			Node: vector.Node{
				ID: "market1",
				Metadata: map[string]string{
					"question":    "Will Bitcoin reach $100k?",
					"description": "Test",
					"outcomes":    `["Yes","No"]`,
					"liquidity":   "50000",
					"end_date":    "2025-12-31",
				},
			},
			Score: 0.95,
		},
	}

	store, _ := NewStore(StoreConfig{
		VectorIndex: idx,
		Embedder:    &mockEmbedder{embedding: []float32{0.1, 0.2}},
	})

	results, err := store.SearchMarkets(context.Background(), "bitcoin price", 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("results count = %d, want 1", len(results))
	}

	if results[0].ID != "market1" {
		t.Errorf("result ID = %q, want %q", results[0].ID, "market1")
	}
	if results[0].Score != 0.95 {
		t.Errorf("result score = %f, want 0.95", results[0].Score)
	}
	if results[0].Question != "Will Bitcoin reach $100k?" {
		t.Errorf("result question = %q", results[0].Question)
	}
}

func TestSearchMarketsEmbedError(t *testing.T) {
	store, _ := NewStore(StoreConfig{
		VectorIndex: newMockVectorIndex(),
		Embedder:    &mockEmbedder{err: errors.New("embed failed")},
	})

	_, err := store.SearchMarkets(context.Background(), "query", 10)
	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestIndexEventsEmpty(t *testing.T) {
	idx := newMockVectorIndex()
	store, _ := NewStore(StoreConfig{
		VectorIndex: idx,
		Embedder:    &mockEmbedder{embedding: []float32{0.1}},
	})

	err := store.IndexEvents(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if idx.upsertCalls != 0 {
		t.Errorf("upsertCalls = %d, want 0", idx.upsertCalls)
	}
}

func TestIndexEvents(t *testing.T) {
	idx := newMockVectorIndex()
	store, _ := NewStore(StoreConfig{
		VectorIndex: idx,
		Embedder:    &mockEmbedder{embedding: []float32{0.1}},
	})

	events := []EventForIndex{
		{
			ID:          "event1",
			Title:       "2024 US Election",
			Description: "Presidential election",
			Slug:        "2024-election",
			Liquidity:   1000000,
			Volume:      500000,
		},
	}

	err := store.IndexEvents(context.Background(), events)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if idx.upsertCalls != 1 {
		t.Errorf("upsertCalls = %d, want 1", idx.upsertCalls)
	}

	node, ok := idx.nodes["event1"]
	if !ok {
		t.Fatal("event1 not found")
	}
	if node.Metadata["type"] != "event" {
		t.Errorf("metadata type = %q, want %q", node.Metadata["type"], "event")
	}
}

func TestSearchEvents(t *testing.T) {
	idx := newMockVectorIndex()
	idx.results = []vector.SearchResult{
		{
			Node: vector.Node{
				ID: "event1",
				Metadata: map[string]string{
					"title":       "2024 US Election",
					"description": "Presidential",
					"slug":        "2024-election",
					"liquidity":   "1000000",
					"volume":      "500000",
				},
			},
			Score: 0.9,
		},
	}

	store, _ := NewStore(StoreConfig{
		VectorIndex: idx,
		Embedder:    &mockEmbedder{embedding: []float32{0.1}},
	})

	results, err := store.SearchEvents(context.Background(), "election", 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("results count = %d, want 1", len(results))
	}
	if results[0].Title != "2024 US Election" {
		t.Errorf("title = %q", results[0].Title)
	}
}

func TestRetrieve(t *testing.T) {
	idx := newMockVectorIndex()
	idx.results = []vector.SearchResult{
		{
			Node: vector.Node{
				ID:      "node1",
				Content: "Test content",
				Metadata: map[string]string{
					"key": "value",
				},
			},
			Score: 0.85,
		},
	}

	store, _ := NewStore(StoreConfig{
		VectorIndex: idx,
		Embedder:    &mockEmbedder{embedding: []float32{0.1, 0.2}},
	})

	result, err := store.Retrieve(context.Background(), retrieve.Query{
		Text: "test query",
		TopK: 10,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Items) != 1 {
		t.Errorf("items count = %d, want 1", len(result.Items))
	}
	if result.Items[0].ID != "node1" {
		t.Errorf("item ID = %q, want %q", result.Items[0].ID, "node1")
	}
	if result.Items[0].Score != 0.85 {
		t.Errorf("item score = %f, want 0.85", result.Items[0].Score)
	}
}

func TestRetrieveWithPrecomputedEmbedding(t *testing.T) {
	idx := newMockVectorIndex()
	idx.results = []vector.SearchResult{}

	// Use an embedder that would fail if called
	store, _ := NewStore(StoreConfig{
		VectorIndex: idx,
		Embedder:    &mockEmbedder{err: errors.New("should not be called")},
	})

	// Provide pre-computed embedding
	_, err := store.Retrieve(context.Background(), retrieve.Query{
		Embedding: []float32{0.1, 0.2, 0.3},
		TopK:      5,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFormatMarketForEmbedding(t *testing.T) {
	market := polymarket.Market{
		Question:    "Will Bitcoin reach $100k?",
		Description: "This market resolves YES if...",
		Outcomes:    `["Yes","No"]`,
	}

	text := formatMarketForEmbedding(market)

	if !containsString(text, "Will Bitcoin reach $100k?") {
		t.Error("should contain question")
	}
	if !containsString(text, "This market resolves YES if...") {
		t.Error("should contain description")
	}
	if !containsString(text, "Outcomes:") {
		t.Error("should contain outcomes label")
	}
}

func TestFormatMarketForEmbeddingNoOutcomes(t *testing.T) {
	market := polymarket.Market{
		Question:    "Test?",
		Description: "Desc",
	}

	text := formatMarketForEmbedding(market)

	if containsString(text, "Outcomes:") {
		t.Error("should not contain outcomes label when empty")
	}
}

func TestFormatEventForEmbedding(t *testing.T) {
	event := EventForIndex{
		Title:       "2024 US Election",
		Description: "Presidential election prediction",
	}

	text := formatEventForEmbedding(event)

	if !containsString(text, "2024 US Election") {
		t.Error("should contain title")
	}
	if !containsString(text, "Presidential election prediction") {
		t.Error("should contain description")
	}
}

func TestTruncateString(t *testing.T) {
	tests := []struct {
		input  string
		maxLen int
		want   string
	}{
		{"hello", 10, "hello"},
		{"hello", 5, "hello"},
		{"hello world", 8, "hello..."},
		{"hello world", 3, "..."},
		{"", 10, ""},
	}

	for _, tt := range tests {
		got := truncateString(tt.input, tt.maxLen)
		if got != tt.want {
			t.Errorf("truncateString(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.want)
		}
	}
}

func TestMarketSearchResultStructure(t *testing.T) {
	result := MarketSearchResult{
		ID:          "test-id",
		Score:       0.95,
		Question:    "Test question?",
		Description: "Description",
		Outcomes:    `["Yes","No"]`,
		Liquidity:   "50000",
		EndDate:     "2025-12-31",
	}

	if result.ID != "test-id" {
		t.Errorf("ID = %q, want %q", result.ID, "test-id")
	}
	if result.Score != 0.95 {
		t.Errorf("Score = %f, want 0.95", result.Score)
	}
}

func TestEventSearchResultStructure(t *testing.T) {
	result := EventSearchResult{
		ID:          "event-id",
		Score:       0.8,
		Title:       "Event Title",
		Description: "Event description",
		Slug:        "event-slug",
		Liquidity:   "100000",
		Volume:      "50000",
	}

	if result.ID != "event-id" {
		t.Errorf("ID = %q, want %q", result.ID, "event-id")
	}
	if result.Title != "Event Title" {
		t.Errorf("Title = %q, want %q", result.Title, "Event Title")
	}
}

func TestEventForIndexStructure(t *testing.T) {
	event := EventForIndex{
		ID:          "e1",
		Title:       "Title",
		Description: "Desc",
		Slug:        "slug",
		Liquidity:   1000,
		Volume:      500,
	}

	if event.ID != "e1" {
		t.Errorf("ID = %q, want %q", event.ID, "e1")
	}
	if event.Liquidity != 1000 {
		t.Errorf("Liquidity = %f, want 1000", event.Liquidity)
	}
}

// Helper function
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
