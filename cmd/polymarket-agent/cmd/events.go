package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"

	"github.com/grokify/polymarket-go/internal/polymarket"
	"github.com/spf13/cobra"
)

var eventsCmd = &cobra.Command{
	Use:   "events",
	Short: "Event operations",
	Long:  `Commands for listing and filtering Polymarket events.`,
}

var eventsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List events with optional filters",
	Long: `Fetches and displays events from Polymarket.
Events group related markets together.`,
	RunE: runEventsList,
}

var (
	eventsListLimit  int
	eventsListActive bool
	eventsListJSON   bool
)

func init() {
	rootCmd.AddCommand(eventsCmd)
	eventsCmd.AddCommand(eventsListCmd)

	eventsListCmd.Flags().IntVarP(&eventsListLimit, "limit", "l", 10, "Maximum number of events to display")
	eventsListCmd.Flags().BoolVar(&eventsListActive, "active", true, "Filter to active events only")
	eventsListCmd.Flags().BoolVar(&eventsListJSON, "json", false, "Output as JSON")
}

func runEventsList(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	// Use SDK client for Gamma API access
	sdkClient, err := polymarket.NewSDKClient(polymarket.SDKConfig{})
	if err != nil {
		return fmt.Errorf("creating SDK client: %w", err)
	}

	events, err := sdkClient.GetEventsGamma(ctx)
	if err != nil {
		return fmt.Errorf("fetching events: %w", err)
	}

	// Limit results
	if len(events) > eventsListLimit {
		events = events[:eventsListLimit]
	}

	if eventsListJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(events)
	}

	logger.Info("events", "count", len(events))

	for i, e := range events {
		marketCount := 0
		if e.Markets != nil {
			marketCount = len(e.Markets)
		}
		logger.Info(fmt.Sprintf("event %d", i+1),
			"title", truncate(e.Title, 50),
			"slug", e.Slug,
			"markets", marketCount,
			"volume", e.Volume,
			"liquidity", e.Liquidity,
		)
	}

	return nil
}
