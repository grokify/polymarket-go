// Package rag provides RAG capabilities including GraphRAG for Polymarket.
package rag

import (
	"context"
	"fmt"
	"strings"

	"github.com/plexusone/omniretrieve/graph"
	"github.com/plexusone/omniretrieve/hybrid"
	"github.com/plexusone/omniretrieve/retrieve"
	"github.com/plexusone/omniretrieve/vector"
)

// Node types for Polymarket graph.
const (
	NodeTypeEvent    = "polymarket_event"
	NodeTypeMarket   = "polymarket_market"
	NodeTypeCategory = "market_category"
	NodeTypeTopic    = "event_topic"
)

// Edge types for Polymarket graph.
const (
	EdgeTypeMentionedIn     = "mentioned_in"     // event → market
	EdgeTypeHasMarket       = "has_market"       // event → market (direct)
	EdgeTypeCategoryIs      = "category_is"      // market → category
	EdgeTypeTopicRelatesTo  = "topic_relates_to" // topic → event
	EdgeTypeCorrelatedWith  = "correlated_with"  // market → market
	EdgeTypeSemanticSimilar = "semantic_similar" // market → market
	EdgeTypeSameEvent       = "same_event"       // market → market (siblings)
)

// GraphStore provides GraphRAG capabilities for Polymarket.
type GraphStore struct {
	graph       graph.KnowledgeGraph
	vectorIndex vector.Index
	embedder    Embedder
	config      GraphStoreConfig
}

// GraphStoreConfig configures the GraphRAG store.
type GraphStoreConfig struct {
	// Graph is the knowledge graph backend.
	Graph graph.KnowledgeGraph

	// VectorIndex is the vector index for hybrid retrieval.
	VectorIndex vector.Index

	// Embedder generates embeddings.
	Embedder Embedder

	// SimilarityThreshold for creating semantic_similar edges (default: 0.8).
	SimilarityThreshold float64

	// CorrelationThreshold for creating correlated_with edges (default: 0.7).
	CorrelationThreshold float64
}

// NewGraphStore creates a new GraphRAG store.
func NewGraphStore(cfg GraphStoreConfig) (*GraphStore, error) {
	if cfg.Graph == nil {
		return nil, fmt.Errorf("graph is required")
	}
	if cfg.SimilarityThreshold == 0 {
		cfg.SimilarityThreshold = 0.8
	}
	if cfg.CorrelationThreshold == 0 {
		cfg.CorrelationThreshold = 0.7
	}

	return &GraphStore{
		graph:       cfg.Graph,
		vectorIndex: cfg.VectorIndex,
		embedder:    cfg.Embedder,
		config:      cfg,
	}, nil
}

// EventForGraph holds event data for graph indexing.
type EventForGraph struct {
	ID          string
	Title       string
	Description string
	Slug        string
	Tags        []string
	MarketIDs   []string
	Liquidity   float64
	Volume      float64
}

// MarketForGraph holds market data for graph indexing.
type MarketForGraph struct {
	ID          string
	EventID     string
	Question    string
	Description string
	Outcomes    string
	Category    string
	Tags        []string
	Liquidity   float64
	Volume      float64
	OutcomeYes  float64
	OutcomeNo   float64
}

// IndexEvent adds an event and its relationships to the graph.
func (gs *GraphStore) IndexEvent(ctx context.Context, event EventForGraph) error {
	// Create event node
	eventNode := graph.Node{
		ID:      event.ID,
		Type:    NodeTypeEvent,
		Content: fmt.Sprintf("%s\n%s", event.Title, event.Description),
		Source:  "polymarket",
		Metadata: map[string]string{
			"title":      event.Title,
			"slug":       event.Slug,
			"liquidity":  fmt.Sprintf("%.2f", event.Liquidity),
			"volume":     fmt.Sprintf("%.2f", event.Volume),
			"market_ids": strings.Join(event.MarketIDs, ","),
		},
	}

	if err := gs.graph.UpsertNode(ctx, eventNode); err != nil {
		return fmt.Errorf("upserting event node: %w", err)
	}

	// Create topic nodes and edges for tags
	for _, tag := range event.Tags {
		topicID := "topic_" + slugify(tag)
		topicNode := graph.Node{
			ID:      topicID,
			Type:    NodeTypeTopic,
			Content: tag,
			Source:  "polymarket",
			Metadata: map[string]string{
				"name": tag,
			},
		}

		if err := gs.graph.UpsertNode(ctx, topicNode); err != nil {
			return fmt.Errorf("upserting topic node: %w", err)
		}

		// Topic relates to event
		edge := graph.Edge{
			From:   topicID,
			To:     event.ID,
			Type:   EdgeTypeTopicRelatesTo,
			Weight: 1.0,
		}

		if err := gs.graph.UpsertEdge(ctx, edge); err != nil {
			return fmt.Errorf("upserting topic edge: %w", err)
		}
	}

	return nil
}

