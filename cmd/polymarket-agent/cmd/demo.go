package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/grokify/polymarket-go/internal/polymarket"
	"github.com/spf13/cobra"
)

var demoCmd = &cobra.Command{
	Use:   "demo",
	Short: "Run demo fetching live Polymarket data",
	Long: `Fetches live market data from Polymarket and displays
high-liquidity markets sorted by liquidity.`,
	RunE: runDemo,
}

var demoLimit int

func init() {
	rootCmd.AddCommand(demoCmd)
	demoCmd.Flags().IntVarP(&demoLimit, "limit", "l", 5, "Number of markets to display")
}

func runDemo(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	logger.Info("fetching live Polymarket data")

	client := polymarket.NewClient()

	// Fetch active markets
	active := true
	markets, err := client.GetMarkets(ctx, polymarket.GetMarketsParams{
		Active: &active,
		Limit:  100,
	})
	if err != nil {
		return fmt.Errorf("fetching markets: %w", err)
	}

	// Filter to high-liquidity markets
	var filtered []polymarket.Market
	for _, m := range markets {
		if m.LiquidityNum > 10000 { // At least $10k
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
	if len(filtered) > demoLimit {
		filtered = filtered[:demoLimit]
	}

	logger.Info("fetched markets", "total", len(markets), "filtered", len(filtered))

	for i, m := range filtered {
		logger.Info(fmt.Sprintf("market %d", i+1),
			"question", truncate(m.Question, 60),
			"liquidity", fmt.Sprintf("$%.0f", m.LiquidityNum),
			"volume_24h", fmt.Sprintf("$%.0f", m.Volume24hr),
			"best_bid", m.BestBid,
			"best_ask", m.BestAsk,
			"spread", m.Spread,
			"end_date", m.EndDateISO,
		)
	}

	return nil
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}
