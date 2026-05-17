# Technical Requirements Document: polymarket-go

## Overview

polymarket-go is a Go-based autonomous trading agent system for Polymarket prediction markets. It leverages the plexusone composable ecosystem for LLM interactions, retrieval, and multi-agent orchestration.

## Design Philosophy

### Composable, Not Monolithic

The plexusone ecosystem follows a **composable architecture** where each component:

- Has a single responsibility
- Can be used independently or combined
- Integrates through well-defined interfaces
- Supports multiple deployment patterns

This contrasts with monolithic frameworks that force all-or-nothing adoption.

### Schema-Driven Agents

Agent definitions are **portable specifications** that can be deployed to multiple targets:

| Deployment Target | Use Case |
|-------------------|----------|
| Claude Code subagents | Development, prototyping, CLI workflows |
| Single Go server | Production monolith, low-latency |
| Microservices | Scalable, distributed, cloud-native |
| Kubernetes | Container orchestration |
| AWS AgentCore | Serverless, managed infrastructure |

The same `multi-agent-spec` agent definitions work across all targets with different deployment schemas.

## Component Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                         polymarket-go                                │
│                    (Application Layer)                               │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │                    multi-agent-spec                          │   │
│  │         (Agent Definitions & Team Orchestration)             │   │
│  │                                                              │   │
│  │  • Agent specs (Markdown + YAML frontmatter)                │   │
│  │  • Team workflows (chain, scatter, graph, crew, swarm)      │   │
│  │  • Deployment schemas (Claude Code, Go server, K8s, etc.)   │   │
│  └─────────────────────────────────────────────────────────────┘   │
│                              │                                       │
│              ┌───────────────┼───────────────┐                      │
│              ▼               ▼               ▼                      │
│  ┌───────────────┐  ┌───────────────┐  ┌───────────────┐          │
│  │  omniretrieve │  │  langchaingo  │  │    omnillm    │          │
│  │     (RAG)     │  │  (Primitives) │  │(LLM Providers)│          │
│  └───────────────┘  └───────────────┘  └───────┬───────┘          │
│         │                   │                   │                   │
│         │                   │          ┌────────┴────────┐         │
│         │                   │          │omnillm-langchaingo        │
│         │                   │          │    (Adapter)    │         │
│         │                   └──────────┴─────────────────┘         │
│         │                                                           │
│  ┌──────┴──────────────────────────────────────────────────────┐  │
│  │                     polymarket-kit                           │  │
│  │              (CLOB, Gamma, Data APIs)                        │  │
│  │         github.com/HuakunShen/polymarket-kit/go-client       │  │
│  └──────────────────────────────────────────────────────────────┘  │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

## Component Responsibilities

### plexusone/omnillm

**Purpose:** Unified LLM provider abstraction

**Key Features:**

- Single interface for multiple providers (OpenAI, Anthropic, Gemini, Bedrock, Ollama)
- Thin providers (stdlib HTTP) and thick providers (official SDKs)
- Priority-based provider registry
- Observability hooks

**Usage in polymarket-go:**

```go
import "github.com/plexusone/omnillm"

client := omnillm.NewClient(omnillm.ClientConfig{
    Providers: []omnillm.ProviderConfig{
        {Provider: omnillm.ProviderNameAnthropic, APIKey: os.Getenv("ANTHROPIC_API_KEY")},
        {Provider: omnillm.ProviderNameOpenAI, APIKey: os.Getenv("OPENAI_API_KEY")},
    },
})
```

### plexusone/omnillm-langchaingo

**Purpose:** Bridge omnillm to LangChainGo's `llms.Model` interface

**Key Features:**

- Implements `GenerateContent()` and `Call()` methods
- Streaming support with callbacks
- Tool/function calling conversion
- Token usage tracking

**Usage:**

```go
import (
    "github.com/plexusone/omnillm"
    "github.com/plexusone/omnillm-langchaingo"
)

llm := langchaingo.New(omnillmClient, "claude-sonnet-4-20250514")
// Use with LangChainGo chains, prompts, memory
```

