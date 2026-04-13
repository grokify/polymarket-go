# Changelog

All notable changes to this project are documented in this file.

This changelog is automatically generated from [CHANGELOG.json](../../CHANGELOG.json) using [schangelog](https://github.com/grokify/structured-changelog).

---

## [v0.1.0] - 2026-04-13

Initial release of polymarket-go, a Go SDK for building AI trading agents on Polymarket prediction markets.

### Highlights

- Complete Polymarket API client for Gamma (markets) and CLOB (trading) APIs
- Multi-agent workflow execution based on multi-agent-spec
- Three specialized agents: Market Analyst, Superforecaster, and Trader
- Integration with omnillm for LLM provider abstraction

### Added

- **Polymarket Client**: Full API client with market discovery, order book access, and price queries
- **Workflow Executor**: DAG-based workflow execution with dependency resolution
- **Spec Loader**: Load agent and team definitions from multi-agent-spec format
- **Market Analyst Agent**: Discovers trading opportunities with edge calculation
- **Superforecaster Agent**: Generates calibrated probability estimates
- **Trader Agent**: Executes trades with Kelly criterion position sizing
- **CLI Tool**: Command-line interface for running agent workflows
- **LangChainGo Integration**: Bridge to omnillm-langchaingo adapter

### Documentation

- API reference for Polymarket client
- Agent specifications and workflow documentation
- Getting started guide and CLI usage

---

For the structured changelog data, see [CHANGELOG.json](../../CHANGELOG.json).
