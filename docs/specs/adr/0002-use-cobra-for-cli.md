# ADR-0002: Use Cobra for CLI Framework

## Status

Accepted

## Context

The initial CLI implementation used Go's standard library `flag` package. As the feature set grew, we needed:

- Subcommand structure (e.g., `markets list`, `trade auto`)
- Consistent flag handling across commands
- Built-in help generation
- Command aliases and completion

The flat flag-based approach was becoming unwieldy.

## Decision

Migrate from `flag` package to [spf13/cobra](https://github.com/spf13/cobra) for CLI management.

Command structure:

```
polymarket-agent
├── demo              # Demo mode
├── markets
│   ├── list          # List markets
│   └── analyze       # Superforecaster analysis
├── events
│   └── list          # List events
├── trade
│   ├── auto          # Autonomous trading loop
│   └── recommend     # Single recommendation
├── rag
│   ├── index         # Build RAG index
│   └── search        # Semantic search
├── graphrag
│   ├── index         # Build knowledge graph
│   ├── related       # Find related markets
│   ├── topic         # Find by topic
│   └── hybrid        # Hybrid search
├── news              # News search
└── search            # Web search
```

## Consequences

**Positive:**

- Clean hierarchical command structure
- Automatic help text generation
- Easy to add new commands
- Consistent UX across all commands
- Shell completion support

**Negative:**

- Additional dependency
- Learning curve for contributors unfamiliar with Cobra

**Mitigations:**

- Cobra is the de facto standard for Go CLIs
- Well-documented with many examples