### plexusone/omniretrieve

**Purpose:** Unified retrieval for RAG systems

**Key Features:**

- Vector similarity search (pgvector, in-memory)
- Knowledge graph traversal
- Hybrid retrieval strategies (parallel, vector→graph, graph→vector)
- Reranking (cross-encoder, heuristic)
- Full observability/tracing

**Usage in polymarket-go:**

```go
import (
    "github.com/plexusone/omniretrieve/retrieve"
    "github.com/plexusone/omniretrieve/hybrid"
)

retriever := hybrid.New(vectorIndex, knowledgeGraph, hybrid.Config{
    Policy:       hybrid.PolicyParallel,
    VectorWeight: 0.6,
    GraphWeight:  0.4,
})

result, _ := retriever.Retrieve(ctx, retrieve.Query{
    Text:     "What are the current odds on Bitcoin reaching $150k?",
    TopK:     10,
    MinScore: 0.7,
})
```

### plexusone/multi-agent-spec

**Purpose:** Portable agent definitions and team orchestration

**Key Features:**

- Agent specs in Markdown with YAML frontmatter
- 6 workflow types (3 deterministic, 3 self-directed)
- Deployment schemas for multiple targets
- Go SDK for loading and execution

**Agent Definition Example:**

```markdown
---
name: market-analyst
description: Analyzes Polymarket prediction markets
model: sonnet
tools: [WebSearch, WebFetch, Read, Write]
role: Market Research Analyst
goal: Identify mispriced markets with >10% edge
---

You are a market analyst specializing in prediction markets...
```

**Team Definition Example:**

```json
{
  "name": "polymarket-trading-team",
  "version": "1.0.0",
  "workflow": {
    "type": "graph",
    "steps": [
      {
        "name": "research",
        "agent": "market-analyst",
        "outputs": [{"name": "market_candidates", "type": "array"}]
      },
      {
        "name": "forecast",
        "agent": "superforecaster",
        "depends_on": ["research"],
        "inputs": [{"name": "markets", "from": "research.market_candidates"}]
      },
      {
        "name": "execute",
        "agent": "trader",
        "depends_on": ["forecast"]
      }
    ]
  }
}
```

### tmc/langchaingo

**Purpose:** LLM application primitives (used selectively)

**What We Use:**

- Prompt templates with variable substitution
- Memory interfaces for conversation history
- Output parsers for structured extraction
- Chain composition

**What We Don't Use:**

- Agent executor (replaced by multi-agent-spec workflows)
- Built-in LLM providers (replaced by omnillm)
- Built-in retrievers (replaced by omniretrieve)

### polymarket-kit/go-client

**Purpose:** Polymarket API client

**Key Features:**

- CLOB API (trading, orders, positions)
- Gamma API (market discovery, events)
- Data API (leaderboards, trades)
- WebSocket streaming
- EIP-712 and HMAC authentication

## Deployment Patterns

### Pattern 1: Claude Code Subagents

**Use Case:** Development, prototyping, interactive workflows

**Deployment Schema:**

```json
{
  "platform": "claude-code",
  "config": {
    "agent_dir": ".claude/agents",
    "model_mapping": {
      "haiku": "claude-haiku-4",
      "sonnet": "claude-sonnet-4",
      "opus": "claude-opus-4"
    }
  }
}
```

**Execution:** Claude Code spawns subagents via Task tool, each agent runs as a separate conversation context.

### Pattern 2: Single Go Server

**Use Case:** Production monolith, low-latency trading

**Deployment Schema:**

```json
{
  "platform": "go-server",
  "config": {
    "port": 8080,
    "concurrency": "goroutines",
    "agent_executor": "in-process"
  }
}
```

**Execution:** All agents run in-process as goroutines, communicating via channels.

```go
type AgentServer struct {
    agents   map[string]*Agent
    omnillm  *omnillm.ChatClient
    retrieve retrieve.Retriever
}

func (s *AgentServer) ExecuteWorkflow(ctx context.Context, team *mas.Team) error {
    // Execute workflow steps, agents communicate via channels
}
```

