package rag

import (
	"context"
	"errors"
	"testing"

	"github.com/plexusone/omniretrieve/graph"
)

// mockKnowledgeGraph implements graph.KnowledgeGraph for testing.
type mockKnowledgeGraph struct {
	nodes   map[string]graph.Node
	edges   []graph.Edge
	err     error
	results *graph.TraversalResult
}

func newMockKnowledgeGraph() *mockKnowledgeGraph {
	return &mockKnowledgeGraph{
		nodes: make(map[string]graph.Node),
	}
}

func (m *mockKnowledgeGraph) AddNode(ctx context.Context, node graph.Node) error {
	if m.err != nil {
		return m.err
	}
	m.nodes[node.ID] = node
	return nil
}

func (m *mockKnowledgeGraph) UpsertNode(ctx context.Context, node graph.Node) error {
	if m.err != nil {
		return m.err
	}
	m.nodes[node.ID] = node
	return nil
}

func (m *mockKnowledgeGraph) AddEdge(ctx context.Context, edge graph.Edge) error {
	if m.err != nil {
		return m.err
	}
	m.edges = append(m.edges, edge)
	return nil
}

func (m *mockKnowledgeGraph) UpsertEdge(ctx context.Context, edge graph.Edge) error {
	if m.err != nil {
		return m.err
	}
	m.edges = append(m.edges, edge)
	return nil
}

func (m *mockKnowledgeGraph) FindNodes(ctx context.Context, nodeType string, filters map[string]string) ([]graph.Node, error) {
	if m.err != nil {
		return nil, m.err
	}
	var result []graph.Node
	for _, n := range m.nodes {
		if n.Type == nodeType {
			result = append(result, n)
		}
	}
	return result, nil
}

func (m *mockKnowledgeGraph) DeleteNode(ctx context.Context, id string) error {
	if m.err != nil {
		return m.err
	}
	delete(m.nodes, id)
	return nil
}

func (m *mockKnowledgeGraph) DeleteEdge(ctx context.Context, from, to, edgeType string) error {
	if m.err != nil {
		return m.err
	}
	return nil
}

func (m *mockKnowledgeGraph) Traverse(ctx context.Context, startIDs []string, opts graph.TraversalOptions) (*graph.TraversalResult, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.results != nil {
		return m.results, nil
	}
	return &graph.TraversalResult{
		Nodes: []graph.Node{},
		Edges: []graph.Edge{},
		Paths: make(map[string][]string),
	}, nil
}

func (m *mockKnowledgeGraph) Name() string {
	return "mock-graph"
}

func TestNodeTypeConstants(t *testing.T) {
	if NodeTypeEvent != "polymarket_event" {
		t.Errorf("NodeTypeEvent = %q, want polymarket_event", NodeTypeEvent)
	}
	if NodeTypeMarket != "polymarket_market" {
		t.Errorf("NodeTypeMarket = %q, want polymarket_market", NodeTypeMarket)
	}
	if NodeTypeCategory != "market_category" {
		t.Errorf("NodeTypeCategory = %q, want market_category", NodeTypeCategory)
	}
	if NodeTypeTopic != "event_topic" {
		t.Errorf("NodeTypeTopic = %q, want event_topic", NodeTypeTopic)
	}
}

func TestEdgeTypeConstants(t *testing.T) {
	expectedTypes := map[string]string{
		"EdgeTypeMentionedIn":     EdgeTypeMentionedIn,
		"EdgeTypeHasMarket":       EdgeTypeHasMarket,
		"EdgeTypeCategoryIs":      EdgeTypeCategoryIs,
		"EdgeTypeTopicRelatesTo":  EdgeTypeTopicRelatesTo,
		"EdgeTypeCorrelatedWith":  EdgeTypeCorrelatedWith,
		"EdgeTypeSemanticSimilar": EdgeTypeSemanticSimilar,
		"EdgeTypeSameEvent":       EdgeTypeSameEvent,
	}

	for name, value := range expectedTypes {
		if value == "" {
			t.Errorf("%s should not be empty", name)
		}
	}
}

