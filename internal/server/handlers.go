package server

import (
	"context"
	"fmt"

	"github.com/grokify/polymarket-go/internal/news"
	"github.com/grokify/polymarket-go/internal/polymarket"
)

// HealthInput is the input for the health check endpoint.
type HealthInput struct{}

// HealthOutput is the output for the health check endpoint.
type HealthOutput struct {
	Body struct {
		Status string `json:"status" example:"ok" doc:"Health status"`
	}
}

func (s *Server) handleHealth(ctx context.Context, input *HealthInput) (*HealthOutput, error) {
	resp := &HealthOutput{}
	resp.Body.Status = "ok"
	return resp, nil
}

// ListMarketsInput is the input for listing markets.
type ListMarketsInput struct {
	Limit        int     `query:"limit" default:"10" minimum:"1" maximum:"100" doc:"Maximum number of markets to return"`
	Active       string  `query:"active" enum:"true,false," doc:"Filter by active status (empty for any)"`
	Closed       string  `query:"closed" enum:"true,false," doc:"Filter by closed status (empty for any)"`
	MinLiquidity float64 `query:"min_liquidity" doc:"Minimum liquidity in USD"`
	MaxLiquidity float64 `query:"max_liquidity" doc:"Maximum liquidity in USD"`
	TextQuery    string  `query:"q" doc:"Text search query"`
	TagSlug      string  `query:"tag" doc:"Filter by tag slug"`
	Order        string  `query:"order" enum:"liquidity,volume,end_date," default:"liquidity" doc:"Sort order field"`
	Ascending    bool    `query:"ascending" doc:"Sort ascending instead of descending"`
	Cursor       string  `query:"cursor" doc:"Pagination cursor"`
}

// ListMarketsOutput is the output for listing markets.
type ListMarketsOutput struct {
	Body struct {
		Markets []MarketResponse `json:"markets" doc:"List of markets"`
		Count   int              `json:"count" doc:"Number of markets returned"`
	}
}

// MarketResponse represents a market in API responses.
type MarketResponse struct {
	ID             string  `json:"id" doc:"Market condition ID"`
	Question       string  `json:"question" doc:"Market question"`
	Slug           string  `json:"slug" doc:"URL slug"`
	Description    string  `json:"description" doc:"Market description"`
	Active         bool    `json:"active" doc:"Whether market is active"`
	Closed         bool    `json:"closed" doc:"Whether market is closed"`
	Liquidity      float64 `json:"liquidity" doc:"Total liquidity in USD"`
	Volume24hr     float64 `json:"volume_24hr" doc:"24-hour volume in USD"`
	Outcomes       string  `json:"outcomes" doc:"JSON array of outcomes"`
	OutcomePrices  string  `json:"outcome_prices" doc:"JSON array of outcome prices"`
	BestBid        float64 `json:"best_bid" doc:"Best bid price"`
	BestAsk        float64 `json:"best_ask" doc:"Best ask price"`
	Spread         float64 `json:"spread" doc:"Bid-ask spread"`
	LastTradePrice float64 `json:"last_trade_price" doc:"Last trade price"`
	EndDate        string  `json:"end_date" doc:"Market end date (ISO 8601)"`
	Image          string  `json:"image,omitempty" doc:"Market image URL"`
}

func (s *Server) handleListMarkets(ctx context.Context, input *ListMarketsInput) (*ListMarketsOutput, error) {
	params := polymarket.GetMarketsParams{
		Limit:     input.Limit,
		LiqMin:    input.MinLiquidity,
		LiqMax:    input.MaxLiquidity,
		TextQuery: input.TextQuery,
		TagSlug:   input.TagSlug,
		Order:     input.Order,
		Ascending: input.Ascending,
		Cursor:    input.Cursor,
	}

	// Convert string bools to *bool
	if input.Active == "true" {
		active := true
		params.Active = &active
	} else if input.Active == "false" {
		active := false
		params.Active = &active
	}
	if input.Closed == "true" {
		closed := true
		params.Closed = &closed
	} else if input.Closed == "false" {
		closed := false
		params.Closed = &closed
	}

	markets, err := s.pmClient.GetMarkets(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("fetching markets: %w", err)
	}

	resp := &ListMarketsOutput{}
	resp.Body.Markets = make([]MarketResponse, len(markets))
	for i, m := range markets {
		resp.Body.Markets[i] = marketToResponse(m)
	}
	resp.Body.Count = len(markets)

	return resp, nil
}