### Pattern 3: Microservices

**Use Case:** Scalable, distributed, cloud-native

**Deployment Schema:**

```json
{
  "platform": "kubernetes",
  "config": {
    "namespace": "polymarket-agents",
    "communication": "grpc",
    "service_mesh": "istio",
    "agents": {
      "market-analyst": {"replicas": 2, "resources": {"cpu": "500m"}},
      "superforecaster": {"replicas": 1, "resources": {"cpu": "1000m"}},
      "trader": {"replicas": 1, "resources": {"cpu": "250m"}}
    }
  }
}
```

**Execution:** Each agent is a separate service, communicating via gRPC or message queues.

## Data Flow

### Trading Workflow

```
1. Market Discovery (Gamma API)
   └─→ Fetch active markets, events, prices

2. RAG Context (omniretrieve)
   └─→ Vector search: similar historical markets
   └─→ Graph traversal: related events, outcomes
   └─→ Hybrid merge with reranking

3. Analysis (multi-agent-spec workflow)
   └─→ Step 1: market-analyst identifies candidates
   └─→ Step 2: superforecaster estimates probabilities
   └─→ Step 3: risk-assessor validates edge

4. Execution (polymarket-kit)
   └─→ Build signed order
   └─→ Submit to CLOB API
   └─→ Monitor fill status
```

### Agent Communication

**Deterministic Workflows (chain, scatter, graph):**

- Schema controls execution order
- Outputs flow through defined ports
- No inter-agent messaging needed

**Self-Directed Workflows (crew, swarm, council):**

- Agents communicate via channels
- Message types: delegate_work, ask_question, share_finding, vote
- Lead agent or consensus determines outcomes

## Integration Points

### omnillm ↔ LangChainGo

```go
// omnillm-langchaingo adapter bridges the interface gap
import "github.com/plexusone/omnillm-langchaingo"

llm := langchaingo.New(omnillmClient, model)

// Now usable with LangChainGo primitives
chain := chains.NewLLMChain(llm, prompt)
```

### omniretrieve ↔ Agent Context

```go
// Inject retrieved context into agent prompts
result, _ := retriever.Retrieve(ctx, query)

prompt := fmt.Sprintf(`
%s

## Retrieved Context
%s

## Query
%s
`, agent.Instructions, formatContext(result), userQuery)
```

### multi-agent-spec ↔ Execution Engine

```go
// Load specs from agents/specs/
loader := mas.Loader{}
agents, _ := loader.LoadAgentsFromDir("agents/specs/agents/")
team, _ := loader.LoadTeamFromFile("agents/specs/team.json")

// Execute based on deployment target
switch deployment.Platform {
case "claude-code":
    executor := NewClaudeCodeExecutor(agents, team)
case "go-server":
    executor := NewInProcessExecutor(agents, team, omnillmClient)
case "kubernetes":
    executor := NewK8sExecutor(agents, team, grpcClient)
}

executor.Run(ctx)
```

## Future Work

### Phase 1: Core Infrastructure

- [ ] Project scaffolding with go.mod
- [ ] Integration tests for component composition
- [ ] Basic Claude Code deployment

### Phase 2: Agent Implementation

- [ ] Define Polymarket-specific agents (analyst, forecaster, trader)
- [ ] Implement RAG pipeline for market context
- [ ] Build order execution logic

### Phase 3: Production Readiness

- [ ] Go server deployment with API
- [ ] Observability and monitoring
- [ ] Risk management and position limits

### Phase 4: Scale

- [ ] Kubernetes deployment
- [ ] Multi-agent coordination
- [ ] Backtesting framework

## References

- [polymarket-kit](https://github.com/HuakunShen/polymarket-kit) - Polymarket API client
- [omnillm](https://github.com/plexusone/omnillm) - Unified LLM providers
- [omniretrieve](https://github.com/plexusone/omniretrieve) - RAG retrieval
- [multi-agent-spec](https://github.com/plexusone/multi-agent-spec) - Agent specifications
- [langchaingo](https://github.com/tmc/langchaingo) - LLM primitives for Go
