package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/grokify/polymarket-go/internal/polymarket"
	"github.com/grokify/polymarket-go/internal/prompts"
	"github.com/plexusone/omnillm-core"
	"github.com/spf13/cobra"
)

var marketsCmd = &cobra.Command{
	Use:   "markets",
	Short: "Market operations",
	Long:  `Commands for listing, filtering, and analyzing Polymarket markets.`,
}

var marketsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List markets with optional filters",
	Long: `Fetches and displays markets from Polymarket.
Supports filtering by activity status and minimum liquidity.`,
	RunE: runMarketsList,
}

var marketsAnalyzeCmd = &cobra.Command{
	Use:   "analyze",
	Short: "Analyze markets using superforecaster",
	Long: `Uses AI superforecaster methodology to analyze market probabilities.
Requires ANTHROPIC_API_KEY environment variable.`,
	RunE: runMarketsAnalyze,
}

var (
	marketsListLimit       int
	marketsListMinLiq      float64
	marketsListActive      bool
	marketsListJSON        bool
	marketsAnalyzeLimit    int
	marketsAnalyzeMinLiq   float64
	marketsAnalyzeQuestion string
)

func init() {
	rootCmd.AddCommand(marketsCmd)
	marketsCmd.AddCommand(marketsListCmd)
	marketsCmd.AddCommand(marketsAnalyzeCmd)

	// List flags
	marketsListCmd.Flags().IntVarP(&marketsListLimit, "limit", "l", 10, "Maximum number of markets to display")
	marketsListCmd.Flags().Float64Var(&marketsListMinLiq, "min-liquidity", 0, "Minimum liquidity filter (USD)")
	marketsListCmd.Flags().BoolVar(&marketsListActive, "active", true, "Filter to active markets only")
	marketsListCmd.Flags().BoolVar(&marketsListJSON, "json", false, "Output as JSON")

	// Analyze flags
	marketsAnalyzeCmd.Flags().IntVarP(&marketsAnalyzeLimit, "limit", "l", 1, "Number of markets to analyze")
	marketsAnalyzeCmd.Flags().Float64Var(&marketsAnalyzeMinLiq, "min-liquidity", 50000, "Minimum liquidity filter (USD)")
	marketsAnalyzeCmd.Flags().StringVarP(&marketsAnalyzeQuestion, "question", "q", "", "Analyze a specific market by question text (partial match)")
}

func runMarketsList(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	client := polymarket.NewClient()

	// Fetch markets
	params := polymarket.GetMarketsParams{
		Limit: 100,
	}
	if marketsListActive {
		params.Active = &marketsListActive
	}

	markets, err := client.GetMarkets(ctx, params)
	if err != nil {
		return fmt.Errorf("fetching markets: %w", err)
	}

	// Filter by minimum liquidity
	var filtered []polymarket.Market
	for _, m := range markets {
		if m.LiquidityNum >= marketsListMinLiq {
			filtered = append(filtered, m)
		}
	}

	// Sort by liquidity descending
	for i := 0; i < len(filtered); i++ {
		for j := i + 1; j < len(filtered); j++ {
			if filtered[j].LiquidityNum > filtered[i].LiquidityNum {
				filtered[i], filtered[j] = filtered[j], filtered[i]
			}
		}
	}

	// Limit results
	if len(filtered) > marketsListLimit {
		filtered = filtered[:marketsListLimit]
	}

	if marketsListJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(filtered)
	}

	logger.Info("markets", "total", len(markets), "filtered", len(filtered))

	for i, m := range filtered {
		logger.Info(fmt.Sprintf("market %d", i+1),
			"question", truncate(m.Question, 50),
			"liquidity", fmt.Sprintf("$%.0f", m.LiquidityNum),
			"volume_24h", fmt.Sprintf("$%.0f", m.Volume24hr),
			"bid/ask", fmt.Sprintf("%.2f/%.2f", m.BestBid, m.BestAsk),
		)
	}

	return nil
}

func runMarketsAnalyze(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	// Check for API key
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		return fmt.Errorf("ANTHROPIC_API_KEY environment variable is required")
	}

	// Get model from root flag
	modelName, _ := cmd.Flags().GetString("model")
	if modelName == "" {
		modelName, _ = cmd.Root().PersistentFlags().GetString("model")
	}

	// Create omnillm client
	llmClient, err := omnillm.NewClient(omnillm.ClientConfig{
		Providers: []omnillm.ProviderConfig{
			{Provider: omnillm.ProviderNameAnthropic, APIKey: apiKey},
		},
		Logger: logger,
	})
	if err != nil {
		return fmt.Errorf("creating LLM client: %w", err)
	}
	defer llmClient.Close()

	// Create agent
	agent := prompts.NewAgent(prompts.AgentConfig{
		LLM:   llmClient.Provider(),
		Model: modelName,
	})

	// Fetch markets
	pmClient := polymarket.NewClient()
	active := true
	markets, err := pmClient.GetMarkets(ctx, polymarket.GetMarketsParams{
		Active: &active,
		Limit:  100,
	})
	if err != nil {
		return fmt.Errorf("fetching markets: %w", err)
	}

	// Filter markets
	var filtered []polymarket.Market
	for _, m := range markets {
		if m.LiquidityNum < marketsAnalyzeMinLiq {
			continue
		}
		if marketsAnalyzeQuestion != "" && !strings.Contains(strings.ToLower(m.Question), strings.ToLower(marketsAnalyzeQuestion)) {
			continue
		}
		filtered = append(filtered, m)
	}

	// Sort by liquidity descending
	for i := 0; i < len(filtered); i++ {
		for j := i + 1; j < len(filtered); j++ {
			if filtered[j].LiquidityNum > filtered[i].LiquidityNum {
				filtered[i], filtered[j] = filtered[j], filtered[i]
			}
		}
	}

	if len(filtered) > marketsAnalyzeLimit {
		filtered = filtered[:marketsAnalyzeLimit]
	}

	if len(filtered) == 0 {
		logger.Info("no markets found matching criteria")
		return nil
	}

	logger.Info("analyzing markets", "count", len(filtered))

	for i, m := range filtered {
		logger.Info(fmt.Sprintf("analyzing market %d/%d", i+1, len(filtered)),
			"question", truncate(m.Question, 50),
			"liquidity", fmt.Sprintf("$%.0f", m.LiquidityNum),
		)

		// Parse outcomes
		outcomes := "Yes, No"
		if m.Outcomes != "" {
			outcomes = strings.Trim(m.Outcomes, "[]\"")
			outcomes = strings.ReplaceAll(outcomes, "\",\"", ", ")
		}

		forecast, err := agent.GetSuperforecast(ctx, m.Description, m.Question, outcomes)
		if err != nil {
			logger.Error("forecast failed", "error", err)
			continue
		}

		// Print forecast summary
		lines := strings.Split(forecast, "\n")
		var summary []string
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line != "" && len(summary) < 5 {
				summary = append(summary, line)
			}
		}

		logger.Info("forecast",
			"market", truncate(m.Question, 40),
			"summary", strings.Join(summary, " | "),
		)

		// Print full forecast
		fmt.Println("\n--- Full Forecast ---")
		fmt.Println(forecast)
		fmt.Println("---------------------")
	}

	return nil
}