// GetMarketInput is the input for getting a single market.
type GetMarketInput struct {
	ConditionID string `path:"conditionId" doc:"Market condition ID"`
}

// GetMarketOutput is the output for getting a single market.
type GetMarketOutput struct {
	Body MarketResponse
}

func (s *Server) handleGetMarket(ctx context.Context, input *GetMarketInput) (*GetMarketOutput, error) {
	market, err := s.pmClient.GetMarket(ctx, input.ConditionID)
	if err != nil {
		return nil, fmt.Errorf("fetching market: %w", err)
	}

	resp := &GetMarketOutput{}
	resp.Body = marketToResponse(*market)
	return resp, nil
}

// GetOrderBookInput is the input for getting an order book.
type GetOrderBookInput struct {
	TokenID string `path:"tokenId" doc:"Market token ID"`
}

// GetOrderBookOutput is the output for getting an order book.
type GetOrderBookOutput struct {
	Body OrderBookResponse
}

// OrderBookResponse represents an order book in API responses.
type OrderBookResponse struct {
	Market string           `json:"market" doc:"Market identifier"`
	Asset  string           `json:"asset_id" doc:"Asset ID"`
	Hash   string           `json:"hash" doc:"Order book hash"`
	Bids   []OrderBookLevel `json:"bids" doc:"Bid levels"`
	Asks   []OrderBookLevel `json:"asks" doc:"Ask levels"`
}

// OrderBookLevel represents a price level.
type OrderBookLevel struct {
	Price string `json:"price" doc:"Price level"`
	Size  string `json:"size" doc:"Size at price level"`
}

func (s *Server) handleGetOrderBook(ctx context.Context, input *GetOrderBookInput) (*GetOrderBookOutput, error) {
	book, err := s.pmClient.GetOrderBook(ctx, input.TokenID)
	if err != nil {
		return nil, fmt.Errorf("fetching order book: %w", err)
	}

	resp := &GetOrderBookOutput{}
	resp.Body = OrderBookResponse{
		Market: book.Market,
		Asset:  book.Asset,
		Hash:   book.Hash,
		Bids:   make([]OrderBookLevel, len(book.Bids)),
		Asks:   make([]OrderBookLevel, len(book.Asks)),
	}
	for i, b := range book.Bids {
		resp.Body.Bids[i] = OrderBookLevel{Price: b.Price, Size: b.Size}
	}
	for i, a := range book.Asks {
		resp.Body.Asks[i] = OrderBookLevel{Price: a.Price, Size: a.Size}
	}

	return resp, nil
}

// GetPriceInput is the input for getting a token price.
type GetPriceInput struct {
	TokenID string `path:"tokenId" doc:"Market token ID"`
}

// GetPriceOutput is the output for getting a token price.
type GetPriceOutput struct {
	Body struct {
		TokenID  string  `json:"token_id" doc:"Token ID"`
		Price    float64 `json:"price" doc:"Current price"`
		MidPrice float64 `json:"mid_price,omitempty" doc:"Mid price"`
		Spread   float64 `json:"spread,omitempty" doc:"Bid-ask spread"`
	}
}

