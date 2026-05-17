# ADR-0007: GraphRAG Knowledge Graph Design

## Status

Accepted

## Context

Polymarket data has rich relationships:

- Events contain multiple markets
- Markets share topics/categories
- Markets may be correlated (price movements)
- Topics connect related events

Pure vector search misses these structural relationships. We need a knowledge graph to capture and traverse these connections.

## Decision

Implement a knowledge graph with the following schema:

### Node Types

| Type | Description | Example |
|------|-------------|---------|
| `polymarket_event` | Prediction event | "2024 US Election" |
| `polymarket_market` | Individual market | "Will Biden win?" |
| `market_category` | Market category | "Politics", "Crypto" |
| `event_topic` | Topic/tag | "elections", "bitcoin" |

### Edge Types

| Type | From → To | Weight | Description |
|------|-----------|--------|-------------|
| `has_market` | Event → Market | 1.0 | Event contains market |
| `topic_relates_to` | Topic → Event | 1.0 | Topic tags event |
| `category_is` | Market → Category | 1.0 | Market categorization |
| `same_event` | Market → Market | 0.9 | Sibling markets |
| `correlated_with` | Market → Market | 0.0-1.0 | Price correlation |
| `semantic_similar` | Market → Market | 0.0-1.0 | Embedding similarity |

### Traversal Patterns

1. **Find related markets**: Start from market, traverse correlation/sibling edges
2. **Find markets by topic**: Start from topic, traverse to events, then to markets
3. **Hybrid search**: Combine vector similarity with graph traversal

## Consequences

**Positive:**

- Captures structural relationships in data
- Enables relationship-aware retrieval
- Supports correlated market discovery
- Explainable results via traversal paths

**Negative:**

- Additional indexing complexity
- Graph must be kept in sync with source data
- In-memory store limits scale (pgvector/Neo4j for production)

**Implementation:**

- `internal/rag/graphrag.go` - GraphStore with schema
- CLI: `graphrag index`, `graphrag related`, `graphrag topic`, `graphrag hybrid`
