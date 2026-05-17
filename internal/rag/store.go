// Package rag provides RAG (Retrieval-Augmented Generation) capabilities for Polymarket.
// It uses omniretrieve for vector and graph-based retrieval with pgvector as the default store.
package rag

import (
	"context"
	"fmt"
	"strings"

	"github.com/agentplexus/omniretrieve/retrieve"
	"github.com/agentplexus/omniretrieve/vector"
	"github.com/grokify/polymarket-go/internal/polymarket"
)

// Store provides RAG capabilities for Polymarket markets and events.
type Store struct {
	vectorIndex vector.Index
	embedder    Embedder
	config      StoreConfig
}

// StoreConfig holds configuration for the RAG store.
type StoreConfig struct {
	// VectorIndex is the vector index to use (pgvector, in-memory, etc.)
	VectorIndex vector.Index

	// Embedder generates embeddings from text.
	Embedder Embedder

	// Dimensions is the embedding dimensions (default: 1536 for OpenAI).
	Dimensions int
}

// Embedder generates embeddings from text.
type Embedder interface {
	Embed(ctx context.Context, text string) ([]float32, error)
	EmbedBatch(ctx context.Context, texts []string) ([][]float32, error)
}

// NewStore creates a new RAG store.
func NewStore(cfg StoreConfig) (*Store, error) {
	if cfg.VectorIndex == nil {
		return nil, fmt.Errorf("vector index is required")
	}
	if cfg.Embedder == nil {
		return nil, fmt.Errorf("embedder is required")
	}
	if cfg.Dimensions == 0 {
		cfg.Dimensions = 1536 // OpenAI default
	}

	return &Store{
		vectorIndex: cfg.VectorIndex,
		embedder:    cfg.Embedder,
		config:      cfg,
	}, nil
}

// IndexMarkets indexes markets into the vector store.
func (s *Store) IndexMarkets(ctx context.Context, markets []polymarket.Market) error {
	if len(markets) == 0 {
		return nil
	}

	// Prepare texts for embedding
	texts := make([]string, len(markets))
	for i, m := range markets {
		texts[i] = formatMarketForEmbedding(m)
	}

	// Generate embeddings
	embeddings, err := s.embedder.EmbedBatch(ctx, texts)
	if err != nil {
		return fmt.Errorf("generating embeddings: %w", err)
	}

	// Index each market
	for i, m := range markets {
		node := vector.Node{
			ID:        m.ConditionID,
			Content:   texts[i],
			Embedding: embeddings[i],
			Source:    "polymarket",
			Metadata: map[string]string{
				"type":        "market",
				"question":    m.Question,
				"description": truncateString(m.Description, 500),
				"outcomes":    m.Outcomes,
				"liquidity":   fmt.Sprintf("%.2f", m.LiquidityNum),
				"volume_24h":  fmt.Sprintf("%.2f", m.Volume24hr),
				"end_date":    m.EndDateISO,
			},
		}

		if err := s.vectorIndex.Upsert(ctx, node); err != nil {
			return fmt.Errorf("indexing market %s: %w", m.ConditionID, err)
		}
	}

	return nil
}

// search performs a semantic search with the given filter.
func (s *Store) search(ctx context.Context, query string, topK int, filters map[string]string) ([]vector.SearchResult, error) {
	embedding, err := s.embedder.Embed(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("generating query embedding: %w", err)
	}

	results, err := s.vectorIndex.Search(ctx, embedding, topK, filters)
	if err != nil {
		return nil, fmt.Errorf("searching vector index: %w", err)
	}

	return results, nil
}

// SearchMarkets performs semantic search for markets.
func (s *Store) SearchMarkets(ctx context.Context, query string, topK int) ([]MarketSearchResult, error) {
	results, err := s.search(ctx, query, topK, map[string]string{"type": "market"})
	if err != nil {
		return nil, err
	}

	searchResults := make([]MarketSearchResult, len(results))
	for i, r := range results {
		searchResults[i] = MarketSearchResult{
			ID:          r.Node.ID,
			Score:       float32(r.Score),
			Question:    r.Node.Metadata["question"],
			Description: r.Node.Metadata["description"],
			Outcomes:    r.Node.Metadata["outcomes"],
			Liquidity:   r.Node.Metadata["liquidity"],
			EndDate:     r.Node.Metadata["end_date"],
		}
	}

	return searchResults, nil
}

