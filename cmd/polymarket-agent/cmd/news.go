package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/grokify/polymarket-go/internal/news"
	"github.com/spf13/cobra"
)

var newsCmd = &cobra.Command{
	Use:   "news [query]",
	Short: "Search for news articles",
	Long: `Searches for news articles using Serper or SerpAPI.

Requires SERPER_API_KEY or SERPAPI_API_KEY environment variable.`,
	Args: cobra.MinimumNArgs(1),
	RunE: runNews,
}

var searchCmd = &cobra.Command{
	Use:   "search [query]",
	Short: "Perform web search",
	Long: `Performs a general web search using Serper or SerpAPI.

Requires SERPER_API_KEY or SERPAPI_API_KEY environment variable.`,
	Args: cobra.MinimumNArgs(1),
	RunE: runSearch,
}

var (
	newsLimit    int
	newsEngine   string
	newsLanguage string
	newsCountry  string
	newsJSON     bool

	searchLimit    int
	searchEngine   string
	searchLanguage string
	searchCountry  string
	searchJSON     bool
)

func init() {
	rootCmd.AddCommand(newsCmd)
	rootCmd.AddCommand(searchCmd)

	// News flags
	newsCmd.Flags().IntVarP(&newsLimit, "limit", "l", 10, "Number of results")
	newsCmd.Flags().StringVarP(&newsEngine, "engine", "e", "", "Search engine (serper or serpapi)")
	newsCmd.Flags().StringVar(&newsLanguage, "lang", "en", "Language code")
	newsCmd.Flags().StringVar(&newsCountry, "country", "us", "Country code")
	newsCmd.Flags().BoolVar(&newsJSON, "json", false, "Output as JSON")

	// Search flags
	searchCmd.Flags().IntVarP(&searchLimit, "limit", "l", 10, "Number of results")
	searchCmd.Flags().StringVarP(&searchEngine, "engine", "e", "", "Search engine (serper or serpapi)")
	searchCmd.Flags().StringVar(&searchLanguage, "lang", "en", "Language code")
	searchCmd.Flags().StringVar(&searchCountry, "country", "us", "Country code")
	searchCmd.Flags().BoolVar(&searchJSON, "json", false, "Output as JSON")
}

func runNews(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	query := strings.Join(args, " ")

	searcher, err := news.NewSearcher(news.SearcherConfig{
		Engine: newsEngine,
	})
	if err != nil {
		return fmt.Errorf("creating searcher: %w", err)
	}

	logger.Info("searching news", "query", query)

	articles, err := searcher.SearchNews(ctx, query, news.SearchOptions{
		NumResults: newsLimit,
		Language:   newsLanguage,
		Country:    newsCountry,
	})
	if err != nil {
		return fmt.Errorf("searching news: %w", err)
	}

	if newsJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(articles)
	}

	logger.Info("news results", "count", len(articles))
	for i, a := range articles {
		logger.Info(fmt.Sprintf("article %d", i+1),
			"title", truncate(a.Title, 60),
			"source", a.Source,
			"date", a.Date,
		)
		fmt.Printf("   %s\n", a.Link)
		if a.Snippet != "" {
			fmt.Printf("   %s\n", truncate(a.Snippet, 100))
		}
		fmt.Println()
	}

	return nil
}

func runSearch(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	query := strings.Join(args, " ")

	searcher, err := news.NewSearcher(news.SearcherConfig{
		Engine: searchEngine,
	})
	if err != nil {
		return fmt.Errorf("creating searcher: %w", err)
	}

	logger.Info("searching web", "query", query)

	result, err := searcher.SearchWeb(ctx, query, news.SearchOptions{
		NumResults: searchLimit,
		Language:   searchLanguage,
		Country:    searchCountry,
	})
	if err != nil {
		return fmt.Errorf("searching web: %w", err)
	}

	if searchJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(result)
	}

	// Show answer box if available
	if result.AnswerBox != nil {
		fmt.Println("=== Answer Box ===")
		if result.AnswerBox.Title != "" {
			fmt.Printf("Title: %s\n", result.AnswerBox.Title)
		}
		if result.AnswerBox.Answer != "" {
			fmt.Printf("Answer: %s\n", result.AnswerBox.Answer)
		}
		if result.AnswerBox.Snippet != "" {
			fmt.Printf("Snippet: %s\n", result.AnswerBox.Snippet)
		}
		fmt.Println()
	}

	logger.Info("web results", "count", len(result.OrganicResults))
	for i, r := range result.OrganicResults {
		logger.Info(fmt.Sprintf("result %d", i+1),
			"title", truncate(r.Title, 60),
			"domain", r.Domain,
		)
		fmt.Printf("   %s\n", r.Link)
		if r.Snippet != "" {
			fmt.Printf("   %s\n", truncate(r.Snippet, 100))
		}
		fmt.Println()
	}

	return nil
}
