package server

import (
	"github.com/danielgtaylor/huma/v2"
)

// registerRoutes registers all API routes.
func (s *Server) registerRoutes() {
	// Health check
	huma.Register(s.api, huma.Operation{
		OperationID: "health",
		Method:      "GET",
		Path:        "/health",
		Summary:     "Health check",
		Description: "Returns server health status",
		Tags:        []string{"System"},
	}, s.handleHealth)

	// Markets
	huma.Register(s.api, huma.Operation{
		OperationID: "listMarkets",
		Method:      "GET",
		Path:        "/markets",
		Summary:     "List markets",
		Description: "Fetches markets from Polymarket with optional filters",
		Tags:        []string{"Markets"},
	}, s.handleListMarkets)

	huma.Register(s.api, huma.Operation{
		OperationID: "getMarket",
		Method:      "GET",
		Path:        "/markets/{conditionId}",
		Summary:     "Get market by condition ID",
		Description: "Fetches a single market by its condition ID",
		Tags:        []string{"Markets"},
	}, s.handleGetMarket)

	huma.Register(s.api, huma.Operation{
		OperationID: "getOrderBook",
		Method:      "GET",
		Path:        "/markets/{tokenId}/orderbook",
		Summary:     "Get order book",
		Description: "Fetches the order book for a market token",
		Tags:        []string{"Markets"},
	}, s.handleGetOrderBook)

	huma.Register(s.api, huma.Operation{
		OperationID: "getPrice",
		Method:      "GET",
		Path:        "/markets/{tokenId}/price",
		Summary:     "Get token price",
		Description: "Fetches the current price for a market token",
		Tags:        []string{"Markets"},
	}, s.handleGetPrice)

	// News search (if searcher is configured)
	if s.searcher != nil {
		huma.Register(s.api, huma.Operation{
			OperationID: "searchNews",
			Method:      "GET",
			Path:        "/news",
			Summary:     "Search news",
			Description: "Searches for news articles relevant to prediction markets",
			Tags:        []string{"News"},
		}, s.handleSearchNews)

		huma.Register(s.api, huma.Operation{
			OperationID: "searchWeb",
			Method:      "GET",
			Path:        "/search",
			Summary:     "Web search",
			Description: "Performs a general web search",
			Tags:        []string{"Search"},
		}, s.handleSearchWeb)
	}

	// RAG search (if RAG store is configured)
	if s.ragStore != nil {
		huma.Register(s.api, huma.Operation{
			OperationID: "ragSearchMarkets",
			Method:      "POST",
			Path:        "/rag/markets/search",
			Summary:     "Semantic search for markets",
			Description: "Performs semantic search over indexed markets using RAG",
			Tags:        []string{"RAG"},
		}, s.handleRAGSearchMarkets)

		huma.Register(s.api, huma.Operation{
			OperationID: "ragSearchEvents",
			Method:      "POST",
			Path:        "/rag/events/search",
			Summary:     "Semantic search for events",
			Description: "Performs semantic search over indexed events using RAG",
			Tags:        []string{"RAG"},
		}, s.handleRAGSearchEvents)

		huma.Register(s.api, huma.Operation{
			OperationID: "ragIndexMarkets",
			Method:      "POST",
			Path:        "/rag/markets/index",
			Summary:     "Index markets",
			Description: "Indexes markets into the RAG vector store",
			Tags:        []string{"RAG"},
		}, s.handleRAGIndexMarkets)
	}
}
