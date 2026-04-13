package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/grokify/polymarket-go/internal/polymarket"
	"github.com/grokify/polymarket-go/internal/prompts"
	"github.com/plexusone/omnillm-core"
	"github.com/spf13/cobra"
)

var tradeCmd = &cobra.Command{
	Use:   "trade",
	Short: "Trading operations",
	Long:  `Commands for executing trades and running autonomous trading strategies.`,
}

var tradeAutoCmd = &cobra.Command{
	Use:   "auto",
	Short: "Run autonomous trading loop",
	Long: `Runs an autonomous trading loop that:
1. Fetches high-liquidity markets
2. Analyzes them using superforecaster methodology
3. Identifies trading opportunities based on edge
4. Optionally executes trades (requires --execute flag)

Requires environment variables:
- ANTHROPIC_API_KEY: For LLM analysis
- POLYGON_WALLET_PRIVATE_KEY: For trading (if --execute)
- POLYMARKET_API_KEY, POLYMARKET_API_SECRET, POLYMARKET_API_PASSPHRASE: For trading (if --execute)`,
	RunE: runTradeAuto,
}

var tradeRecommendCmd = &cobra.Command{
	Use:   "recommend",
	Short: "Get a single trade recommendation",
	Long: `Analyzes markets and returns a single best trade recommendation
without executing it.`,
	RunE: runTradeRecommend,
}

var (
	tradeAutoInterval  time.Duration
	tradeAutoExecute   bool
	tradeAutoMinLiq    float64
	tradeAutoMaxTrades int
	tradeRecMinLiq     float64
)

func init() {
	rootCmd.AddCommand(tradeCmd)
	tradeCmd.AddCommand(tradeAutoCmd)
	tradeCmd.AddCommand(tradeRecommendCmd)

	// Auto trading flags
	tradeAutoCmd.Flags().DurationVarP(&tradeAutoInterval, "interval", "i", 1*time.Hour, "Interval between trading cycles")
	tradeAutoCmd.Flags().BoolVar(&tradeAutoExecute, "execute", false, "Actually execute trades (default: dry run)")
	tradeAutoCmd.Flags().Float64Var(&tradeAutoMinLiq, "min-liquidity", 50000, "Minimum liquidity for markets (USD)")
	tradeAutoCmd.Flags().IntVar(&tradeAutoMaxTrades, "max-trades", 1, "Maximum trades per cycle")

	// Recommend flags
	tradeRecommendCmd.Flags().Float64Var(&tradeRecMinLiq, "min-liquidity", 50000, "Minimum liquidity for markets (USD)")
}

