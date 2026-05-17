# ADR-0008: REST Server with Huma and Chi

## Status

Accepted

## Context

The project needs a REST API server that:

- Exposes the internal SDK functionality via HTTP
- Automatically generates an OpenAPI specification
- Maintains consistency with CLI and future MCP server implementations
- Follows the principle of thin wrappers around internal packages

Options considered:

1. Standard library `net/http` with manual OpenAPI spec
2. Gin/Echo with swagger annotations
3. Huma with Chi router

## Decision

Use [Huma v2](https://github.com/danielgtaylor/huma) with [Chi v5](https://github.com/go-chi/chi) router.

Huma provides:

- **Automatic OpenAPI generation** from Go types and struct tags
- **Type-safe request/response handling** with input validation
- **Chi adapter** for seamless integration with Chi middleware
- **Built-in documentation UI** at `/docs`

Chi provides:

- **Lightweight router** with middleware support
- **Standard middleware** (logging, recovery, timeouts)
- **Context-based routing**

## Implementation

The server follows a thin wrapper pattern:

```
internal/server/
├── server.go    # Server setup, Chi + Huma configuration
├── routes.go    # Route registration
└── handlers.go  # Request/response types and handlers
```

Key design decisions:

1. **Request/Response types** define the API contract using Huma struct tags
2. **Handlers** are thin wrappers that delegate to internal packages
3. **Optional features** (news, RAG) are conditionally registered based on configuration

### API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/health` | Health check |
| GET | `/markets` | List markets with filters |
| GET | `/markets/{conditionId}` | Get single market |
| GET | `/markets/{tokenId}/orderbook` | Get order book |
| GET | `/markets/{tokenId}/price` | Get token price |
| GET | `/news` | Search news (optional) |
| GET | `/search` | Web search (optional) |
| POST | `/rag/markets/search` | Semantic market search (optional) |
| POST | `/rag/events/search` | Semantic event search (optional) |
| POST | `/rag/markets/index` | Index markets for RAG (optional) |

## Consequences

**Positive:**

- OpenAPI spec generated automatically from Go types
- Type-safe API with compile-time checking
- Documentation UI available without extra tooling
- Thin wrapper maintains separation of concerns
- Easy to extend with new endpoints

**Negative:**

- Huma has specific conventions for request/response types
- Learning curve for Huma struct tags
- Additional dependencies (huma, chi)

**Dependencies Added:**

- `github.com/danielgtaylor/huma/v2` v2.38.0
- `github.com/go-chi/chi/v5` v5.2.5
