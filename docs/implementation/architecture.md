# RedisForge Implementation Architecture

RedisForge is shaped like a small production service. The point is not to create a large domain. The point is to make Redis patterns easy to inspect in realistic backend code.

## System Flow

```mermaid
graph LR
    A([Client]) --> B[HTTP Router]
    B --> C[Handler]
    C --> D[Repository Interface]
    D --> E[Cache-Aside Decorator]
    E --> F[(RedisJSON Cache)]
    E -.-> G[(Fallback Store)]
    C -- Write --> H[(Redis Streams)]
    H --> I[Audit Worker]
    C -- Search --> J[(RediSearch)]
    J -.-> F
```

## Code Map

| Area | Path | Responsibility |
| --- | --- | --- |
| Entrypoint | `cmd/redisforge/main.go` | Calls `app.Run()` and exits on fatal startup errors |
| App wiring | `internal/app/app.go` | Creates config, logger, Redis clients, repositories, workers, router, and shutdown flow |
| Config | `internal/config` | Typed environment config and validation |
| Domain | `internal/domain` | `Item`, audit event, and sentinel errors |
| Handlers | `internal/handlers` | HTTP endpoints and JSON/error helpers |
| Redis wrappers | `internal/redisx` | Small focused wrappers around Redis modules |
| Repositories | `internal/repo` | Item repository interface, memory fallback, cache-aside decorator |
| Workers | `internal/workers` | Redis Streams audit consumer |
| Observability | `internal/observability` | Metrics and tracing hooks |
| Deployments | `deployments` | Redis Stack, Sentinel, Cluster, Prometheus, Grafana |

## Request Lifecycle

Example: `GET /v1/items/{id}`

```mermaid
sequenceDiagram
    participant C as Client
    participant R as chi Router
    participant H as HandleGetItem
    participant CR as CacheItemRepo
    participant JS as JSONStore
    participant FB as Fallback Store

    C->>R: GET /v1/items/{id}
    R->>H: Dispatch
    H->>CR: GetByID(id)
    CR->>JS: GetItem(id)
    alt Cache Hit
        JS-->>CR: Item
        CR-->>H: Item
    else Cache Miss
        JS-->>CR: ErrNotFound
        CR->>FB: GetByID(id)
        FB-->>CR: Item
        CR->>JS: SetItem (backfill)
        CR-->>H: Item
    end
    H-->>C: JSON Response
```

This is a cache-aside read path. Redis accelerates reads, but the fallback repository remains the source of truth.

## Write Lifecycle

Example: `POST /v1/items`

```mermaid
sequenceDiagram
    participant C as Client
    participant H as HandleCreateItem
    participant BF as BloomFilter
    participant R as Repository
    participant S as StreamClient

    C->>H: POST /v1/items
    H->>BF: Exists(idempotency_key)
    alt Might Exist
        BF-->>H: true
        H-->>C: 409 Duplicate
    else New Key
        BF-->>H: false
        H->>R: Create(item)
        R-->>H: Created Item
        H->>BF: Add(idempotency_key)
        H->>S: Append(audit event)
        H-->>C: 201 Created
    end
```

The Bloom filter is a fast pre-check. If it says a key does not exist, it is definitely new. If it says a key might exist, RedisForge treats it as a duplicate to demonstrate the shape of the idempotency flow.

## Worker Lifecycle

The audit worker demonstrates durable background processing:

```mermaid
sequenceDiagram
    participant S as Redis Stream
    participant W as Audit Worker

    W->>S: EnsureGroup (audit-processors)
    loop Every 500ms
        W->>S: XREADGROUP (block 2s)
        S-->>W: New Events
        W->>W: Process Event
        W->>S: XACK
    end
    loop Every 5s
        W->>S: XAUTOCLAIM (idle > 30s)
        S-->>W: Stale Events
        W->>W: Process Stale Event
        W->>S: XACK
    end
```

Use this as the reference implementation when revising Redis Streams.

## Topology Boundary

`internal/redisx/client.go` returns a `redis.UniversalClient`. That keeps topology concerns in one place:

- Single-node client for local learning.
- Sentinel failover client for high availability.
- Cluster client for horizontal scale.

The rest of the app receives an interface and does not need to know which topology is active.

## Why The Domain Is Small

The project uses Items because a small domain keeps the architecture readable. The important lessons are the Redis patterns:

- cache-aside reads
- write invalidation/update discipline
- JSON document storage
- search index design
- idempotency with probabilistic data structures
- durable async processing
- topology-aware Redis clients
- observability around cache and Redis behavior

