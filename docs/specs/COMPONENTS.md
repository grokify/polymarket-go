# Component Integration Guide

## plexusone Ecosystem

The plexusone ecosystem provides composable building blocks for AI applications. Each component is independently useful but designed to work together seamlessly.

```
┌─────────────────────────────────────────────────────────────────┐
│                     plexusone ecosystem                          │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────────┐ │
│  │  omnillm    │  │omniretrieve │  │    multi-agent-spec     │ │
│  │  (LLM)      │  │   (RAG)     │  │  (Agent Definitions)    │ │
│  └──────┬──────┘  └──────┬──────┘  └────────────┬────────────┘ │
│         │                │                      │               │
│         │         ┌──────┴──────┐               │               │
│         │         │             │               │               │
│  ┌──────┴──────┐  │             │               │               │
│  │  omnillm-   │  │             │               │               │
│  │ langchaingo │  │             │               │               │
│  └──────┬──────┘  │             │               │               │
│         │         │             │               │               │
│         └─────────┴─────────────┴───────────────┘               │
│                         │                                        │
│                         ▼                                        │
│              ┌─────────────────────┐                            │
│              │   Your Application  │                            │
│              │   (polymarket-go)   │                            │
│              └─────────────────────┘                            │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

## Component Details

### omnillm

**Module:** `github.com/plexusone/omnillm`

**Variants:**

| Package | Dependencies | Use Case |
|---------|--------------|----------|
| `omnillm-core` | stdlib only | Minimal footprint, thin providers |
| `omnillm` | All official SDKs | Batteries-included, thick providers |
| `omnillm-openai` | OpenAI SDK | Just OpenAI with full SDK features |
| `omnillm-anthropic` | Anthropic SDK | Just Anthropic with full SDK features |

**Interface:**

```go
type Provider interface {
    CreateChatCompletion(ctx context.Context, req *ChatCompletionRequest) (*ChatCompletionResponse, error)
    CreateChatCompletionStream(ctx context.Context, req *ChatCompletionRequest) (ChatCompletionStream, error)
    Close() error
    Name() string
}
```

### omnillm-langchaingo

**Module:** `github.com/plexusone/omnillm-langchaingo`

**Purpose:** Adapter to use omnillm with LangChainGo

**Interface Implemented:**

```go
// From github.com/tmc/langchaingo/llms
type Model interface {
    GenerateContent(ctx context.Context, messages []MessageContent, options ...CallOption) (*ContentResponse, error)
    Call(ctx context.Context, prompt string, options ...CallOption) (string, error)
}
```

**Usage:**

```go
import (
    "github.com/plexusone/omnillm"
    "github.com/plexusone/omnillm-langchaingo"
    "github.com/tmc/langchaingo/chains"
)

// Create omnillm client
client := omnillm.NewClient(config)

// Wrap with LangChainGo adapter
llm := langchaingo.New(client, "claude-sonnet-4-20250514")

// Use with LangChainGo
chain := chains.NewLLMChain(llm, prompt)
result, _ := chain.Call(ctx, inputs)
```

### omniretrieve

**Module:** `github.com/plexusone/omniretrieve`

**Core Interface:**

```go
type Retriever interface {
    Retrieve(ctx context.Context, q Query) (*Result, error)
}

type Query struct {
    Text      string
    Embedding []float32        // Optional pre-computed
    TopK      int
    MinScore  float64
    Modes     []Mode           // Vector, Graph, Hybrid
    Filters   map[string]string
}
```

**Retrieval Modes:**

| Mode | Implementation | Best For |
|------|----------------|----------|
| Vector | Similarity search | Semantic matching |
| Graph | BFS traversal | Relationship exploration |
| Hybrid | Combined | Comprehensive context |

**Hybrid Policies:**

```go
// Parallel: Run both, merge with weights
hybrid.PolicyParallel

// Vector first, expand via graph
hybrid.PolicyVectorThenGraph

