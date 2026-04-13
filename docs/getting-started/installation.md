# Installation

## Requirements

- Go 1.21 or later
- Git

## Install via Go

```bash
go get github.com/grokify/polymarket-go
```

## Install from Source

```bash
git clone https://github.com/grokify/polymarket-go.git
cd polymarket-go
go mod tidy
```

## Verify Installation

```bash
go run ./cmd/polymarket-agent/ --help
```

## Dependencies

The following dependencies are automatically installed:

| Package | Version | Purpose |
|---------|---------|---------|
| `omnillm-core` | v0.15.0 | LLM provider abstraction |
| `omnillm-langchaingo` | v0.1.0 | LangChainGo adapter |
| `langchaingo` | v0.1.14 | Chains, agents, memory |
| `yaml.v3` | v3.0.1 | Agent spec parsing |

## Optional: Environment Variables

For LLM integration, set your API keys:

```bash
# For Anthropic Claude
export ANTHROPIC_API_KEY=your-key-here

# For OpenAI
export OPENAI_API_KEY=your-key-here
```

## Next Steps

- [Quick Start](quickstart.md) - Run your first demo
- [CLI Usage](cli.md) - Learn the command-line interface
