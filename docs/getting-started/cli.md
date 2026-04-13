# CLI Usage

The `polymarket-agent` CLI loads agent specifications and executes trading workflows.

## Basic Usage

```bash
go run ./cmd/polymarket-agent/ [flags]
```

## Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--demo` | bool | false | Run demo fetching live Polymarket data |
| `--demo-limit` | int | 5 | Number of markets to fetch in demo mode |

## Examples

### Run Without Demo

Load and display agent specs without fetching live data:

```bash
go run ./cmd/polymarket-agent/
```

Output:

```
level=INFO msg="loading agents" dir=agents/specs/agents
level=INFO msg="loaded agents" count=3
level=INFO msg=agent name=market-analyst model=sonnet tools="[WebSearch WebFetch Read Write]"
level=INFO msg=agent name=superforecaster model=sonnet tools="[WebSearch WebFetch Read]"
level=INFO msg=agent name=trader model=haiku tools="[Read Write]"
level=INFO msg="loading team" file=agents/specs/team.json
level=INFO msg="loaded team" name=polymarket-trading-team version=0.1.0 workflow_type=graph steps=3
level=INFO msg="loading deployment" file=agents/specs/deployment-go-server.json
level=INFO msg="loaded deployment" platform=go-server team=polymarket-trading-team
level=INFO msg="workflow steps"
level=INFO msg="step 1" name=discover agent=market-analyst depends_on=none
level=INFO msg="step 2" name=forecast agent=superforecaster depends_on=[discover]
level=INFO msg="step 3" name=execute agent=trader depends_on=[forecast]
level=INFO msg="ready to execute" note="omnillm integration pending"
```

### Run Demo Mode

Fetch live market data from Polymarket:

```bash
go run ./cmd/polymarket-agent/ --demo
```

### Fetch More Markets

```bash
go run ./cmd/polymarket-agent/ --demo --demo-limit=20
```

## Build Binary

```bash
go build -o polymarket-agent ./cmd/polymarket-agent/
./polymarket-agent --demo
```

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | Error (check stderr for details) |

## Configuration Files

The CLI loads configuration from these paths:

| File | Purpose |
|------|---------|
| `agents/specs/agents/*.md` | Agent definitions |
| `agents/specs/team.json` | Team workflow |
| `agents/specs/deployment-go-server.json` | Deployment config |