// Graph first, ground via vector
hybrid.PolicyGraphThenVector
```

### multi-agent-spec

**Module:** `github.com/plexusone/multi-agent-spec/sdk/go`

**Agent Definition Format:**

```markdown
---
name: market-analyst
namespace: trading
description: Analyzes prediction markets
model: sonnet
tools: [WebSearch, WebFetch, Read]
role: Market Research Analyst
goal: Identify alpha opportunities
backstory: Expert in prediction market dynamics
dependencies: [data-fetcher]
---

You are a market analyst specializing in prediction markets.
Your task is to identify mispriced markets...
```

**Team Definition Format:**

```json
{
  "name": "trading-team",
  "version": "1.0.0",
  "agents": ["trading/market-analyst", "trading/forecaster", "trading/executor"],
  "workflow": {
    "type": "graph",
    "steps": [...]
  }
}
```

**Workflow Types:**

| Type | Category | Execution Control |
|------|----------|-------------------|
| `chain` | Deterministic | Schema |
| `scatter` | Deterministic | Schema |
| `graph` | Deterministic | Schema |
| `crew` | Self-directed | Lead agent |
| `swarm` | Self-directed | Task queue |
| `council` | Self-directed | Consensus |

## Deployment Flexibility

The same agent definitions can be deployed to multiple targets:

### Claude Code Subagents

**deployment.json:**

```json
{
  "platform": "claude-code",
  "team": "trading-team",
  "config": {
    "agent_dir": ".claude/agents",
    "model_mapping": {
      "haiku": "claude-haiku-4",
      "sonnet": "claude-sonnet-4",
      "opus": "claude-opus-4"
    },
    "tool_mapping": {
      "WebSearch": "WebSearch",
      "WebFetch": "WebFetch",
      "Read": "Read",
      "Write": "Write",
      "Bash": "Bash"
    }
  }
}
```

**Generated Output:** Markdown files in `.claude/agents/` that Claude Code can invoke via Task tool.

### Go Server (Monolith)

**deployment.json:**

```json
{
  "platform": "go-server",
  "team": "trading-team",
  "config": {
    "port": 8080,
    "metrics_port": 9090,
    "concurrency": {
      "model": "goroutines",
      "max_concurrent_agents": 10
    },
    "llm": {
      "provider": "omnillm",
      "default_model": "claude-sonnet-4-20250514"
    },
    "retrieval": {
      "provider": "omniretrieve",
      "vector_backend": "pgvector",
      "connection_string": "${PGVECTOR_URL}"
    }
  }
}
```

**Execution Model:**

```go
type InProcessExecutor struct {
    agents    map[string]*mas.Agent
    team      *mas.Team
    llm       provider.Provider
    retriever retrieve.Retriever
}

func (e *InProcessExecutor) ExecuteStep(ctx context.Context, step mas.Step) (map[string]any, error) {
    agent := e.agents[step.Agent]

    // Build context with RAG
    ragContext, _ := e.retriever.Retrieve(ctx, retrieve.Query{
        Text: step.Inputs["query"].(string),
        TopK: 10,
    })

    // Execute via omnillm
    resp, _ := e.llm.CreateChatCompletion(ctx, &provider.ChatCompletionRequest{
        Model: agent.Model.String(),
        Messages: []provider.Message{
            {Role: provider.RoleSystem, Content: agent.Instructions},
            {Role: provider.RoleUser, Content: buildPrompt(ragContext, step.Inputs)},
        },
    })

    return parseOutputs(resp, step.Outputs), nil
}
```

### Microservices (Kubernetes)

**deployment.json:**

```json
{
  "platform": "kubernetes",
  "team": "trading-team",
  "config": {
    "namespace": "polymarket",
    "communication": {
      "protocol": "grpc",
      "service_mesh": "istio"
    },
    "agents": {
      "trading/market-analyst": {
        "replicas": 2,
        "resources": {"cpu": "500m", "memory": "512Mi"},
        "autoscaling": {"min": 1, "max": 5, "target_cpu": 70}
      },
      "trading/forecaster": {
        "replicas": 1,
        "resources": {"cpu": "1000m", "memory": "1Gi"}
      },
      "trading/executor": {
        "replicas": 1,
        "resources": {"cpu": "250m", "memory": "256Mi"}
      }
    },
    "shared": {
      "llm_service": "omnillm-gateway",
      "retrieval_service": "omniretrieve-api",
      "message_queue": "nats"
    }
  }
}
```

**Generated Artifacts:**

- Helm chart with per-agent deployments
- Service definitions for gRPC communication
- ConfigMaps for agent instructions
- Secrets for API keys

## Composition Patterns

### Pattern 1: Minimal (Direct omnillm)

```go
import "github.com/plexusone/omnillm"