func (s *Server) handleGetPrice(ctx context.Context, input *GetPriceInput) (*GetPriceOutput, error) {
	price, err := s.pmClient.GetPrice(ctx, input.TokenID)
	if err != nil {
		return nil, fmt.Errorf("fetching price: %w", err)
	}

	resp := &GetPriceOutput{}
	resp.Body.TokenID = price.TokenID
	resp.Body.Price = price.Price

	// Optionally fetch mid price and spread
	if mid, err := s.pmClient.GetMidPrice(ctx, input.TokenID); err == nil {
		resp.Body.MidPrice = mid.Mid
	}
	if spread, err := s.pmClient.GetSpread(ctx, input.TokenID); err == nil {
		resp.Body.Spread = spread.Spread
	}

	return resp, nil
}

// SearchNewsInput is the input for searching news.
type SearchNewsInput struct {
	Query      string `query:"q" required:"true" doc:"Search query"`
	NumResults int    `query:"limit" default:"10" minimum:"1" maximum:"100" doc:"Number of results"`
	Language   string `query:"lang" doc:"Language code (e.g., en)"`
	Country    string `query:"country" doc:"Country code (e.g., us)"`
}

// SearchNewsOutput is the output for searching news.
type SearchNewsOutput struct {
	Body struct {
		Articles []NewsArticleResponse `json:"articles" doc:"List of news articles"`
		Count    int                   `json:"count" doc:"Number of articles returned"`
	}
}

// NewsArticleResponse represents a news article in API responses.
type NewsArticleResponse struct {
	Title     string `json:"title" doc:"Article title"`
	Link      string `json:"link" doc:"Article URL"`
	Source    string `json:"source" doc:"Source publication"`
	Date      string `json:"date" doc:"Publication date"`
	Snippet   string `json:"snippet" doc:"Article snippet"`
	Thumbnail string `json:"thumbnail,omitempty" doc:"Thumbnail image URL"`
}

func (s *Server) handleSearchNews(ctx context.Context, input *SearchNewsInput) (*SearchNewsOutput, error) {
	articles, err := s.searcher.SearchNews(ctx, input.Query, news.SearchOptions{
		NumResults: input.NumResults,
		Language:   input.Language,
		Country:    input.Country,
	})
	if err != nil {
		return nil, fmt.Errorf("searching news: %w", err)
	}

	resp := &SearchNewsOutput{}
	resp.Body.Articles = make([]NewsArticleResponse, len(articles))
	for i, a := range articles {
		resp.Body.Articles[i] = NewsArticleResponse{
			Title:     a.Title,
			Link:      a.Link,
			Source:    a.Source,
			Date:      a.Date,
			Snippet:   a.Snippet,
			Thumbnail: a.Thumbnail,
		}
	}
	resp.Body.Count = len(articles)

	return resp, nil
}

// SearchWebInput is the input for web search.
type SearchWebInput struct {
	Query      string `query:"q" required:"true" doc:"Search query"`
	NumResults int    `query:"limit" default:"10" minimum:"1" maximum:"100" doc:"Number of results"`
	Language   string `query:"lang" doc:"Language code (e.g., en)"`
	Country    string `query:"country" doc:"Country code (e.g., us)"`
}

// SearchWebOutput is the output for web search.
type SearchWebOutput struct {
	Body struct {
		Results   []WebSearchResultResponse `json:"results" doc:"Search results"`
		AnswerBox *AnswerBoxResponse        `json:"answer_box,omitempty" doc:"Featured answer snippet"`
		Count     int                       `json:"count" doc:"Number of results returned"`
	}
}

// WebSearchResultResponse represents a web search result.
type WebSearchResultResponse struct {
	Position int    `json:"position" doc:"Result position"`
	Title    string `json:"title" doc:"Page title"`
	Link     string `json:"link" doc:"Page URL"`
	Snippet  string `json:"snippet" doc:"Page snippet"`
	Domain   string `json:"domain" doc:"Domain name"`
	Date     string `json:"date,omitempty" doc:"Publication date"`
}

