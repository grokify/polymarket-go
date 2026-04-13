// Command polymarket-agent runs Polymarket trading agents.
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/grokify/polymarket-go/internal/executor"
	"github.com/grokify/polymarket-go/internal/loader"
	"github.com/grokify/polymarket-go/internal/polymarket"
	"github.com/grokify/polymarket-go/internal/prompts"
	"github.com/plexusone/omnillm-core"
)

var (
	demo         = flag.Bool("demo", false, "Run demo fetching live Polymarket data")
	demoLimit    = flag.Int("demo-limit", 5, "Number of markets to fetch in demo mode")
	analyze      = flag.Bool("analyze", false, "Analyze markets using superforecaster (requires ANTHROPIC_API_KEY)")
	analyzeLimit = flag.Int("analyze-limit", 1, "Number of markets to analyze")
	model        = flag.String("model", "claude-sonnet-4-20250514", "Model to use for analysis")
)

func main() {
	flag.Parse()

	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	// Load specs
	l := loader.NewLoader()

	logger.Info("loading agents", "dir", "agents/specs/agents")
	agents, err := l.LoadAgentsFromDir("agents/specs/agents")
	if err != nil {
		return fmt.Errorf("loading agents: %w", err)
	}
	logger.Info("loaded agents", "count", len(agents))

	for _, agent := range agents {
		logger.Info("agent",
			"name", agent.QualifiedName(),
			"model", agent.Model,
			"tools", agent.Tools,
		)
	}

	// Load team
	logger.Info("loading team", "file", "agents/specs/team.json")
	team, err := l.LoadTeam("agents/specs/team.json")
	if err != nil {
		return fmt.Errorf("loading team: %w", err)
	}
	logger.Info("loaded team",
		"name", team.Name,
		"version", team.Version,
		"workflow_type", team.Workflow.Type,
		"steps", len(team.Workflow.Steps),
	)

	// Load deployment
	logger.Info("loading deployment", "file", "agents/specs/deployment-go-server.json")
	deployment, err := l.LoadDeployment("agents/specs/deployment-go-server.json")
	if err != nil {
		return fmt.Errorf("loading deployment: %w", err)
	}
	logger.Info("loaded deployment",
		"platform", deployment.Platform,
		"team", deployment.Team,
	)

	// Create agent map
	agentMap := loader.AgentMap(agents)

	// Convert to executor specs
	agentSpecs := make(map[string]executor.AgentSpec)
	for name, agent := range agentMap {
		agentSpecs[name] = executor.AgentSpec{
			Name:         agent.Name,
			Instructions: agent.Instructions,
			Model:        agent.Model,
			Tools:        agent.Tools,
		}
	}

	// TODO: Initialize omnillm client
	// client := omnillm.NewClient(omnillm.ClientConfig{...})

	// For now, just print workflow
	logger.Info("workflow steps")
	for i, step := range team.Workflow.Steps {
		deps := "none"
		if len(step.DependsOn) > 0 {
			deps = fmt.Sprintf("%v", step.DependsOn)
		}
		logger.Info(fmt.Sprintf("step %d", i+1),
			"name", step.Name,
			"agent", step.Agent,
			"depends_on", deps,
		)
	}

	logger.Info("ready to execute",
		"note", "omnillm integration pending",
	)

	// Demo mode: fetch live market data
	if *demo {
		logger.Info("running demo: fetching live Polymarket data")
		if err := runDemo(ctx, logger, *demoLimit); err != nil {
			return fmt.Errorf("demo: %w", err)
		}
	}

	// Analyze mode: use superforecaster to analyze markets
	if *analyze {
		logger.Info("running analysis: superforecaster prediction")
		if err := runAnalyze(ctx, logger, *analyzeLimit, *model); err != nil {
			return fmt.Errorf("analyze: %w", err)
		}
	}

	_ = agentSpecs

	return nil
}

func runDemo(ctx context.Context, logger *slog.Logger, limit int) error {
	client := polymarket.NewClient()

	// Fetch active markets - get more than we need and filter/sort locally
	active := true
	markets, err := client.GetMarkets(ctx, polymarket.GetMarketsParams{
		Active: &active,
		Limit:  100, // Fetch more to find high-liquidity ones
	})
	if err != nil {
		return fmt.Errorf("fetching markets: %w", err)
	}

	// Filter and sort by liquidity descending
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
	if len(filtered) > limit {
		filtered = filtered[:limit]
	}

	logger.Info("fetched markets from Polymarket", "total", len(markets), "filtered", len(filtered))

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

func runAnalyze(ctx context.Context, logger *slog.Logger, limit int, modelName string) error {
	// Check for API key
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		return fmt.Errorf("ANTHROPIC_API_KEY environment variable is required for analysis")
	}

	// Create omnillm client
	client, err := omnillm.NewClient(omnillm.ClientConfig{
		Providers: []omnillm.ProviderConfig{
			{Provider: omnillm.ProviderNameAnthropic, APIKey: apiKey},
		},
		Logger: logger,
	})
	if err != nil {
		return fmt.Errorf("creating omnillm client: %w", err)
	}
	defer client.Close()

	// Create agent using the underlying provider
	agent := prompts.NewAgent(prompts.AgentConfig{
		LLM:   client.Provider(),
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

	// Filter to high-liquidity markets
	var filtered []polymarket.Market
	for _, m := range markets {
		if m.LiquidityNum > 50000 { // At least $50k liquidity
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

	if len(filtered) > limit {
		filtered = filtered[:limit]
	}

	logger.Info("analyzing markets", "count", len(filtered))

	for i, m := range filtered {
		logger.Info(fmt.Sprintf("analyzing market %d/%d", i+1, len(filtered)),
			"question", truncate(m.Question, 60),
			"liquidity", fmt.Sprintf("$%.0f", m.LiquidityNum),
		)

		// Get superforecast
		outcomes := "Yes, No" // Default for binary markets
		if m.Outcomes != "" {
			// Parse outcomes JSON array if available
			outcomes = strings.Trim(m.Outcomes, "[]\"")
			outcomes = strings.ReplaceAll(outcomes, "\",\"", ", ")
		}

		forecast, err := agent.GetSuperforecast(ctx, m.Description, m.Question, outcomes)
		if err != nil {
			logger.Error("forecast failed", "error", err)
			continue
		}

		// Print forecast (truncated for readability)
		lines := strings.Split(forecast, "\n")
		var summary []string
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line != "" && len(summary) < 5 {
				summary = append(summary, line)
			}
		}

		logger.Info("forecast result",
			"market", truncate(m.Question, 40),
			"summary", strings.Join(summary, " | "),
		)
	}

	return nil
}