client := omnillm.NewClient(config)
resp, _ := client.ChatCompletion(ctx, req)
```

**When to use:** Simple LLM calls without orchestration or RAG.

### Pattern 2: With LangChainGo Primitives

```go
import (
    "github.com/plexusone/omnillm"
    "github.com/plexusone/omnillm-langchaingo"
    "github.com/tmc/langchaingo/chains"
    "github.com/tmc/langchaingo/prompts"
)

llm := langchaingo.New(omnillmClient, model)
chain := chains.NewLLMChain(llm, prompt)
```

**When to use:** Need prompt templates, memory, or output parsing.

### Pattern 3: With RAG

```go
import (
    "github.com/plexusone/omnillm"
    "github.com/plexusone/omniretrieve/hybrid"
)

retriever := hybrid.New(vectorIndex, graph, config)
context, _ := retriever.Retrieve(ctx, query)

// Inject into prompt
resp, _ := omnillmClient.ChatCompletion(ctx, &provider.ChatCompletionRequest{
    Messages: []provider.Message{
        {Role: provider.RoleSystem, Content: systemPrompt},
        {Role: provider.RoleUser, Content: fmt.Sprintf("Context:\n%s\n\nQuery: %s", context, query)},
    },
})
```

**When to use:** Need grounded responses with retrieved context.

### Pattern 4: Full Stack (Multi-Agent)

```go
import (
    "github.com/plexusone/omnillm"
    "github.com/plexusone/omnillm-langchaingo"
    "github.com/plexusone/omniretrieve/hybrid"
    mas "github.com/plexusone/multi-agent-spec/sdk/go"
)

// Load specs from agents/specs/
loader := mas.Loader{}
agents, _ := loader.LoadAgentsFromDir("agents/specs/agents/")
team, _ := loader.LoadTeamFromFile("agents/specs/team.json")

// Create components
llm := langchaingo.New(omnillmClient, defaultModel)
retriever := hybrid.New(vectorIndex, graph, config)

// Execute workflow
executor := NewWorkflowExecutor(agents, team, llm, retriever)
result, _ := executor.Run(ctx, inputs)
```

**When to use:** Complex multi-step workflows with multiple specialized agents.

## Why Not Use LangChainGo Agents Directly?

| Aspect | LangChainGo Agents | multi-agent-spec |
|--------|-------------------|------------------|
| Definition | Code-based | Schema-based (portable) |
| Multi-agent | Single executor | Full team orchestration |
| Deployment | Go only | Multiple targets |
| Workflow types | ReAct loop only | 6 types (deterministic + self-directed) |
| Tool mapping | Code changes | Config changes |

**Recommendation:** Use multi-agent-spec for agent definitions and orchestration. Use LangChainGo for primitives (prompts, memory, parsing) when needed.

## Next Steps

1. Define Polymarket-specific agents in `agents/` directory
2. Create team workflow for trading pipeline
3. Implement deployment schemas for each target
4. Build execution engine that reads specs and runs workflows