// AnswerBoxResponse represents a featured answer.
type AnswerBoxResponse struct {
	Title   string `json:"title,omitempty" doc:"Answer title"`
	Answer  string `json:"answer,omitempty" doc:"Direct answer"`
	Snippet string `json:"snippet,omitempty" doc:"Answer snippet"`
	Link    string `json:"link,omitempty" doc:"Source link"`
}

func (s *Server) handleSearchWeb(ctx context.Context, input *SearchWebInput) (*SearchWebOutput, error) {
	result, err := s.searcher.SearchWeb(ctx, input.Query, news.SearchOptions{
		NumResults: input.NumResults,
		Language:   input.Language,
		Country:    input.Country,
	})
	if err != nil {
		return nil, fmt.Errorf("searching web: %w", err)
	}

	resp := &SearchWebOutput{}
	resp.Body.Results = make([]WebSearchResultResponse, len(result.OrganicResults))
	for i, r := range result.OrganicResults {
		resp.Body.Results[i] = WebSearchResultResponse{
			Position: r.Position,
			Title:    r.Title,
			Link:     r.Link,
			Snippet:  r.Snippet,
			Domain:   r.Domain,
			Date:     r.Date,
		}
	}
	if result.AnswerBox != nil {
		resp.Body.AnswerBox = &AnswerBoxResponse{
			Title:   result.AnswerBox.Title,
			Answer:  result.AnswerBox.Answer,
			Snippet: result.AnswerBox.Snippet,
			Link:    result.AnswerBox.Link,
		}
	}
	resp.Body.Count = len(result.OrganicResults)

	return resp, nil
}

// RAGSearchInput is the input for RAG semantic search.
type RAGSearchInput struct {
	Body struct {
		Query string `json:"query" required:"true" doc:"Semantic search query"`
		TopK  int    `json:"top_k" default:"10" minimum:"1" maximum:"100" doc:"Number of results to return"`
	}
}

// RAGSearchMarketsOutput is the output for RAG market search.
type RAGSearchMarketsOutput struct {
	Body struct {
		Results []RAGMarketResult `json:"results" doc:"Search results"`
		Count   int               `json:"count" doc:"Number of results returned"`
	}
}

// RAGMarketResult represents a RAG market search result.
type RAGMarketResult struct {
	ID          string  `json:"id" doc:"Market condition ID"`
	Score       float32 `json:"score" doc:"Relevance score"`
	Question    string  `json:"question" doc:"Market question"`
	Description string  `json:"description" doc:"Market description"`
	Outcomes    string  `json:"outcomes" doc:"Market outcomes"`
	Liquidity   string  `json:"liquidity" doc:"Market liquidity"`
	EndDate     string  `json:"end_date" doc:"Market end date"`
}

func (s *Server) handleRAGSearchMarkets(ctx context.Context, input *RAGSearchInput) (*RAGSearchMarketsOutput, error) {
	topK := input.Body.TopK
	if topK == 0 {
		topK = 10
	}

	results, err := s.ragStore.SearchMarkets(ctx, input.Body.Query, topK)
	if err != nil {
		return nil, fmt.Errorf("searching markets: %w", err)
	}

	resp := &RAGSearchMarketsOutput{}
	resp.Body.Results = make([]RAGMarketResult, len(results))
	for i, r := range results {
		resp.Body.Results[i] = RAGMarketResult{
			ID:          r.ID,
			Score:       r.Score,
			Question:    r.Question,
			Description: r.Description,
			Outcomes:    r.Outcomes,
			Liquidity:   r.Liquidity,
			EndDate:     r.EndDate,
		}
	}
	resp.Body.Count = len(results)

	return resp, nil
}

// RAGSearchEventsOutput is the output for RAG event search.
type RAGSearchEventsOutput struct {
	Body struct {
		Results []RAGEventResult `json:"results" doc:"Search results"`
		Count   int              `json:"count" doc:"Number of results returned"`
	}
}