func TestNewGraphStore(t *testing.T) {
	tests := []struct {
		name    string
		cfg     GraphStoreConfig
		wantErr bool
	}{
		{
			name:    "missing graph",
			cfg:     GraphStoreConfig{},
			wantErr: true,
		},
		{
			name: "valid config",
			cfg: GraphStoreConfig{
				Graph: newMockKnowledgeGraph(),
			},
			wantErr: false,
		},
		{
			name: "custom thresholds",
			cfg: GraphStoreConfig{
				Graph:                newMockKnowledgeGraph(),
				SimilarityThreshold:  0.9,
				CorrelationThreshold: 0.8,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store, err := NewGraphStore(tt.cfg)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
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

func TestNewGraphStoreDefaults(t *testing.T) {
	store, err := NewGraphStore(GraphStoreConfig{
		Graph: newMockKnowledgeGraph(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if store.config.SimilarityThreshold != 0.8 {
		t.Errorf("default SimilarityThreshold = %f, want 0.8", store.config.SimilarityThreshold)
	}
	if store.config.CorrelationThreshold != 0.7 {
		t.Errorf("default CorrelationThreshold = %f, want 0.7", store.config.CorrelationThreshold)
	}
}

func TestIndexEvent(t *testing.T) {
	mockGraph := newMockKnowledgeGraph()
	store, _ := NewGraphStore(GraphStoreConfig{Graph: mockGraph})

	event := EventForGraph{
		ID:          "event1",
		Title:       "2024 Election",
		Description: "Presidential election",
		Slug:        "2024-election",
		Tags:        []string{"politics", "election"},
		MarketIDs:   []string{"m1", "m2"},
		Liquidity:   1000000,
		Volume:      500000,
	}

	err := store.IndexEvent(context.Background(), event)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check event node was created
	if _, ok := mockGraph.nodes["event1"]; !ok {
		t.Error("event node should be created")
	}

	// Check topic nodes were created
	if _, ok := mockGraph.nodes["topic_politics"]; !ok {
		t.Error("topic_politics node should be created")
	}
	if _, ok := mockGraph.nodes["topic_election"]; !ok {
		t.Error("topic_election node should be created")
	}

	// Check edges were created (2 topic->event edges)
	if len(mockGraph.edges) != 2 {
		t.Errorf("edges count = %d, want 2", len(mockGraph.edges))
	}
}

func TestIndexEventError(t *testing.T) {
	mockGraph := newMockKnowledgeGraph()
	mockGraph.err = errors.New("database error")
	store, _ := NewGraphStore(GraphStoreConfig{Graph: mockGraph})

	event := EventForGraph{ID: "e1", Title: "Test"}
	err := store.IndexEvent(context.Background(), event)

	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestIndexMarket(t *testing.T) {
	mockGraph := newMockKnowledgeGraph()
	store, _ := NewGraphStore(GraphStoreConfig{Graph: mockGraph})

	market := MarketForGraph{
		ID:          "market1",
		EventID:     "event1",
		Question:    "Will X happen?",
		Description: "Description",
		Outcomes:    `["Yes","No"]`,
		Category:    "politics",
		Liquidity:   50000,
		Volume:      10000,
		OutcomeYes:  0.65,
		OutcomeNo:   0.35,
	}

	err := store.IndexMarket(context.Background(), market)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check market node
	if _, ok := mockGraph.nodes["market1"]; !ok {
		t.Error("market node should be created")
	}

	// Check category node
	if _, ok := mockGraph.nodes["category_politics"]; !ok {
		t.Error("category_politics node should be created")
	}

	// Check edges (event->market + market->category)
	if len(mockGraph.edges) != 2 {
		t.Errorf("edges count = %d, want 2", len(mockGraph.edges))
	}
}

func TestIndexMarketNoEventID(t *testing.T) {
	mockGraph := newMockKnowledgeGraph()
	store, _ := NewGraphStore(GraphStoreConfig{Graph: mockGraph})

	market := MarketForGraph{
		ID:       "market1",
		Question: "Test?",
		Category: "sports",
	}

	err := store.IndexMarket(context.Background(), market)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should only have 1 edge (market->category), no event->market edge
	if len(mockGraph.edges) != 1 {
		t.Errorf("edges count = %d, want 1", len(mockGraph.edges))
	}
}

func TestIndexMarketNoCategory(t *testing.T) {
	mockGraph := newMockKnowledgeGraph()
	store, _ := NewGraphStore(GraphStoreConfig{Graph: mockGraph})

	market := MarketForGraph{
		ID:       "market1",
		EventID:  "event1",
		Question: "Test?",
	}

	err := store.IndexMarket(context.Background(), market)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should only have event->market edge
	if len(mockGraph.edges) != 1 {
		t.Errorf("edges count = %d, want 1", len(mockGraph.edges))
	}
}

func TestLinkSiblingMarkets(t *testing.T) {
	mockGraph := newMockKnowledgeGraph()
	store, _ := NewGraphStore(GraphStoreConfig{Graph: mockGraph})

	err := store.LinkSiblingMarkets(context.Background(), "event1", []string{"m1", "m2", "m3"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should create bidirectional edges: (m1-m2, m2-m1), (m1-m3, m3-m1), (m2-m3, m3-m2) = 6 edges
	if len(mockGraph.edges) != 6 {
		t.Errorf("edges count = %d, want 6", len(mockGraph.edges))
	}
}

func TestLinkSiblingMarketsEmpty(t *testing.T) {
	mockGraph := newMockKnowledgeGraph()
	store, _ := NewGraphStore(GraphStoreConfig{Graph: mockGraph})

	err := store.LinkSiblingMarkets(context.Background(), "event1", []string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(mockGraph.edges) != 0 {
		t.Errorf("edges count = %d, want 0", len(mockGraph.edges))
	}
}

func TestLinkSiblingMarketsSingle(t *testing.T) {
	mockGraph := newMockKnowledgeGraph()
	store, _ := NewGraphStore(GraphStoreConfig{Graph: mockGraph})

	err := store.LinkSiblingMarkets(context.Background(), "event1", []string{"m1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Single market = no pairs
	if len(mockGraph.edges) != 0 {
		t.Errorf("edges count = %d, want 0", len(mockGraph.edges))
	}
}

func TestAddCorrelation(t *testing.T) {
	mockGraph := newMockKnowledgeGraph()
	store, _ := NewGraphStore(GraphStoreConfig{
		Graph:                mockGraph,
		CorrelationThreshold: 0.7,
	})

	// Above threshold
	err := store.AddCorrelation(context.Background(), "m1", "m2", 0.8)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(mockGraph.edges) != 2 { // bidirectional
		t.Errorf("edges count = %d, want 2", len(mockGraph.edges))
	}
}

func TestAddCorrelationBelowThreshold(t *testing.T) {
	mockGraph := newMockKnowledgeGraph()
	store, _ := NewGraphStore(GraphStoreConfig{
		Graph:                mockGraph,
		CorrelationThreshold: 0.7,
	})

	// Below threshold - should not add edge
	err := store.AddCorrelation(context.Background(), "m1", "m2", 0.5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(mockGraph.edges) != 0 {
		t.Errorf("edges count = %d, want 0 (below threshold)", len(mockGraph.edges))
	}
}

func TestFindRelatedMarkets(t *testing.T) {
	mockGraph := newMockKnowledgeGraph()
	mockGraph.results = &graph.TraversalResult{
		Nodes: []graph.Node{
			{ID: "m1", Type: NodeTypeMarket, Content: "Market 1"},
			{ID: "m2", Type: NodeTypeMarket, Content: "Market 2"},
		},
		Edges: []graph.Edge{
			{From: "m1", To: "m2", Type: EdgeTypeCorrelatedWith, Weight: 0.8},
		},
		Paths: map[string][]string{
			"m1": {},
			"m2": {"m1", EdgeTypeCorrelatedWith, "m2"},
		},
	}
	store, _ := NewGraphStore(GraphStoreConfig{Graph: mockGraph})

	results, err := store.FindRelatedMarkets(context.Background(), "m1", 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should exclude the starting node m1
	if len(results) != 1 {
		t.Errorf("results count = %d, want 1", len(results))
	}
	if len(results) > 0 && results[0].ID != "m2" {
		t.Errorf("result ID = %q, want m2", results[0].ID)
	}
}

func TestFindRelatedMarketsDefaultDepth(t *testing.T) {
	mockGraph := newMockKnowledgeGraph()
	mockGraph.results = &graph.TraversalResult{
		Nodes: []graph.Node{},
		Edges: []graph.Edge{},
		Paths: make(map[string][]string),
	}
	store, _ := NewGraphStore(GraphStoreConfig{Graph: mockGraph})

	// Depth 0 or negative should default to 2
	_, err := store.FindRelatedMarkets(context.Background(), "m1", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFindMarketsForEvent(t *testing.T) {
	mockGraph := newMockKnowledgeGraph()
	mockGraph.results = &graph.TraversalResult{
		Nodes: []graph.Node{
			{ID: "event1", Type: NodeTypeEvent},
			{ID: "m1", Type: NodeTypeMarket, Content: "Market 1"},
			{ID: "m2", Type: NodeTypeMarket, Content: "Market 2"},
		},
		Edges: []graph.Edge{},
		Paths: map[string][]string{
			"m1": {"event1", EdgeTypeHasMarket, "m1"},
			"m2": {"event1", EdgeTypeHasMarket, "m2"},
		},
	}
	store, _ := NewGraphStore(GraphStoreConfig{Graph: mockGraph})

	results, err := store.FindMarketsForEvent(context.Background(), "event1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should only return markets, not the event
	if len(results) != 2 {
		t.Errorf("results count = %d, want 2", len(results))
	}
}

func TestFindEventsByTopic(t *testing.T) {
	mockGraph := newMockKnowledgeGraph()
	mockGraph.results = &graph.TraversalResult{
		Nodes: []graph.Node{
			{ID: "topic_politics", Type: NodeTypeTopic},
			{ID: "e1", Type: NodeTypeEvent, Content: "Election 2024"},
		},
		Edges: []graph.Edge{},
		Paths: map[string][]string{
			"e1": {"topic_politics", EdgeTypeTopicRelatesTo, "e1"},
		},
	}
	store, _ := NewGraphStore(GraphStoreConfig{Graph: mockGraph})

	results, err := store.FindEventsByTopic(context.Background(), "politics")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should only return events
	if len(results) != 1 {
		t.Errorf("results count = %d, want 1", len(results))
	}
}

func TestHybridSearchMissingDependencies(t *testing.T) {
	mockGraph := newMockKnowledgeGraph()
	store, _ := NewGraphStore(GraphStoreConfig{Graph: mockGraph})

	_, err := store.HybridSearch(context.Background(), "query", HybridSearchOptions{})
	if err == nil {
		t.Error("expected error for missing vector index and embedder")
	}
}

func TestSlugify(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"politics", "politics"},
		{"US Politics", "us_politics"},
		{"crypto-currency", "crypto_currency"},
		{"Space Exploration", "space_exploration"},
		{"", ""},
	}

	for _, tt := range tests {
		got := slugify(tt.input)
		if got != tt.want {
			t.Errorf("slugify(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestComputeGraphScore(t *testing.T) {
	tests := []struct {
		name  string
		path  []string
		edges []graph.Edge
		want  float64
	}{
		{
			name:  "empty path",
			path:  []string{},
			edges: []graph.Edge{},
			want:  1.0,
		},
		{
			name:  "single hop with weight",
			path:  []string{"a", "b"},
			edges: []graph.Edge{{From: "a", To: "b", Weight: 1.0}},
			want:  0.8, // 1.0 * 1.0 * 0.8
		},
		{
			name:  "single hop no weight",
			path:  []string{"a", "b"},
			edges: []graph.Edge{}, // No matching edge
			want:  0.4,            // 1.0 * 0.5 * 0.8
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := computeGraphScore(tt.path, tt.edges)
			if got != tt.want {
				t.Errorf("computeGraphScore() = %f, want %f", got, tt.want)
			}
		})
	}
}

func TestExtractEdgeTypes(t *testing.T) {
	tests := []struct {
		path []string
		want []string
	}{
		{[]string{}, nil},
		{[]string{"a"}, nil},
		{[]string{"a", "edge1", "b"}, []string{"edge1"}},
		{[]string{"a", "e1", "b", "e2", "c"}, []string{"e1", "e2"}},
	}

	for _, tt := range tests {
		got := extractEdgeTypes(tt.path)
		if len(got) != len(tt.want) {
			t.Errorf("extractEdgeTypes(%v) = %v, want %v", tt.path, got, tt.want)
		}
	}
}

func TestGraphSearchResultStructure(t *testing.T) {
	r := GraphSearchResult{
		ID:        "id",
		Type:      "market",
		Content:   "content",
		Score:     0.9,
		Path:      []string{"a", "b"},
		EdgeTypes: []string{"e1"},
		Metadata:  map[string]string{"key": "value"},
	}

	if r.ID != "id" {
		t.Errorf("ID = %q, want id", r.ID)
	}
	if r.Score != 0.9 {
		t.Errorf("Score = %f, want 0.9", r.Score)
	}
}

func TestHybridSearchOptionsDefaults(t *testing.T) {
	opts := HybridSearchOptions{}

	if opts.TopK != 0 {
		t.Errorf("default TopK = %d, want 0", opts.TopK)
	}
	if opts.VectorWeight != 0 {
		t.Errorf("default VectorWeight = %f, want 0", opts.VectorWeight)
	}
}

func TestEntityHintStructure(t *testing.T) {
	hint := EntityHint{
		ID:   "entity1",
		Type: "market",
	}

	if hint.ID != "entity1" {
		t.Errorf("ID = %q, want entity1", hint.ID)
	}
	if hint.Type != "market" {
		t.Errorf("Type = %q, want market", hint.Type)
	}
}

func TestEventForGraphStructure(t *testing.T) {
	e := EventForGraph{
		ID:          "e1",
		Title:       "Title",
		Description: "Desc",
		Slug:        "slug",
		Tags:        []string{"t1"},
		MarketIDs:   []string{"m1"},
		Liquidity:   1000,
		Volume:      500,
	}

	if e.ID != "e1" {
		t.Errorf("ID = %q, want e1", e.ID)
	}
}

func TestMarketForGraphStructure(t *testing.T) {
	m := MarketForGraph{
		ID:          "m1",
		EventID:     "e1",
		Question:    "Q",
		Description: "D",
		Outcomes:    "O",
		Category:    "C",
		Tags:        []string{"t1"},
		Liquidity:   1000,
		Volume:      500,
		OutcomeYes:  0.6,
		OutcomeNo:   0.4,
	}

	if m.ID != "m1" {
		t.Errorf("ID = %q, want m1", m.ID)
	}
	if m.OutcomeYes != 0.6 {
		t.Errorf("OutcomeYes = %f, want 0.6", m.OutcomeYes)
	}
}
