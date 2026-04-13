# Spec Loader

The loader package provides loading of multi-agent-spec definitions from files.

## Import

```go
import "github.com/grokify/polymarket-go/internal/loader"
```

## Creating a Loader

```go
loader := loader.NewLoader()
```

## Loading Agents

### LoadAgent

Load a single agent from a markdown file with YAML frontmatter.

```go
func (l *Loader) LoadAgent(path string) (*Agent, error)
```

**Example:**

```go
agent, err := loader.LoadAgent("agents/specs/agents/market-analyst.md")
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Loaded agent: %s\n", agent.Name)
```

### LoadAgentsFromDir

Load all agents from a directory recursively.

```go
func (l *Loader) LoadAgentsFromDir(dir string) ([]*Agent, error)
```

**Example:**

```go
agents, err := loader.LoadAgentsFromDir("agents/specs/agents")
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Loaded %d agents\n", len(agents))
```

### Agent Type

```go
type Agent struct {
    Name         string   `yaml:"name"`
    Namespace    string   `yaml:"namespace,omitempty"`
    Description  string   `yaml:"description"`
    Model        string   `yaml:"model"`
    Tools        []string `yaml:"tools"`
    Role         string   `yaml:"role,omitempty"`
    Goal         string   `yaml:"goal,omitempty"`
    Backstory    string   `yaml:"backstory,omitempty"`
    Dependencies []string `yaml:"dependencies,omitempty"`
    Instructions string   // Parsed from markdown body
}
```

### Agent File Format

Agent files use markdown with YAML frontmatter:

```markdown
---
name: market-analyst
model: sonnet
tools:
  - WebSearch
  - WebFetch
  - Read
  - Write
role: Market Research Analyst
goal: Identify mispriced markets with >10% expected edge
dependencies: []
---

# Instructions

You are a market research analyst specializing in prediction markets...

## Analysis Framework

1. **Understand the Question**: Parse the exact resolution criteria
2. **Gather Evidence**: Search for relevant news and data
...
```

### QualifiedName

Get the fully qualified agent name (namespace/name).

```go
func (a *Agent) QualifiedName() string
```

**Example:**

```go
// Agent with namespace "trading" and name "analyst"
agent.QualifiedName() // Returns "trading/analyst"

// Agent without namespace
agent.QualifiedName() // Returns "analyst"
```

### AgentMap

Convert agent slice to map for quick lookup.

```go
func AgentMap(agents []*Agent) map[string]*Agent
```

**Example:**

```go
agents, _ := loader.LoadAgentsFromDir("agents/specs/agents")
agentMap := loader.AgentMap(agents)

analyst := agentMap["market-analyst"]
```

## Loading Teams

### LoadTeam

Load a team specification from a JSON file.

```go
func (l *Loader) LoadTeam(path string) (*Team, error)
```

**Example:**

```go
team, err := loader.LoadTeam("agents/specs/team.json")
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Team: %s v%s\n", team.Name, team.Version)
```

### Team Type

```go
type Team struct {
    Name        string   `json:"name"`
    Version     string   `json:"version"`
    Description string   `json:"description,omitempty"`
    Agents      []string `json:"agents"`
    Workflow    Workflow `json:"workflow"`
    Context     string   `json:"context,omitempty"`
}

type Workflow struct {
    Type  string `json:"type"`  // chain, scatter, graph, crew, swarm, council
    Steps []Step `json:"steps"`
}

type Step struct {
    Name      string   `json:"name"`
    Agent     string   `json:"agent"`
    DependsOn []string `json:"depends_on,omitempty"`
    Inputs    []Port   `json:"inputs,omitempty"`
    Outputs   []Port   `json:"outputs,omitempty"`
}

type Port struct {
    Name        string `json:"name"`
    Type        string `json:"type"`
    Description string `json:"description,omitempty"`
    From        string `json:"from,omitempty"`
    Default     any    `json:"default,omitempty"`
}
```

### Team File Format

```json
{
  "name": "polymarket-trading-team",
  "version": "0.1.0",
  "description": "Autonomous trading team for Polymarket",
  "agents": ["market-analyst", "superforecaster", "trader"],
  "workflow": {
    "type": "graph",
    "steps": [
      {
        "name": "discover",
        "agent": "market-analyst",
        "outputs": [
          {"name": "market_candidates", "type": "array"}
        ]
      },
      {
        "name": "forecast",
        "agent": "superforecaster",
        "depends_on": ["discover"],
        "inputs": [
          {"name": "markets", "from": "discover.market_candidates"}
        ]
      }
    ]
  }
}
```

