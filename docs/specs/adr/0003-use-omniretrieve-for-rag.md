# ADR-0003: Use omniretrieve for RAG and GraphRAG

## Status

Accepted

## Context

The trading agent needs retrieval capabilities for:

- Semantic search over markets and events
- Knowledge graph for relationship-aware retrieval
- Hybrid retrieval combining vector and graph approaches

Options considered:

1. Build custom RAG implementation
2. Use LangChain's retrieval components
3. Use omniretrieve (internal library)

## Decision

Use [plexusone/omniretrieve](https://github.com/plexusone/omniretrieve) for all retrieval operations.

omniretrieve provides:

- **Vector retrieval** with pluggable backends (in-memory, pgvector)
- **Graph retrieval** with knowledge graph traversal
- **Hybrid retrieval** combining vector + graph with configurable weights
- **Unified interfaces** for consistent API across retrieval modes

## Consequences

**Positive:**

- Single library for all retrieval patterns
- Consistent API across vector, graph, and hybrid modes
- Supports production backends (pgvector, Neo4j planned)
- Built-in observability and tracing
- We control the library and can extend as needed

**Negative:**

- Less mature than established libraries
- Smaller community

**Implementation:**

- `internal/rag/store.go` - Vector-based RAG store
- `internal/rag/graphrag.go` - GraphRAG with knowledge graph
- `internal/rag/adapter.go` - Adapter for prompts package integration
