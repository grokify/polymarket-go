package rag

import (
	"context"

	"github.com/grokify/polymarket-go/internal/prompts"
)

// StoreAdapter adapts the Store to implement prompts.RAGSearcher.
type StoreAdapter struct {
	store *Store
}

// NewStoreAdapter creates a new adapter for the Store.
func NewStoreAdapter(store *Store) *StoreAdapter {
	return &StoreAdapter{store: store}
}

// SearchMarkets implements prompts.RAGSearcher.
func (a *StoreAdapter) SearchMarkets(ctx context.Context, query string, topK int) ([]prompts.RAGMarketResult, error) {
	results, err := a.store.SearchMarkets(ctx, query, topK)
	if err != nil {
		return nil, err
	}

	adapted := make([]prompts.RAGMarketResult, len(results))
	for i, r := range results {
		adapted[i] = prompts.RAGMarketResult{
			ID:          r.ID,
			Score:       r.Score,
			Question:    r.Question,
			Description: r.Description,
			Outcomes:    r.Outcomes,
			Liquidity:   r.Liquidity,
			EndDate:     r.EndDate,
		}
	}

	return adapted, nil
}

// SearchEvents implements prompts.RAGSearcher.
func (a *StoreAdapter) SearchEvents(ctx context.Context, query string, topK int) ([]prompts.RAGEventResult, error) {
	results, err := a.store.SearchEvents(ctx, query, topK)
	if err != nil {
		return nil, err
	}

	adapted := make([]prompts.RAGEventResult, len(results))
	for i, r := range results {
		adapted[i] = prompts.RAGEventResult{
			ID:          r.ID,
			Score:       r.Score,
			Title:       r.Title,
			Description: r.Description,
			Slug:        r.Slug,
			Liquidity:   r.Liquidity,
			Volume:      r.Volume,
		}
	}

	return adapted, nil
}