// RAGEventResult represents a RAG event search result.
type RAGEventResult struct {
	ID          string  `json:"id" doc:"Event ID"`
	Score       float32 `json:"score" doc:"Relevance score"`
	Title       string  `json:"title" doc:"Event title"`
	Description string  `json:"description" doc:"Event description"`
	Slug        string  `json:"slug" doc:"Event slug"`
	Liquidity   string  `json:"liquidity" doc:"Event liquidity"`
	Volume      string  `json:"volume" doc:"Event volume"`
}

func (s *Server) handleRAGSearchEvents(ctx context.Context, input *RAGSearchInput) (*RAGSearchEventsOutput, error) {
	topK := input.Body.TopK
	if topK == 0 {
		topK = 10
	}

	results, err := s.ragStore.SearchEvents(ctx, input.Body.Query, topK)
	if err != nil {
		return nil, fmt.Errorf("searching events: %w", err)
	}

	resp := &RAGSearchEventsOutput{}
	resp.Body.Results = make([]RAGEventResult, len(results))
	for i, r := range results {
		resp.Body.Results[i] = RAGEventResult{
			ID:          r.ID,
			Score:       r.Score,
			Title:       r.Title,
			Description: r.Description,
			Slug:        r.Slug,
			Liquidity:   r.Liquidity,
			Volume:      r.Volume,
		}
	}
	resp.Body.Count = len(results)

	return resp, nil
}

// RAGIndexMarketsInput is the input for indexing markets.
type RAGIndexMarketsInput struct {
	Body struct {
		Limit        int     `json:"limit" default:"100" doc:"Number of markets to index"`
		MinLiquidity float64 `json:"min_liquidity" doc:"Minimum liquidity filter"`
		ActiveOnly   bool    `json:"active_only" default:"true" doc:"Only index active markets"`
	}
}

// RAGIndexMarketsOutput is the output for indexing markets.
type RAGIndexMarketsOutput struct {
	Body struct {
		Indexed int    `json:"indexed" doc:"Number of markets indexed"`
		Message string `json:"message" doc:"Status message"`
	}
}

func (s *Server) handleRAGIndexMarkets(ctx context.Context, input *RAGIndexMarketsInput) (*RAGIndexMarketsOutput, error) {
	limit := input.Body.Limit
	if limit == 0 {
		limit = 100
	}

	// Fetch markets from Polymarket
	params := polymarket.GetMarketsParams{
		Limit:  limit,
		LiqMin: input.Body.MinLiquidity,
	}
	if input.Body.ActiveOnly {
		active := true
		params.Active = &active
	}

	markets, err := s.pmClient.GetMarkets(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("fetching markets: %w", err)
	}

	// Index into RAG store
	if err := s.ragStore.IndexMarkets(ctx, markets); err != nil {
		return nil, fmt.Errorf("indexing markets: %w", err)
	}

	resp := &RAGIndexMarketsOutput{}
	resp.Body.Indexed = len(markets)
	resp.Body.Message = fmt.Sprintf("Successfully indexed %d markets", len(markets))

	return resp, nil
}

// marketToResponse converts a polymarket.Market to a MarketResponse.
func marketToResponse(m polymarket.Market) MarketResponse {
	return MarketResponse{
		ID:             m.ConditionID,
		Question:       m.Question,
		Slug:           m.Slug,
		Description:    m.Description,
		Active:         m.Active,
		Closed:         m.Closed,
		Liquidity:      m.LiquidityNum,
		Volume24hr:     m.Volume24hr,
		Outcomes:       m.Outcomes,
		OutcomePrices:  m.OutcomePrices,
		BestBid:        m.BestBid,
		BestAsk:        m.BestAsk,
		Spread:         m.Spread,
		LastTradePrice: m.LastTradePrice,
		EndDate:        m.EndDateISO,
		Image:          m.Image,
	}
}