// IndexMarket adds a market and its relationships to the graph.
func (gs *GraphStore) IndexMarket(ctx context.Context, market MarketForGraph) error {
	// Create market node
	marketNode := graph.Node{
		ID:      market.ID,
		Type:    NodeTypeMarket,
		Content: fmt.Sprintf("%s\n%s\nOutcomes: %s", market.Question, market.Description, market.Outcomes),
		Source:  "polymarket",
		Metadata: map[string]string{
			"question":    market.Question,
			"description": truncateString(market.Description, 500),
			"outcomes":    market.Outcomes,
			"category":    market.Category,
			"event_id":    market.EventID,
			"liquidity":   fmt.Sprintf("%.2f", market.Liquidity),
			"volume":      fmt.Sprintf("%.2f", market.Volume),
			"outcome_yes": fmt.Sprintf("%.4f", market.OutcomeYes),
			"outcome_no":  fmt.Sprintf("%.4f", market.OutcomeNo),
		},
	}

	if err := gs.graph.UpsertNode(ctx, marketNode); err != nil {
		return fmt.Errorf("upserting market node: %w", err)
	}

	// Link market to event
	if market.EventID != "" {
		edge := graph.Edge{
			From:   market.EventID,
			To:     market.ID,
			Type:   EdgeTypeHasMarket,
			Weight: 1.0,
		}

		if err := gs.graph.UpsertEdge(ctx, edge); err != nil {
			return fmt.Errorf("upserting event-market edge: %w", err)
		}
	}

	// Link market to category
	if market.Category != "" {
		categoryID := "category_" + slugify(market.Category)
		categoryNode := graph.Node{
			ID:      categoryID,
			Type:    NodeTypeCategory,
			Content: market.Category,
			Source:  "polymarket",
			Metadata: map[string]string{
				"name": market.Category,
			},
		}

		if err := gs.graph.UpsertNode(ctx, categoryNode); err != nil {
			return fmt.Errorf("upserting category node: %w", err)
		}

		edge := graph.Edge{
			From:   market.ID,
			To:     categoryID,
			Type:   EdgeTypeCategoryIs,
			Weight: 1.0,
		}

		if err := gs.graph.UpsertEdge(ctx, edge); err != nil {
			return fmt.Errorf("upserting category edge: %w", err)
		}
	}

	return nil
}

// LinkSiblingMarkets creates same_event edges between markets under the same event.
func (gs *GraphStore) LinkSiblingMarkets(ctx context.Context, eventID string, marketIDs []string) error {
	// Create edges between all pairs of markets under the same event
	for i := 0; i < len(marketIDs); i++ {
		for j := i + 1; j < len(marketIDs); j++ {
			edge := graph.Edge{
				From:   marketIDs[i],
				To:     marketIDs[j],
				Type:   EdgeTypeSameEvent,
				Weight: 0.9,
				Metadata: map[string]string{
					"event_id": eventID,
				},
			}

			if err := gs.graph.UpsertEdge(ctx, edge); err != nil {
				return fmt.Errorf("upserting sibling edge: %w", err)
			}

			// Bidirectional
			edge.From, edge.To = edge.To, edge.From
			if err := gs.graph.UpsertEdge(ctx, edge); err != nil {
				return fmt.Errorf("upserting sibling edge (reverse): %w", err)
			}
		}
	}

	return nil
}

