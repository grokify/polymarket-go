// Package cmd provides the CLI commands for polymarket-agent.
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "polymarket-agent",
	Short: "AI trading agents for Polymarket prediction markets",
	Long: `polymarket-agent is a CLI for running AI-powered trading agents
on Polymarket prediction markets.

It provides tools for:
- Fetching and analyzing market data
- Running superforecaster probability estimation
- Executing autonomous trading strategies
- Managing positions and orders`,
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	// Global flags can be added here
	rootCmd.PersistentFlags().StringP("model", "m", "claude-sonnet-4-20250514", "LLM model to use")
}