// IndexEvents indexes events into the vector store.
func (s *Store) IndexEvents(ctx context.Context, events []EventForIndex) error {
	if len(events) == 0 {
		return nil
	}

	// Prepare texts for embedding
	texts := make([]string, len(events))
	for i, e := range events {
		texts[i] = formatEventForEmbedding(e)
	}

	// Generate embeddings
	embeddings, err := s.embedder.EmbedBatch(ctx, texts)
	if err != nil {
		return fmt.Errorf("generating embeddings: %w", err)
	}

	// Index each event
	for i, e := range events {
		node := vector.Node{
			ID:        e.ID,
			Content:   texts[i],
			Embedding: embeddings[i],
			Source:    "polymarket",
			Metadata: map[string]string{
				"type":        "event",
				"title":       e.Title,
				"description": truncateString(e.Description, 500),
				"slug":        e.Slug,
				"liquidity":   fmt.Sprintf("%.2f", e.Liquidity),
				"volume":      fmt.Sprintf("%.2f", e.Volume),
			},
		}

		if err := s.vectorIndex.Upsert(ctx, node); err != nil {
			return fmt.Errorf("indexing event %s: %w", e.ID, err)
		}
	}

	return nil
}

// SearchEvents performs semantic search for events.
func (s *Store) SearchEvents(ctx context.Context, query string, topK int) ([]EventSearchResult, error) {
	results, err := s.search(ctx, query, topK, map[string]string{"type": "event"})
	if err != nil {
		return nil, err
	}

	searchResults := make([]EventSearchResult, len(results))
	for i, r := range results {
		searchResults[i] = EventSearchResult{
			ID:          r.Node.ID,
			Score:       float32(r.Score),
			Title:       r.Node.Metadata["title"],
			Description: r.Node.Metadata["description"],
			Slug:        r.Node.Metadata["slug"],
			Liquidity:   r.Node.Metadata["liquidity"],
			Volume:      r.Node.Metadata["volume"],
		}
	}

	return searchResults, nil
}

// Retrieve performs a general retrieval query.
func (s *Store) Retrieve(ctx context.Context, query retrieve.Query) (*retrieve.Result, error) {
	// Generate embedding if not provided
	if len(query.Embedding) == 0 && query.Text != "" {
		embedding, err := s.embedder.Embed(ctx, query.Text)
		if err != nil {
			return nil, fmt.Errorf("generating embedding: %w", err)
		}
		query.Embedding = embedding
	}

	// Search vector index
	results, err := s.vectorIndex.Search(ctx, query.Embedding, query.TopK, query.Filters)
	if err != nil {
		return nil, fmt.Errorf("searching: %w", err)
	}

	// Convert to retrieve.Result
	items := make([]retrieve.ContextItem, len(results))
	for i, r := range results {
		items[i] = retrieve.ContextItem{
			ID:       r.Node.ID,
			Content:  r.Node.Content,
			Score:    r.Score,
			Metadata: r.Node.Metadata,
		}
	}

	return &retrieve.Result{
		Items: items,
		Query: query,
	}, nil
}

// MarketSearchResult represents a market search result.
type MarketSearchResult struct {
	ID          string  `json:"id"`
	Score       float32 `json:"score"`
	Question    string  `json:"question"`
	Description string  `json:"description"`
	Outcomes    string  `json:"outcomes"`
	Liquidity   string  `json:"liquidity"`
	EndDate     string  `json:"end_date"`
}

// EventSearchResult represents an event search result.
type EventSearchResult struct {
	ID          string  `json:"id"`
	Score       float32 `json:"score"`
	Title       string  `json:"title"`
	Description string  `json:"description"`
	Slug        string  `json:"slug"`
	Liquidity   string  `json:"liquidity"`
	Volume      string  `json:"volume"`
}

// EventForIndex holds event data for indexing.
type EventForIndex struct {
	ID          string
	Title       string
	Description string
	Slug        string
	Liquidity   float64
	Volume      float64
}

// formatMarketForEmbedding creates an embedding-friendly text for a market.
func formatMarketForEmbedding(m polymarket.Market) string {
	parts := []string{
		m.Question,
		m.Description,
	}
	if m.Outcomes != "" {
		parts = append(parts, "Outcomes: "+m.Outcomes)
	}
	return strings.Join(parts, " ")
}

// formatEventForEmbedding creates an embedding-friendly text for an event.
func formatEventForEmbedding(e EventForIndex) string {
	parts := []string{
		e.Title,
		e.Description,
	}
	return strings.Join(parts, " ")
}

// truncateString truncates a string to maxLen characters.
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
