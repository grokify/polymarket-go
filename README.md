# polymarket-go

[![Go CI][go-ci-svg]][go-ci-url]
[![Go Lint][go-lint-svg]][go-lint-url]
[![Go SAST][go-sast-svg]][go-sast-url]
[![Go Report Card][goreport-svg]][goreport-url]
[![Docs][docs-godoc-svg]][docs-godoc-url]
[![Visualization][viz-svg]][viz-url]
[![License][license-svg]][license-url]

 [go-ci-svg]: https://github.com/grokify/polymarket-go/actions/workflows/go-ci.yaml/badge.svg?branch=main
 [go-ci-url]: https://github.com/grokify/polymarket-go/actions/workflows/go-ci.yaml
 [go-lint-svg]: https://github.com/grokify/polymarket-go/actions/workflows/go-lint.yaml/badge.svg?branch=main
 [go-lint-url]: https://github.com/grokify/polymarket-go/actions/workflows/go-lint.yaml
 [go-sast-svg]: https://github.com/grokify/polymarket-go/actions/workflows/go-sast-codeql.yaml/badge.svg?branch=main
 [go-sast-url]: https://github.com/grokify/polymarket-go/actions/workflows/go-sast-codeql.yaml
 [goreport-svg]: https://goreportcard.com/badge/github.com/grokify/polymarket-go
 [goreport-url]: https://goreportcard.com/report/github.com/grokify/polymarket-go
 [docs-godoc-svg]: https://pkg.go.dev/badge/github.com/grokify/polymarket-go
 [docs-godoc-url]: https://pkg.go.dev/github.com/grokify/polymarket-go
 [viz-svg]: https://img.shields.io/badge/visualizaton-Go-blue.svg
 [viz-url]: https://mango-dune-07a8b7110.1.azurestaticapps.net/?repo=grokify%2Fpolymarket-go
 [loc-svg]: https://tokei.rs/b1/github/grokify/polymarket-go
 [repo-url]: https://github.com/grokify/polymarket-go
 [license-svg]: https://img.shields.io/badge/license-MIT-blue.svg
 [license-url]: https://github.com/grokify/polymarket-go/blob/master/LICENSE

Go SDK for building AI trading agents on [Polymarket](https://polymarket.com) prediction markets.

## Features

- **Polymarket API Client** - Full client for Gamma (markets) and CLOB (trading) APIs
- **Multi-Agent Workflows** - Define agent teams using [multi-agent-spec](https://github.com/plexusone/multi-agent-spec) format
- **LLM Integration** - Works with any LLM via [omnillm](https://github.com/plexusone/omnillm) + [LangChainGo](https://github.com/tmc/langchaingo)
- **Portable Specs** - Same agent definitions deploy to Claude Code, Go servers, or Kubernetes

## Installation

```bash
go get github.com/grokify/polymarket-go
```

## Quick Start

### Run the Demo

Fetch live market data from Polymarket:

```bash
go run ./cmd/polymarket-agent/ --demo --demo-limit=10
```

Output:
```
level=INFO msg="market 1" question="Will Panama win the 2026 FIFA World Cup?" liquidity=$4007373 ...
level=INFO msg="market 2" question="Will Haiti win the 2026 FIFA World Cup?" liquidity=$3860101 ...
```

### Use the Polymarket Client

```go
import "github.com/grokify/polymarket-go/internal/polymarket"

client := polymarket.NewClient()

// Fetch active markets
markets, err := client.GetMarkets(ctx, polymarket.GetMarketsParams{
    Active: &active,
    Limit:  10,
    Order:  "liquidity",
})

// Get order book for a token
book, err := client.GetOrderBook(ctx, tokenID)
```

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│  polymarket-go                                              │
├─────────────────────────────────────────────────────────────┤
│  cmd/polymarket-agent/     CLI for running agent workflows  │
├─────────────────────────────────────────────────────────────┤
│  agents/specs/             Multi-agent-spec definitions     │
│  ├── agents/               Agent markdown files             │
│  ├── team.json             Workflow configuration           │
│  └── deployment-*.json     Platform-specific configs        │
├─────────────────────────────────────────────────────────────┤
│  internal/                                                  │
│  ├── polymarket/           Polymarket API client            │
│  ├── loader/               Spec file parsers                │
│  ├── executor/             Workflow execution engine        │
│  └── tools/                Agent tools for Polymarket       │
├─────────────────────────────────────────────────────────────┤
│  omnillm-langchaingo       LangChainGo adapter              │
├─────────────────────────────────────────────────────────────┤
│  omnillm-core              Unified LLM provider abstraction │
└─────────────────────────────────────────────────────────────┘
```

## Agent Team

The default trading team consists of three agents in a graph workflow:

| Agent | Role | Model |
|-------|------|-------|
| **market-analyst** | Discovers trading opportunities | sonnet |
| **superforecaster** | Generates probability estimates | sonnet |
| **trader** | Executes trades based on analysis | haiku |

Workflow: `discover → forecast → execute`

## Agent Specs

Agents are defined in markdown with YAML frontmatter:

```markdown
---
name: market-analyst
model: sonnet
tools: [WebSearch, WebFetch, Read, Write]
role: Market Research Analyst
goal: Identify mispriced markets with >10% expected edge
---

You are a market analyst specializing in Polymarket...
```

## Deployment Targets

The same agent specs can deploy to multiple platforms:

| Platform | Config File | Use Case |
|----------|-------------|----------|
| Go Server | `deployment-go-server.json` | Production trading |
| Claude Code | `deployment-claude-code.json` | Development/testing |

## Dependencies

- [omnillm-core](https://github.com/plexusone/omnillm-core) - LLM provider abstraction
- [omnillm-langchaingo](https://github.com/plexusone/omnillm-langchaingo) - LangChainGo adapter
- [langchaingo](https://github.com/tmc/langchaingo) - Go LLM framework

## Documentation

- [Technical Requirements Document](docs/design/TRD.md)
- [Component Integration Guide](docs/design/COMPONENTS.md)

## License

MIT License - see [LICENSE](LICENSE) for details.
