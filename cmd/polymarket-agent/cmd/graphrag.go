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

var graphragCmd = &cobra.Command{
	Use:   "graphrag",
	Short: "GraphRAG operations for relationship-aware retrieval",
	Long:  `Commands for building and querying the knowledge graph of markets, events, and their relationships.`,
}

var graphragIndexCmd = &cobra.Command{
	Use:   "index",
	Short: "Build the knowledge graph from Polymarket data",
	Long: `Fetches markets and events from Polymarket and builds a knowledge graph
with relationships like event-to-market, market correlations, and topic clustering.`,
	RunE: runGraphRAGIndex,
}

var graphragRelatedCmd = &cobra.Command{
	Use:   "related [market-id]",
	Short: "Find markets related to a given market",
	Long:  `Traverses the knowledge graph to find correlated or semantically similar markets.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runGraphRAGRelated,
}

var graphragTopicCmd = &cobra.Command{
	Use:   "topic [topic-name]",
	Short: "Find events related to a topic",
	Long:  `Finds events related to a topic/tag through graph traversal.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runGraphRAGTopic,
}

var graphragHybridCmd = &cobra.Command{
	Use:   "hybrid [query]",
	Short: "Perform hybrid vector+graph search",
	Long:  `Combines semantic vector search with graph traversal for comprehensive results.`,
	Args:  cobra.MinimumNArgs(1),
	RunE:  runGraphRAGHybrid,
}

var (
	graphragDepth        int
	graphragLimit        int
	graphragJSON         bool
	graphragVectorWeight float64
	graphragGraphWeight  float64
)

func init() {
	rootCmd.AddCommand(graphragCmd)
	graphragCmd.AddCommand(graphragIndexCmd)
	graphragCmd.AddCommand(graphragRelatedCmd)
	graphragCmd.AddCommand(graphragTopicCmd)
	graphragCmd.AddCommand(graphragHybridCmd)

	// Related flags
	graphragRelatedCmd.Flags().IntVar(&graphragDepth, "depth", 2, "Traversal depth")
	graphragRelatedCmd.Flags().IntVarP(&graphragLimit, "limit", "l", 10, "Maximum results")
	graphragRelatedCmd.Flags().BoolVar(&graphragJSON, "json", false, "Output as JSON")

	// Topic flags
	graphragTopicCmd.Flags().IntVarP(&graphragLimit, "limit", "l", 10, "Maximum results")
	graphragTopicCmd.Flags().BoolVar(&graphragJSON, "json", false, "Output as JSON")

	// Hybrid flags
	graphragHybridCmd.Flags().IntVar(&graphragDepth, "depth", 2, "Traversal depth")
	graphragHybridCmd.Flags().IntVarP(&graphragLimit, "limit", "l", 10, "Maximum results")
	graphragHybridCmd.Flags().Float64Var(&graphragVectorWeight, "vector-weight", 0.6, "Weight for vector results")
	graphragHybridCmd.Flags().Float64Var(&graphragGraphWeight, "graph-weight", 0.4, "Weight for graph results")
	graphragHybridCmd.Flags().BoolVar(&graphragJSON, "json", false, "Output as JSON")
}

// Global graph store (in-memory for now)
var globalGraphStore *rag.GraphStore

func getOrCreateGraphStore() (*rag.GraphStore, error) {
	if globalGraphStore != nil {
		return globalGraphStore, nil
	}

	// Create in-memory graph
	memGraph := memory.NewKnowledgeGraph("polymarket")

	// Create embedder for hybrid search
	embedder, err := rag.NewOpenAIEmbedder(rag.OpenAIEmbedderConfig{})
	if err != nil {
		// Embedder is optional for non-hybrid operations
		embedder = nil
	}

	// Create vector index for hybrid search
	var vectorIndex *memory.VectorIndex
	if embedder != nil {
		vectorIndex = memory.NewVectorIndex("polymarket")
	}

	store, err := rag.NewGraphStore(rag.GraphStoreConfig{
		Graph:       memGraph,
		VectorIndex: vectorIndex,
		Embedder:    embedder,
	})
	if err != nil {
		return nil, err
	}

	globalGraphStore = store
	return store, nil
}

