// Package server provides a REST API server for Polymarket operations.
// It uses Huma with Chi router for automatic OpenAPI spec generation.
package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/grokify/polymarket-go/internal/news"
	"github.com/grokify/polymarket-go/internal/polymarket"
	"github.com/grokify/polymarket-go/internal/rag"
)

// Server is the REST API server.
type Server struct {
	router   chi.Router
	api      huma.API
	config   Config
	logger   *slog.Logger
	pmClient *polymarket.Client
	searcher *news.Searcher
	ragStore *rag.Store
}

// Config holds server configuration.
type Config struct {
	// Port is the server port (default: 8080).
	Port int

	// Logger is the structured logger.
	Logger *slog.Logger

	// PolymarketClient is the Polymarket API client.
	PolymarketClient *polymarket.Client

	// NewsSearcher is the news search client (optional).
	NewsSearcher *news.Searcher

	// RAGStore is the RAG store (optional).
	RAGStore *rag.Store
}

// New creates a new REST API server.
func New(cfg Config) (*Server, error) {
	if cfg.Port == 0 {
		cfg.Port = 8080
	}
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}
	if cfg.PolymarketClient == nil {
		cfg.PolymarketClient = polymarket.NewClient()
	}

	// Create Chi router with middleware
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	// Create Huma API with OpenAPI config
	humaConfig := huma.DefaultConfig("Polymarket Agent API", "1.0.0")
	humaConfig.Info.Description = "REST API for Polymarket prediction markets with AI-powered analysis"
	humaConfig.Servers = []*huma.Server{
		{URL: fmt.Sprintf("http://localhost:%d", cfg.Port)},
	}

	api := humachi.New(r, humaConfig)

	s := &Server{
		router:   r,
		api:      api,
		config:   cfg,
		logger:   cfg.Logger,
		pmClient: cfg.PolymarketClient,
		searcher: cfg.NewsSearcher,
		ragStore: cfg.RAGStore,
	}

	// Register routes
	s.registerRoutes()

	return s, nil
}

// Router returns the underlying Chi router.
func (s *Server) Router() chi.Router {
	return s.router
}

// API returns the Huma API for testing or customization.
func (s *Server) API() huma.API {
	return s.api
}

// newHTTPServer creates a new http.Server with proper timeouts.
func (s *Server) newHTTPServer() *http.Server {
	return &http.Server{
		Addr:              fmt.Sprintf(":%d", s.config.Port),
		Handler:           s.router,
		ReadTimeout:       15 * time.Second,
		ReadHeaderTimeout: 10 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
	}
}

// ListenAndServe starts the HTTP server.
func (s *Server) ListenAndServe() error {
	srv := s.newHTTPServer()
	s.logger.Info("starting server", "addr", srv.Addr)
	return srv.ListenAndServe()
}

// ListenAndServeContext starts the HTTP server with context for graceful shutdown.
func (s *Server) ListenAndServeContext(ctx context.Context) error {
	srv := s.newHTTPServer()

	// Graceful shutdown in background goroutine
	shutdownComplete := make(chan struct{})
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second) //nolint:govet // intentional shadow
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			s.logger.Error("server shutdown error", "error", err)
		}
		close(shutdownComplete)
	}()

	s.logger.Info("starting server", "addr", srv.Addr)
	err := srv.ListenAndServe()

	// Wait for shutdown goroutine if we're shutting down gracefully
	if err == http.ErrServerClosed {
		<-shutdownComplete
		return nil
	}
	return err
}
