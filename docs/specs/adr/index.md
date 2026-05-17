# Architecture Decision Records

This directory contains Architecture Decision Records (ADRs) documenting significant architectural decisions made in polymarket-go.

## What is an ADR?

An ADR is a document that captures an important architectural decision made along with its context and consequences. ADRs help:

- Document the reasoning behind decisions
- Provide context for future developers
- Enable better architectural discussions
- Track the evolution of the system

## ADR Index

| ADR | Title | Status |
|-----|-------|--------|
| [0001](0001-use-polymarket-go-sdk.md) | Use polymarket-go-sdk for Trading | Accepted |
| [0002](0002-use-cobra-for-cli.md) | Use Cobra for CLI Framework | Accepted |
| [0003](0003-use-omniretrieve-for-rag.md) | Use omniretrieve for RAG and GraphRAG | Accepted |
| [0004](0004-use-omniserp-for-search.md) | Use omniserp for News and Web Search | Accepted |
| [0005](0005-use-omnillm-for-llm-abstraction.md) | Use omnillm-core for LLM Abstraction | Accepted |
| [0006](0006-resilience-patterns.md) | Implement Resilience Patterns | Accepted |
| [0007](0007-graphrag-knowledge-graph-design.md) | GraphRAG Knowledge Graph Design | Accepted |
| [0008](0008-rest-server-with-huma-chi.md) | REST Server with Huma and Chi | Accepted |

## ADR Template

New ADRs should follow this template:

```markdown
# ADR-NNNN: Title

## Status

Proposed | Accepted | Deprecated | Superseded

## Context

What is the issue that we're seeing that motivates this decision?

## Decision

What is the change that we're proposing and/or doing?

## Consequences

**Positive:**
- What becomes easier?

**Negative:**
- What becomes more difficult?

**Implementation:**
- Key files and packages affected
```

## References

- [ADR GitHub Organization](https://adr.github.io/)
- [Michael Nygard's ADR article](https://cognitect.com/blog/2011/11/15/documenting-architecture-decisions)
