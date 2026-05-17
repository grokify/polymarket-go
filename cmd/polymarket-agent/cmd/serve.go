package cmd

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/grokify/polymarket-go/internal/news"
	"github.com/grokify/polymarket-go/internal/polymarket"
	"github.com/grokify/polymarket-go/internal/server"
	"github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the REST API server",
	Long: `Starts the Polymarket Agent REST API server.

The server provides:
- Market data endpoints (/markets, /markets/{id})
- Order book and pricing (/markets/{tokenId}/orderbook, /markets/{tokenId}/price)
- News search (/news) - requires SERPER_API_KEY or SERPAPI_API_KEY
- Web search (/search) - requires SERPER_API_KEY or SERPAPI_API_KEY
- RAG semantic search (/rag/markets/search, /rag/events/search) - requires RAG store setup

OpenAPI spec is available at /docs.`,
	RunE: runServe,
}

var (
	servePort         int
	serveWithNews     bool
	serveSearchEngine string
)

func init() {
	rootCmd.AddCommand(serveCmd)

	serveCmd.Flags().IntVarP(&servePort, "port", "p", 8080, "Server port")
	serveCmd.Flags().BoolVar(&serveWithNews, "with-news", false, "Enable news/search endpoints (requires SERPER_API_KEY or SERPAPI_API_KEY)")
	serveCmd.Flags().StringVar(&serveSearchEngine, "search-engine", "", "Search engine to use (serper or serpapi)")
}

func runServe(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		slog.Info("shutdown signal received")
		cancel()
	}()

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	// Create Polymarket client
	pmClient := polymarket.NewClient()

	// Server config
	cfg := server.Config{
		Port:             servePort,
		Logger:           logger,
		PolymarketClient: pmClient,
	}

	// Optionally enable news search
	if serveWithNews {
		searcher, err := news.NewSearcher(news.SearcherConfig{
			Engine: serveSearchEngine,
		})
		if err != nil {
			logger.Warn("news search disabled", "error", err)
		} else {
			cfg.NewsSearcher = searcher
			logger.Info("news search enabled", "engine", serveSearchEngine)
		}
	}

	// Create and start server
	srv, err := server.New(cfg)
	if err != nil {
		return err
	}

	logger.Info("starting REST API server", "port", servePort)

	return srv.ListenAndServeContext(ctx)
}