func runTradeAuto(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	// Check for API key
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		return fmt.Errorf("ANTHROPIC_API_KEY environment variable is required")
	}

	// Get model from root flag
	modelName, _ := cmd.Root().PersistentFlags().GetString("model")

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

	// Create Polymarket client
	pmClient := polymarket.NewClient()

	// Check if we can execute trades
	var sdkClient *polymarket.SDKClient
	if tradeAutoExecute {
		sdkClient, err = polymarket.NewSDKClient(polymarket.SDKConfig{})
		if err != nil {
			return fmt.Errorf("creating SDK client for trading: %w", err)
		}
		if !sdkClient.IsAuthenticated() {
			return fmt.Errorf("SDK client not authenticated - check POLYGON_WALLET_PRIVATE_KEY and API credentials")
		}
		logger.Info("trading mode: LIVE (trades will be executed)")
	} else {
		logger.Info("trading mode: DRY RUN (no trades will be executed)")
	}

	// Handle graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	logger.Info("starting autonomous trading loop", "interval", tradeAutoInterval)

	ticker := time.NewTicker(tradeAutoInterval)
	defer ticker.Stop()

	// Run immediately, then on interval
	runTradingCycle(ctx, logger, agent, pmClient, sdkClient)

	for {
		select {
		case <-ticker.C:
			runTradingCycle(ctx, logger, agent, pmClient, sdkClient)
		case <-sigCh:
			logger.Info("received shutdown signal, stopping...")
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func runTradingCycle(ctx context.Context, logger *slog.Logger, agent *prompts.Agent, pmClient *polymarket.Client, sdkClient *polymarket.SDKClient) {
	logger.Info("starting trading cycle")

	// Fetch markets
	active := true
	markets, err := pmClient.GetMarkets(ctx, polymarket.GetMarketsParams{
		Active: &active,
		Limit:  100,
	})
	if err != nil {
		logger.Error("failed to fetch markets", "error", err)
		return
	}

	// Filter by liquidity
	var filtered []polymarket.Market
	for _, m := range markets {
		if m.LiquidityNum >= tradeAutoMinLiq {
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

	// Limit to top markets for analysis
	if len(filtered) > 10 {
		filtered = filtered[:10]
	}

	logger.Info("analyzing markets", "count", len(filtered))

	tradesExecuted := 0
	for _, m := range filtered {
		if tradesExecuted >= tradeAutoMaxTrades {
			break
		}

		// Convert to MarketInfo for agent
		marketInfo := prompts.MarketInfo{
			ID:          m.ConditionID,
			Question:    m.Question,
			Description: m.Description,
			Outcomes:    parseOutcomes(m.Outcomes),
			Liquidity:   m.LiquidityNum,
			Volume:      m.Volume24hr,
		}

		// Parse outcome prices
		if m.OutcomePrices != "" {
			prices := strings.Trim(m.OutcomePrices, "[]")
			for _, p := range strings.Split(prices, ",") {
				var price float64
				_, _ = fmt.Sscanf(strings.TrimSpace(p), "%f", &price)
				marketInfo.OutcomePrices = append(marketInfo.OutcomePrices, price)
			}
		}

		// Get trade recommendation
		rec, err := agent.SourceBestTrade(ctx, marketInfo)
		if err != nil {
			logger.Error("trade analysis failed", "market", truncate(m.Question, 40), "error", err)
			continue
		}

		logger.Info("trade recommendation",
			"market", truncate(m.Question, 40),
			"side", rec.Side,
			"price", fmt.Sprintf("%.2f", rec.Price),
			"size", fmt.Sprintf("%.1f%%", rec.Size*100),
		)

		// Execute trade if enabled
		if sdkClient != nil && tradeAutoExecute {
			logger.Info("trade execution not yet implemented - would execute here")
			// TODO: Implement actual trade execution
			// This would use sdkClient.PlaceOrder()
		}

		tradesExecuted++
	}

	logger.Info("trading cycle complete", "trades_analyzed", tradesExecuted)
}

func runTradeRecommend(cmd *cobra.Command, args []string) error {
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
	modelName, _ := cmd.Root().PersistentFlags().GetString("model")

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

	// Filter by liquidity and sort
	var filtered []polymarket.Market
	for _, m := range markets {
		if m.LiquidityNum >= tradeRecMinLiq {
			filtered = append(filtered, m)
		}
	}

	for i := 0; i < len(filtered); i++ {
		for j := i + 1; j < len(filtered); j++ {
			if filtered[j].LiquidityNum > filtered[i].LiquidityNum {
				filtered[i], filtered[j] = filtered[j], filtered[i]
			}
		}
	}

	if len(filtered) == 0 {
		logger.Info("no markets found matching criteria")
		return nil
	}

	// Analyze top market
	m := filtered[0]
	logger.Info("analyzing top market",
		"question", truncate(m.Question, 50),
		"liquidity", fmt.Sprintf("$%.0f", m.LiquidityNum),
	)

	// Convert to MarketInfo
	marketInfo := prompts.MarketInfo{
		ID:          m.ConditionID,
		Question:    m.Question,
		Description: m.Description,
		Outcomes:    parseOutcomes(m.Outcomes),
		Liquidity:   m.LiquidityNum,
		Volume:      m.Volume24hr,
	}

	if m.OutcomePrices != "" {
		prices := strings.Trim(m.OutcomePrices, "[]")
		for _, p := range strings.Split(prices, ",") {
			var price float64
			_, _ = fmt.Sscanf(strings.TrimSpace(p), "%f", &price)
			marketInfo.OutcomePrices = append(marketInfo.OutcomePrices, price)
		}
	}

	rec, err := agent.SourceBestTrade(ctx, marketInfo)
	if err != nil {
		return fmt.Errorf("trade analysis failed: %w", err)
	}

	fmt.Println("\n=== Trade Recommendation ===")
	fmt.Printf("Market: %s\n", m.Question)
	fmt.Printf("Side: %s\n", rec.Side)
	fmt.Printf("Price: %.4f\n", rec.Price)
	fmt.Printf("Size: %.1f%% of bankroll\n", rec.Size*100)
	fmt.Println("\n--- Forecast ---")
	fmt.Println(rec.Forecast)
	fmt.Println("================")

	return nil
}

func parseOutcomes(outcomes string) []string {
	if outcomes == "" {
		return []string{"Yes", "No"}
	}
	// Parse JSON array string like ["Yes","No"]
	outcomes = strings.Trim(outcomes, "[]")
	outcomes = strings.ReplaceAll(outcomes, "\"", "")
	return strings.Split(outcomes, ",")
}
