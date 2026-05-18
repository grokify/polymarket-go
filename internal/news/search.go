// Package news provides news and web search capabilities for Polymarket market research.
// It uses omniserp for multi-provider search (Serper, SerpAPI).
package news

import (
	"context"
	"strings"

	perrors "github.com/grokify/polymarket-go/internal/errors"
	"github.com/plexusone/omniserp"
	"github.com/plexusone/omniserp/client"
)

// Searcher provides news and web search capabilities.
type Searcher struct {
	client *client.Client
}

// SearcherConfig holds configuration for the searcher.
type SearcherConfig struct {
	// Engine is the search engine to use ("serper" or "serpapi").
	// If empty, uses SEARCH_ENGINE env var or defaults to "serper".
	Engine string
}

// NewSearcher creates a new Searcher.
func NewSearcher(cfg SearcherConfig) (*Searcher, error) {
	var c *client.Client
	var err error

	if cfg.Engine != "" {
		c, err = client.NewWithEngine(cfg.Engine)
	} else {
		c, err = client.New()
	}
	if err != nil {
		return nil, &perrors.ConfigurationError{
			Component: "Searcher",
			Setting:   "engine",
			Reason:    "creating search client",
			Err:       err,
		}
	}

	return &Searcher{client: c}, nil
}

// SearchNews searches for news articles.
func (s *Searcher) SearchNews(ctx context.Context, query string, opts SearchOptions) ([]NewsArticle, error) {
	params := omniserp.SearchParams{
		Query:      query,
		NumResults: opts.NumResults,
		Language:   opts.Language,
		Country:    opts.Country,
	}

	if params.NumResults == 0 {
		params.NumResults = 10
	}

	result, err := s.client.SearchNewsNormalized(ctx, params)
	if err != nil {
		return nil, &perrors.SearchError{
			Engine: "omniserp",
			Query:  query,
			Reason: "news search failed",
			Err:    err,
		}
	}

	articles := make([]NewsArticle, len(result.NewsResults))
	for i, r := range result.NewsResults {
		articles[i] = NewsArticle{
			Title:     r.Title,
			Link:      r.Link,
			Source:    r.Source,
			Date:      r.Date,
			Snippet:   r.Snippet,
			Thumbnail: r.Thumbnail,
		}
	}

	return articles, nil
}

// SearchWeb performs a general web search.
func (s *Searcher) SearchWeb(ctx context.Context, query string, opts SearchOptions) (*WebSearchResult, error) {
	params := omniserp.SearchParams{
		Query:      query,
		NumResults: opts.NumResults,
		Language:   opts.Language,
		Country:    opts.Country,
	}

	if params.NumResults == 0 {
		params.NumResults = 10
	}

	result, err := s.client.SearchNormalized(ctx, params)
	if err != nil {
		return nil, &perrors.SearchError{
			Engine: "omniserp",
			Query:  query,
			Reason: "web search failed",
			Err:    err,
		}
	}

	webResult := &WebSearchResult{
		OrganicResults: make([]OrganicResult, len(result.OrganicResults)),
	}

	for i, r := range result.OrganicResults {
		webResult.OrganicResults[i] = OrganicResult{
			Position: r.Position,
			Title:    r.Title,
			Link:     r.Link,
			Snippet:  r.Snippet,
			Domain:   r.Domain,
			Date:     r.Date,
		}
	}

	if result.AnswerBox != nil {
		webResult.AnswerBox = &AnswerBox{
			Title:   result.AnswerBox.Title,
			Answer:  result.AnswerBox.Answer,
			Snippet: result.AnswerBox.Snippet,
			Link:    result.AnswerBox.Link,
		}
	}

	return webResult, nil
}

// GetNewsForMarket searches for news relevant to a market question.
func (s *Searcher) GetNewsForMarket(ctx context.Context, question string, opts SearchOptions) ([]NewsArticle, error) {
	// Extract key terms from the question
	query := extractKeyTerms(question)
	return s.SearchNews(ctx, query, opts)
}

// GetNewsForKeywords searches for news by keywords.
func (s *Searcher) GetNewsForKeywords(ctx context.Context, keywords []string, opts SearchOptions) ([]NewsArticle, error) {
	query := strings.Join(keywords, " ")
	return s.SearchNews(ctx, query, opts)
}

// ScrapeWebpage extracts content from a URL.
func (s *Searcher) ScrapeWebpage(ctx context.Context, url string) (*ScrapedContent, error) {
	result, err := s.client.ScrapeWebpage(ctx, omniserp.ScrapeParams{
		URL: url,
	})
	if err != nil {
		return nil, &perrors.SearchError{
			Engine: "omniserp",
			Query:  url,
			Reason: "webpage scrape failed",
			Err:    err,
		}
	}

	// Result.Data contains the scraped content
	content := &ScrapedContent{
		URL: url,
	}

	if data, ok := result.Data.(map[string]any); ok {
		if text, ok := data["text"].(string); ok {
			content.Text = text
		}
		if title, ok := data["title"].(string); ok {
			content.Title = title
		}
	}

	return content, nil
}

// SearchOptions holds options for search operations.
type SearchOptions struct {
	NumResults int    // Number of results (default: 10)
	Language   string // Language code (e.g., "en")
	Country    string // Country code (e.g., "us")
}

// NewsArticle represents a news article.
type NewsArticle struct {
	Title     string `json:"title"`
	Link      string `json:"link"`
	Source    string `json:"source"`
	Date      string `json:"date"`
	Snippet   string `json:"snippet"`
	Thumbnail string `json:"thumbnail,omitempty"`
}

// WebSearchResult represents web search results.
type WebSearchResult struct {
	OrganicResults []OrganicResult `json:"organic_results"`
	AnswerBox      *AnswerBox      `json:"answer_box,omitempty"`
}

// OrganicResult represents a web search result.
type OrganicResult struct {
	Position int    `json:"position"`
	Title    string `json:"title"`
	Link     string `json:"link"`
	Snippet  string `json:"snippet"`
	Domain   string `json:"domain"`
	Date     string `json:"date,omitempty"`
}

// AnswerBox represents a featured answer snippet.
type AnswerBox struct {
	Title   string `json:"title,omitempty"`
	Answer  string `json:"answer,omitempty"`
	Snippet string `json:"snippet,omitempty"`
	Link    string `json:"link,omitempty"`
}

// ScrapedContent represents scraped webpage content.
type ScrapedContent struct {
	URL   string `json:"url"`
	Title string `json:"title"`
	Text  string `json:"text"`
}

// extractKeyTerms extracts key search terms from a question.
func extractKeyTerms(question string) string {
	// Remove common question words
	stopWords := []string{
		"will", "the", "a", "an", "is", "are", "was", "were",
		"be", "been", "being", "have", "has", "had",
		"do", "does", "did", "can", "could", "would", "should",
		"what", "when", "where", "who", "why", "how",
		"this", "that", "these", "those",
		"in", "on", "at", "to", "for", "of", "with", "by",
		"and", "or", "but", "not", "from", "into",
	}

	words := strings.Fields(strings.ToLower(question))
	var filtered []string

	stopSet := make(map[string]bool)
	for _, w := range stopWords {
		stopSet[w] = true
	}

	for _, word := range words {
		// Remove punctuation
		word = strings.Trim(word, "?!.,;:'\"")
		if word != "" && !stopSet[word] && len(word) > 2 {
			filtered = append(filtered, word)
		}
	}

	return strings.Join(filtered, " ")
}