// AddCorrelation adds a correlation edge between two markets.
func (gs *GraphStore) AddCorrelation(ctx context.Context, marketID1, marketID2 string, correlation float64) error {
	if correlation < gs.config.CorrelationThreshold {
		return nil // Below threshold, don't add edge
	}

	edge := graph.Edge{
		From:   marketID1,
		To:     marketID2,
		Type:   EdgeTypeCorrelatedWith,
		Weight: correlation,
		Metadata: map[string]string{
			"correlation": fmt.Sprintf("%.4f", correlation),
		},
	}

	if err := gs.graph.UpsertEdge(ctx, edge); err != nil {
		return fmt.Errorf("upserting correlation edge: %w", err)
	}

	// Bidirectional
	edge.From, edge.To = edge.To, edge.From
	if err := gs.graph.UpsertEdge(ctx, edge); err != nil {
		return fmt.Errorf("upserting correlation edge (reverse): %w", err)
	}

	return nil
}

// FindRelatedMarkets finds markets related to a given market via graph traversal.
func (gs *GraphStore) FindRelatedMarkets(ctx context.Context, marketID string, depth int) ([]GraphSearchResult, error) {
	if depth <= 0 {
		depth = 2
	}

	result, err := gs.graph.Traverse(ctx, []string{marketID}, graph.TraversalOptions{
		Depth:     depth,
		NodeTypes: []string{NodeTypeMarket},
		EdgeTypes: []string{EdgeTypeCorrelatedWith, EdgeTypeSameEvent, EdgeTypeSemanticSimilar},
	})
	if err != nil {
		return nil, fmt.Errorf("traversing graph: %w", err)
	}

	searchResults := make([]GraphSearchResult, 0, len(result.Nodes))
	for _, node := range result.Nodes {
		if node.ID == marketID {
			continue // Skip the starting node
		}

		path := result.Paths[node.ID]
		score := computeGraphScore(path, result.Edges)

		searchResults = append(searchResults, GraphSearchResult{
			ID:        node.ID,
			Type:      node.Type,
			Content:   node.Content,
			Score:     float32(score),
			Path:      path,
			Metadata:  node.Metadata,
			EdgeTypes: extractEdgeTypes(path),
		})
	}

	return searchResults, nil
}

// FindMarketsForEvent finds all markets associated with an event.
func (gs *GraphStore) FindMarketsForEvent(ctx context.Context, eventID string) ([]GraphSearchResult, error) {
	result, err := gs.graph.Traverse(ctx, []string{eventID}, graph.TraversalOptions{
		Depth:     1,
		NodeTypes: []string{NodeTypeMarket},
		EdgeTypes: []string{EdgeTypeHasMarket},
	})
	if err != nil {
		return nil, fmt.Errorf("traversing graph: %w", err)
	}

	searchResults := make([]GraphSearchResult, 0, len(result.Nodes))
	for _, node := range result.Nodes {
		if node.Type != NodeTypeMarket {
			continue
		}

		path := result.Paths[node.ID]
		score := computeGraphScore(path, result.Edges)

		searchResults = append(searchResults, GraphSearchResult{
			ID:       node.ID,
			Type:     node.Type,
			Content:  node.Content,
			Score:    float32(score),
			Path:     path,
			Metadata: node.Metadata,
		})
	}

	return searchResults, nil
}

// FindEventsByTopic finds events related to a topic.
func (gs *GraphStore) FindEventsByTopic(ctx context.Context, topic string) ([]GraphSearchResult, error) {
	topicID := "topic_" + slugify(topic)

	result, err := gs.graph.Traverse(ctx, []string{topicID}, graph.TraversalOptions{
		Depth:     1,
		NodeTypes: []string{NodeTypeEvent},
		EdgeTypes: []string{EdgeTypeTopicRelatesTo},
	})
	if err != nil {
		return nil, fmt.Errorf("traversing graph: %w", err)
	}

	searchResults := make([]GraphSearchResult, 0, len(result.Nodes))
	for _, node := range result.Nodes {
		if node.Type != NodeTypeEvent {
			continue
		}

		path := result.Paths[node.ID]
		score := computeGraphScore(path, result.Edges)

		searchResults = append(searchResults, GraphSearchResult{
			ID:       node.ID,
			Type:     node.Type,
			Content:  node.Content,
			Score:    float32(score),
			Path:     path,
			Metadata: node.Metadata,
		})
	}

	return searchResults, nil
}

