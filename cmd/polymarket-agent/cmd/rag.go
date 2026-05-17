package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"

	"github.com/agentplexus/omniretrieve/memory"
	"github.com/grokify/polymarket-go/internal/polymarket"
	"github.com/grokify/polymarket-go/internal/rag"
	"github.com/spf13/cobra"
)

var ragCmd = &cobra.Command{
	Use:   "rag",
	Short: "RAG (Retrieval-Augmented Generation) operations",
	Long:  `Commands for indexing and searching markets/events using vector similarity.`,
}

var ragIndexCmd = &cobra.Command{
	Use:   "index",
	Short: "Index markets and events into vector store",
	Long: `Fetches markets and events from Polymarket and indexes them
into the vector store for semantic search.

Requires OPENAI_API_KEY for embeddings.`,
	RunE: runRAGIndex,
}

var ragSearchCmd = &cobra.Command{
	Use:   "search [query]",
	Short: "Search markets and events semantically",
	Long: `Performs semantic search across indexed markets and events.

Requires OPENAI_API_KEY for embeddings.`,
	Args: cobra.MinimumNArgs(1),
	RunE: runRAGSearch,
}

var (
	ragIndexType   string
	ragSearchType  string
	ragSearchLimit int
	ragSearchJSON  bool
	ragUseInMemory bool
)

func init() {
	rootCmd.AddCommand(ragCmd)
	ragCmd.AddCommand(ragIndexCmd)
	ragCmd.AddCommand(ragSearchCmd)

	// Index flags
	ragIndexCmd.Flags().StringVar(&ragIndexType, "type", "all", "Type to index: markets, events, or all")
	ragIndexCmd.Flags().BoolVar(&ragUseInMemory, "in-memory", true, "Use in-memory store (for testing)")

	// Search flags
	ragSearchCmd.Flags().StringVar(&ragSearchType, "type", "all", "Type to search: markets, events, or all")
	ragSearchCmd.Flags().IntVarP(&ragSearchLimit, "limit", "l", 5, "Number of results to return")
	ragSearchCmd.Flags().BoolVar(&ragSearchJSON, "json", false, "Output as JSON")
	ragSearchCmd.Flags().BoolVar(&ragUseInMemory, "in-memory", true, "Use in-memory store (for testing)")
}

func runRAGIndex(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	// Create embedder
	embedder, err := rag.NewOpenAIEmbedder(rag.OpenAIEmbedderConfig{})
	if err != nil {
		return fmt.Errorf("creating embedder: %w", err)
	}

	// Create vector index (in-memory for now)
	vectorIndex := memory.NewVectorIndex("polymarket")

	// Create RAG store
	store, err := rag.NewStore(rag.StoreConfig{
		VectorIndex: vectorIndex,
		Embedder:    embedder,
		Dimensions:  embedder.Dimensions(),
	})
	if err != nil {
		return fmt.Errorf("creating RAG store: %w", err)
	}

	// Fetch and index markets
	if ragIndexType == "all" || ragIndexType == "markets" {
		logger.Info("fetching markets from Polymarket")
		pmClient := polymarket.NewClient()
		active := true
		markets, err := pmClient.GetMarkets(ctx, polymarket.GetMarketsParams{
			Active: &active,
			Limit:  100,
		})
		if err != nil {
			return fmt.Errorf("fetching markets: %w", err)
		}

		logger.Info("indexing markets", "count", len(markets))
		if err := store.IndexMarkets(ctx, markets); err != nil {
			return fmt.Errorf("indexing markets: %w", err)
		}
		logger.Info("indexed markets", "count", len(markets))
	}

	// Fetch and index events
	if ragIndexType == "all" || ragIndexType == "events" {
		logger.Info("fetching events from Polymarket")
		sdkClient, err := polymarket.NewSDKClient(polymarket.SDKConfig{})
		if err != nil {
			return fmt.Errorf("creating SDK client: %w", err)
		}

		gammaEvents, err := sdkClient.GetEventsGamma(ctx)
		if err != nil {
			return fmt.Errorf("fetching events: %w", err)
		}

		// Convert to EventForIndex
		events := make([]rag.EventForIndex, len(gammaEvents))
		for i, e := range gammaEvents {
			liq, _ := e.Liquidity.Float64()
			vol, _ := e.Volume.Float64()
			events[i] = rag.EventForIndex{
				ID:          e.ID,
				Title:       e.Title,
				Description: e.Description,
				Slug:        e.Slug,
				Liquidity:   liq,
				Volume:      vol,
			}
		}

		logger.Info("indexing events", "count", len(events))
		if err := store.IndexEvents(ctx, events); err != nil {
			return fmt.Errorf("indexing events: %w", err)
		}
		logger.Info("indexed events", "count", len(events))
	}

	logger.Info("indexing complete")
	return nil
}

func runRAGSearch(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	query := args[0]

	// Create embedder
	embedder, err := rag.NewOpenAIEmbedder(rag.OpenAIEmbedderConfig{})
	if err != nil {
		return fmt.Errorf("creating embedder: %w", err)
	}

	// Create vector index (in-memory for now)
	// Note: In production, this would load from pgvector or a persistent store
	vectorIndex := memory.NewVectorIndex("polymarket")

	// Create RAG store
	store, err := rag.NewStore(rag.StoreConfig{
		VectorIndex: vectorIndex,
		Embedder:    embedder,
		Dimensions:  embedder.Dimensions(),
	})
	if err != nil {
		return fmt.Errorf("creating RAG store: %w", err)
	}

	// For demo purposes, we need to index first
	// In production, this would use a persistent store
	logger.Info("note: using in-memory store - data must be indexed first in same session")

	// Search markets
	if ragSearchType == "all" || ragSearchType == "markets" {
		results, err := store.SearchMarkets(ctx, query, ragSearchLimit)
		if err != nil {
			return fmt.Errorf("searching markets: %w", err)
		}

		if ragSearchJSON {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			if err := enc.Encode(results); err != nil {
				return err
			}
		} else {
			logger.Info("market results", "count", len(results))
			for i, r := range results {
				logger.Info(fmt.Sprintf("result %d", i+1),
					"score", fmt.Sprintf("%.4f", r.Score),
					"question", truncate(r.Question, 50),
					"liquidity", r.Liquidity,
				)
			}
		}
	}

	// Search events
	if ragSearchType == "all" || ragSearchType == "events" {
		results, err := store.SearchEvents(ctx, query, ragSearchLimit)
		if err != nil {
			return fmt.Errorf("searching events: %w", err)
		}

		if ragSearchJSON {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			if err := enc.Encode(results); err != nil {
				return err
			}
		} else {
			logger.Info("event results", "count", len(results))
			for i, r := range results {
				logger.Info(fmt.Sprintf("result %d", i+1),
					"score", fmt.Sprintf("%.4f", r.Score),
					"title", truncate(r.Title, 50),
					"liquidity", r.Liquidity,
				)
			}
		}
	}

	return nil
}
