# polymarket-go

Go SDK for building AI trading agents on [Polymarket](https://polymarket.com) prediction markets.

## Features

- **Polymarket API Client** - Full client for Gamma (markets) and CLOB (trading) APIs
- **Multi-Agent Workflows** - Define agent teams using [multi-agent-spec](https://github.com/plexusone/multi-agent-spec) format
- **LLM Integration** - Works with any LLM via [omnillm](https://github.com/plexusone/omnillm) + [LangChainGo](https://github.com/tmc/langchaingo)
- **Portable Specs** - Same agent definitions deploy to Claude Code, Go servers, or Kubernetes

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│  polymarket-go                                               │
├─────────────────────────────────────────────────────────────┤
│  cmd/polymarket-agent/     CLI for running agent workflows   │
├─────────────────────────────────────────────────────────────┤
│  agents/specs/             Multi-agent-spec definitions      │
│  ├── agents/               Agent markdown files              │
│  ├── team.json             Workflow configuration            │
│  └── deployment-*.json     Platform-specific configs         │
├─────────────────────────────────────────────────────────────┤
│  internal/                                                   │
│  ├── polymarket/           Polymarket API client             │
│  ├── loader/               Spec file parsers                 │
│  ├── executor/             Workflow execution engine         │
│  └── tools/                Agent tools for Polymarket        │
├─────────────────────────────────────────────────────────────┤
│  omnillm-langchaingo       LangChainGo adapter               │
├─────────────────────────────────────────────────────────────┤
│  omnillm-core              Unified LLM provider abstraction  │
└─────────────────────────────────────────────────────────────┘
```

## Agent Team

The default trading team consists of three agents in a graph workflow:

| Agent | Role | Model |
|-------|------|-------|
| [**market-analyst**](agents/market-analyst.md) | Discovers trading opportunities | sonnet |
| [**superforecaster**](agents/superforecaster.md) | Generates probability estimates | sonnet |
| [**trader**](agents/trader.md) | Executes trades based on analysis | haiku |

**Workflow:** `discover → forecast → execute`

## Quick Links

- [Installation](getting-started/installation.md)
- [Quick Start](getting-started/quickstart.md)
- [Technical Requirements](design/TRD.md)
- [API Reference](api/polymarket.md)
- [Changelog](releases/CHANGELOG.md)

## Dependencies

| Package | Purpose |
|---------|---------|
| [omnillm-core](https://github.com/plexusone/omnillm-core) | LLM provider abstraction |
| [omnillm-langchaingo](https://github.com/plexusone/omnillm-langchaingo) | LangChainGo adapter |
| [langchaingo](https://github.com/tmc/langchaingo) | Go LLM framework |