## Loading Deployments

### LoadDeployment

Load a deployment specification from a JSON file.

```go
func (l *Loader) LoadDeployment(path string) (*Deployment, error)
```

**Example:**

```go
deployment, err := loader.LoadDeployment("agents/specs/deployment-go-server.json")
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Platform: %s\n", deployment.Platform)
```

### Deployment Type

```go
type Deployment struct {
    Platform string           `json:"platform"`
    Team     string           `json:"team"`
    Version  string           `json:"version"`
    Config   DeploymentConfig `json:"config"`
}
```

### DeploymentConfig

Platform-specific configuration options:

```go
type DeploymentConfig struct {
    // Claude Code specific
    AgentDir     string            `json:"agent_dir,omitempty"`
    SpecDir      string            `json:"spec_dir,omitempty"`
    ModelMapping map[string]string `json:"model_mapping,omitempty"`
    ToolMapping  map[string]string `json:"tool_mapping,omitempty"`

    // Go Server specific
    Server      *ServerConfig      `json:"server,omitempty"`
    Concurrency *ConcurrencyConfig `json:"concurrency,omitempty"`
    LLM         *LLMConfig         `json:"llm,omitempty"`
    Retrieval   *RetrievalConfig   `json:"retrieval,omitempty"`
    Polymarket  *PolymarketConfig  `json:"polymarket,omitempty"`
    Risk        *RiskConfig        `json:"risk,omitempty"`
}
```

### Configuration Structs

**ServerConfig:**

```go
type ServerConfig struct {
    Port           int    `json:"port"`
    MetricsPort    int    `json:"metrics_port,omitempty"`
    HealthEndpoint string `json:"health_endpoint,omitempty"`
}
```

**ConcurrencyConfig:**

```go
type ConcurrencyConfig struct {
    Model               string `json:"model"`
    MaxConcurrentAgents int    `json:"max_concurrent_agents"`
    StepTimeoutSeconds  int    `json:"step_timeout_seconds,omitempty"`
}
```

**LLMConfig:**

```go
type LLMConfig struct {
    Provider     string            `json:"provider"`
    Config       json.RawMessage   `json:"config,omitempty"`
    ModelMapping map[string]string `json:"model_mapping,omitempty"`
}
```

**PolymarketConfig:**

```go
type PolymarketConfig struct {
    CLOBURL       string `json:"clob_url"`
    GammaURL      string `json:"gamma_url"`
    ChainID       int    `json:"chain_id"`
    PrivateKeyEnv string `json:"private_key_env"`
}
```

**RiskConfig:**

```go
type RiskConfig struct {
    MaxPositionPercent float64 `json:"max_position_percent"`
    MaxExposurePercent float64 `json:"max_exposure_percent"`
    MinEdgePercent     float64 `json:"min_edge_percent"`
    KellyMultiplier    float64 `json:"kelly_multiplier"`
}
```

## Complete Loading Example

```go
package main

import (
    "fmt"
    "log"

    "github.com/grokify/polymarket-go/internal/loader"
)

func main() {
    l := loader.NewLoader()

    // Load team
    team, err := l.LoadTeam("agents/specs/team.json")
    if err != nil {
        log.Fatal(err)
    }

    // Load agents
    agents, err := l.LoadAgentsFromDir("agents/specs/agents")
    if err != nil {
        log.Fatal(err)
    }
    agentMap := loader.AgentMap(agents)

    // Load deployment
    deployment, err := l.LoadDeployment("agents/specs/deployment-go-server.json")
    if err != nil {
        log.Fatal(err)
    }

    // Verify all agents exist
    for _, agentName := range team.Agents {
        if _, ok := agentMap[agentName]; !ok {
            log.Fatalf("Agent %s not found", agentName)
        }
    }

    fmt.Printf("Team: %s v%s\n", team.Name, team.Version)
    fmt.Printf("Agents: %d\n", len(agents))
    fmt.Printf("Platform: %s\n", deployment.Platform)
}
```