// HybridSearch performs combined vector + graph search.
func (gs *GraphStore) HybridSearch(ctx context.Context, query string, opts HybridSearchOptions) ([]GraphSearchResult, error) {
	if gs.vectorIndex == nil || gs.embedder == nil {
		return nil, fmt.Errorf("vector index and embedder required for hybrid search")
	}

	if opts.TopK == 0 {
		opts.TopK = 10
	}
	if opts.VectorWeight == 0 && opts.GraphWeight == 0 {
		opts.VectorWeight = 0.6
		opts.GraphWeight = 0.4
	}
	if opts.MaxDepth == 0 {
		opts.MaxDepth = 2
	}

	// Generate query embedding
	embedding, err := gs.embedder.Embed(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("generating embedding: %w", err)
	}

	// Create vector retriever
	vectorRetriever := vector.NewRetriever(vector.RetrieverConfig{
		Index: gs.vectorIndex,
	})

	// Create graph retriever
	graphRetriever := graph.NewRetriever(graph.RetrieverConfig{
		Graph:        gs.graph,
		DefaultDepth: opts.MaxDepth,
	})

	// Create hybrid retriever
	hybridRetriever := hybrid.NewRetriever(hybrid.RetrieverConfig{
		Vector:    vectorRetriever,
		Graph:     graphRetriever,
		Policy:    hybrid.PolicyParallel,
		Weights:   hybrid.Weights{Vector: opts.VectorWeight, Graph: opts.GraphWeight},
		DedupByID: true,
	})

	// Build retrieval query
	retrieveQuery := retrieve.Query{
		Text:      query,
		Embedding: embedding,
		TopK:      opts.TopK,
		MaxDepth:  opts.MaxDepth,
		Filters:   opts.Filters,
	}

	// Add entity hints if provided
	for _, hint := range opts.EntityHints {
		retrieveQuery.Entities = append(retrieveQuery.Entities, retrieve.EntityHint{
			ID:   hint.ID,
			Type: hint.Type,
		})
	}

	// Execute hybrid retrieval
	result, err := hybridRetriever.Retrieve(ctx, retrieveQuery)
	if err != nil {
		return nil, fmt.Errorf("hybrid retrieval: %w", err)
	}

	// Convert results
	searchResults := make([]GraphSearchResult, len(result.Items))
	for i, item := range result.Items {
		searchResults[i] = GraphSearchResult{
			ID:       item.ID,
			Type:     item.Metadata["type"],
			Content:  item.Content,
			Score:    float32(item.Score),
			Metadata: item.Metadata,
			Path:     item.Provenance.GraphPath,
		}
	}

	return searchResults, nil
}

// GraphSearchResult represents a graph search result.
type GraphSearchResult struct {
	ID        string            `json:"id"`
	Type      string            `json:"type"`
	Content   string            `json:"content"`
	Score     float32           `json:"score"`
	Path      []string          `json:"path,omitempty"`
	EdgeTypes []string          `json:"edge_types,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

// HybridSearchOptions configures hybrid search.
type HybridSearchOptions struct {
	TopK         int
	VectorWeight float64
	GraphWeight  float64
	MaxDepth     int
	Filters      map[string]string
	EntityHints  []EntityHint
}

// EntityHint provides a starting point for graph traversal.
type EntityHint struct {
	ID   string
	Type string
}

// slugify converts a string to a slug format.
func slugify(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, " ", "_")
	s = strings.ReplaceAll(s, "-", "_")
	return s
}

// computeGraphScore calculates a relevance score based on path length and edge weights.
func computeGraphScore(path []string, edges []graph.Edge) float64 {
	if len(path) == 0 {
		return 1.0 // Start nodes have max score
	}

	// Build edge lookup
	edgeWeights := make(map[string]float64)
	for _, e := range edges {
		key := e.From + "->" + e.To
		edgeWeights[key] = e.Weight
	}

	// Calculate cumulative score with decay
	score := 1.0
	decayFactor := 0.8 // Score decays by 20% per hop

	for i := 0; i < len(path)-1; i++ {
		key := path[i] + "->" + path[i+1]
		weight := edgeWeights[key]
		if weight == 0 {
			weight = 0.5 // Default weight
		}
		score *= weight * decayFactor
	}

	return score
}

// extractEdgeTypes extracts edge types from a path (path contains alternating nodes and edges).
func extractEdgeTypes(path []string) []string {
	// Path format is typically: [node1, edge_type, node2, edge_type, node3, ...]
	// We extract even indices (1, 3, 5, ...) which are edge types
	var edgeTypes []string
	for i := 1; i < len(path); i += 2 {
		edgeTypes = append(edgeTypes, path[i])
	}
	return edgeTypes
}