func runGraphRAGIndex(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	store, err := getOrCreateGraphStore()
	if err != nil {
		return fmt.Errorf("creating graph store: %w", err)
	}

	// Fetch events from Gamma API
	logger.Info("fetching events from Polymarket")
	sdkClient, err := polymarket.NewSDKClient(polymarket.SDKConfig{})
	if err != nil {
		return fmt.Errorf("creating SDK client: %w", err)
	}

	gammaEvents, err := sdkClient.GetEventsGamma(ctx)
	if err != nil {
		return fmt.Errorf("fetching events: %w", err)
	}

	logger.Info("indexing events and markets", "event_count", len(gammaEvents))

	eventCount := 0
	marketCount := 0

	for _, e := range gammaEvents {
		liq, _ := e.Liquidity.Float64()
		vol, _ := e.Volume.Float64()

		// Collect market IDs and tags
		var marketIDs []string
		var allTags []string
		for _, m := range e.Markets {
			marketIDs = append(marketIDs, m.ConditionID)
			for _, t := range m.Tags {
				allTags = append(allTags, t.Label)
			}
		}

		// Add category as a tag
		if e.Category != "" {
			allTags = append([]string{e.Category}, allTags...)
		}

		// Index event
		event := rag.EventForGraph{
			ID:          e.ID,
			Title:       e.Title,
			Description: e.Description,
			Slug:        e.Slug,
			Tags:        allTags,
			MarketIDs:   marketIDs,
			Liquidity:   liq,
			Volume:      vol,
		}

		if err := store.IndexEvent(ctx, event); err != nil {
			logger.Warn("failed to index event", "id", e.ID, "error", err)
			continue
		}
		eventCount++

		// Index markets under this event
		for _, m := range e.Markets {
			mliq, _ := m.Liquidity.Float64()
			mvol, _ := m.Volume.Float64()

			// Extract tag labels
			var marketTags []string
			for _, t := range m.Tags {
				marketTags = append(marketTags, t.Label)
			}

			market := rag.MarketForGraph{
				ID:          m.ConditionID,
				EventID:     e.ID,
				Question:    m.Question,
				Description: "", // Market doesn't have Description field
				Outcomes:    m.Outcomes,
				Category:    e.Category,
				Tags:        marketTags,
				Liquidity:   mliq,
				Volume:      mvol,
				OutcomeYes:  0, // Parsed from OutcomePrices if needed
				OutcomeNo:   0,
			}

			if err := store.IndexMarket(ctx, market); err != nil {
				logger.Warn("failed to index market", "id", m.ConditionID, "error", err)
				continue
			}
			marketCount++
		}

		// Link sibling markets (markets under the same event)
		if len(marketIDs) > 1 {
			if err := store.LinkSiblingMarkets(ctx, e.ID, marketIDs); err != nil {
				logger.Warn("failed to link siblings", "event_id", e.ID, "error", err)
			}
		}
	}

	logger.Info("indexing complete",
		"events_indexed", eventCount,
		"markets_indexed", marketCount,
	)

	return nil
}

func runGraphRAGRelated(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	marketID := args[0]

	store, err := getOrCreateGraphStore()
	if err != nil {
		return fmt.Errorf("creating graph store: %w", err)
	}

	logger.Info("finding related markets", "market_id", marketID, "depth", graphragDepth)

	results, err := store.FindRelatedMarkets(ctx, marketID, graphragDepth)
	if err != nil {
		return fmt.Errorf("finding related markets: %w", err)
	}

	// Limit results
	if len(results) > graphragLimit {
		results = results[:graphragLimit]
	}

	if graphragJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(results)
	}

	logger.Info("related markets", "count", len(results))
	for i, r := range results {
		logger.Info(fmt.Sprintf("result %d", i+1),
			"id", r.ID,
			"score", fmt.Sprintf("%.4f", r.Score),
			"type", r.Type,
			"question", truncate(r.Metadata["question"], 60),
		)
		if len(r.EdgeTypes) > 0 {
			fmt.Printf("   via: %v\n", r.EdgeTypes)
		}
	}

	return nil
}

func runGraphRAGTopic(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	topic := args[0]

	store, err := getOrCreateGraphStore()
	if err != nil {
		return fmt.Errorf("creating graph store: %w", err)
	}

	logger.Info("finding events by topic", "topic", topic)

	results, err := store.FindEventsByTopic(ctx, topic)
	if err != nil {
		return fmt.Errorf("finding events: %w", err)
	}

	// Limit results
	if len(results) > graphragLimit {
		results = results[:graphragLimit]
	}

	if graphragJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(results)
	}

	logger.Info("events for topic", "topic", topic, "count", len(results))
	for i, r := range results {
		logger.Info(fmt.Sprintf("result %d", i+1),
			"id", r.ID,
			"score", fmt.Sprintf("%.4f", r.Score),
			"title", truncate(r.Metadata["title"], 60),
		)
	}

	return nil
}

func runGraphRAGHybrid(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	query := args[0]

	store, err := getOrCreateGraphStore()
	if err != nil {
		return fmt.Errorf("creating graph store: %w", err)
	}

	logger.Info("performing hybrid search",
		"query", query,
		"vector_weight", graphragVectorWeight,
		"graph_weight", graphragGraphWeight,
	)

	results, err := store.HybridSearch(ctx, query, rag.HybridSearchOptions{
		TopK:         graphragLimit,
		VectorWeight: graphragVectorWeight,
		GraphWeight:  graphragGraphWeight,
		MaxDepth:     graphragDepth,
	})
	if err != nil {
		return fmt.Errorf("hybrid search: %w", err)
	}

	if graphragJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(results)
	}

	logger.Info("hybrid results", "count", len(results))
	for i, r := range results {
		question := r.Metadata["question"]
		if question == "" {
			question = r.Metadata["title"]
		}
		logger.Info(fmt.Sprintf("result %d", i+1),
			"id", r.ID,
			"score", fmt.Sprintf("%.4f", r.Score),
			"type", r.Type,
			"content", truncate(question, 60),
		)
	}

	return nil
}
