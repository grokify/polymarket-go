# Quick Start

This guide will get you fetching live Polymarket data in under 5 minutes.

## Run the Demo

The fastest way to see polymarket-go in action:

```bash
go run ./cmd/polymarket-agent/ --demo --demo-limit=10
```

This fetches the top 10 markets by liquidity from Polymarket:

```
level=INFO msg="loading agents" dir=agents/specs/agents
level=INFO msg="loaded agents" count=3
level=INFO msg="running demo: fetching live Polymarket data"
level=INFO msg="fetched markets from Polymarket" total=100 filtered=10
level=INFO msg="market 1" question="Will Panama win the 2026 FIFA World Cup?" liquidity=$4007373 ...
level=INFO msg="market 2" question="Will Haiti win the 2026 FIFA World Cup?" liquidity=$3860101 ...
```

## Use the Polymarket Client

### Fetch Markets

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/grokify/polymarket-go/internal/polymarket"
)

func main() {
    client := polymarket.NewClient()
    ctx := context.Background()

    active := true
    markets, err := client.GetMarkets(ctx, polymarket.GetMarketsParams{
        Active: &active,
        Limit:  5,
        Order:  "liquidity",
    })
    if err != nil {
        log.Fatal(err)
    }

    for _, m := range markets {
        fmt.Printf("%s - $%.0f liquidity\n", m.Question, m.LiquidityNum)
    }
}
```

### Get Order Book

```go
book, err := client.GetOrderBook(ctx, tokenID)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Bids: %d, Asks: %d\n", len(book.Bids), len(book.Asks))
```

### Get Price Data

```go
// Current price
price, _ := client.GetPrice(ctx, tokenID)

// Mid price
mid, _ := client.GetMidPrice(ctx, tokenID)

// Bid-ask spread
spread, _ := client.GetSpread(ctx, tokenID)
```

## Load Agent Specs

```go
package main

import (
    "fmt"
    "log"

    "github.com/grokify/polymarket-go/internal/loader"
)

func main() {
    l := loader.NewLoader()

    // Load all agents from directory
    agents, err := l.LoadAgentsFromDir("agents/specs/agents")
    if err != nil {
        log.Fatal(err)
    }

    for _, agent := range agents {
        fmt.Printf("Agent: %s, Model: %s, Tools: %v\n",
            agent.Name, agent.Model, agent.Tools)
    }

    // Load team configuration
    team, err := l.LoadTeam("agents/specs/team.json")
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Team: %s, Workflow: %s, Steps: %d\n",
        team.Name, team.Workflow.Type, len(team.Workflow.Steps))
}
```

## Next Steps

- [CLI Usage](cli.md) - Full CLI reference
- [Agent Specs](../agents/index.md) - Understand agent definitions
- [API Reference](../api/polymarket.md) - Complete API documentation
