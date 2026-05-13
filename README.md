# RedisForge

<p align="center">
	<img src="https://img.shields.io/badge/Go-1.25%2B-00ADD8?style=for-the-badge&logo=go&logoColor=white" alt="Go version" />
	<img src="https://img.shields.io/badge/Redis-Stack-DC382D?style=for-the-badge&logo=redis&logoColor=white" alt="Redis Stack" />
	<img src="https://img.shields.io/badge/Status-Feature%20Complete-2E7D32?style=for-the-badge" alt="Status" />
</p>

RedisForge is a production-shaped Go service that teaches Redis architecture patterns through one intentionally small domain: **Items**. Rather than hide Redis patterns behind a large business system, RedisForge exercises RedisJSON, RedisBloom, RediSearch, Streams, Pub/Sub, Sentinel, and Cluster topologies in explicit code you can read and modify.

The codebase is **feature-complete**: all 15 build phases are implemented and tested. The project builds cleanly, tests pass with race detection, and the service runs against single-node, Sentinel HA, and Cluster topologies.

Supporting docs include [docs/redis-decisions.md](docs/redis-decisions.md) (configuration justifications) and [docs/profiling-results.md](docs/profiling-results.md) (performance baseline for tuning).

---

## Table of Contents

- [Why This Project Exists](#why-this-project-exists)
- [Product Vision](#product-vision)
- [Current Status](#current-status)
- [Architecture](#architecture)
- [Tech Stack](#tech-stack)
- [Project Structure](#project-structure)
- [Key Components](#key-components)
- [Redis Modules & Use Cases](#redis-modules--use-cases)
- [Pub/Sub vs Streams](#pubsub-vs-streams)
- [Sentinel vs Cluster](#sentinel-vs-cluster)
- [Local Development](#local-development)
- [Testing](#testing)
- [Development Roadmap](#development-roadmap)
- [Learning Goals](#learning-goals)
- [Docs](#docs)

---

## Why This Project Exists

RedisForge teaches the part of backend engineering that simple JSON APIs let you postpone:

- **Module selection and tradeoffs** — When to use RedisJSON vs HASH, when to reach for RediSearch, when Bloom filters save a database round-trip.
- **Cache-aside architecture** — Repository patterns that keep the fallback database as source of truth while Redis multiplies throughput.
- **Document storage and indexing** — Partial-path updates, schema design for search, how to keep normalized data in sync.
- **Probabilistic data structures** — Idempotency checks with Bloom filters: false negatives guaranteed, false positives acceptable.
- **Durable async processing** — Streams with consumer groups: exactly-once semantics, claiming stale entries, handling dead consumers.
- **Ephemeral notifications** — Pub/Sub for real-time broadcasts that can be lost; contrast with Streams for durability.
- **High availability** — Sentinel for automated failover; Cluster for horizontal scale.
- **Hash-tag discipline** — Multi-key operations in Cluster mode require hash tags; the code shows why.

The goal is to keep the domain small (Items, Audit Events) so the Redis architecture stays visible rather than buried under business logic. Every decision is justified and every module has a real job.

---

## Product Vision

RedisForge is a Redis systems learning environment for one domain: **Items**. Each Item has an ID, name, category, numeric score, and tags. Simple enough that the domain never distracted—complex enough that every Redis module has a real job:

| Module | Job |
|---|---|
| **RedisJSON** | Cache whole Item objects; enable partial-path updates (e.g. increment score in-place) |
| **RedisBloom** | Fast idempotency pre-check before creating Items |
| **RediSearch** | Full-text + faceted search over cached Items |
| **Streams** | Durable audit log: every Item mutation emits an event |
| **Pub/Sub** | Ephemeral real-time notifications (compare with Streams) |
| **Sentinel** | HA failover: kill the master, watch the client reconnect |
| **Cluster** | Horizontal scale: same app, cluster client, hash tags for multi-key ops |

The intended learning flow:

1. Start with single-node RedisJSON + Bloom storage
2. Add RediSearch for discovery
3. Add Streams for durable audit processing
4. Add Pub/Sub for ephemeral notifications
5. Demonstrate Sentinel failover (HA)
6. Demonstrate Cluster sharding (scale)

At each step, the code shape stays the same—only config and topology selection change. This is how patterns port from RedisForge into production systems.

---

## Current Status

**All phases are complete and tested.** The service compiles cleanly, passes race-detection tests, and implements all 15 build phases.

| Phase | Description | Status |
|---|---|---|
| RF-0 | Bootstrap: entrypoint, config, logging, compose file | ✅ Complete |
| RF-1 | Config & logging with slog | ✅ Complete |
| RF-2 | Domain types and sentinel errors | ✅ Complete |
| RF-3 | Redis client: single-node, Sentinel, Cluster | ✅ Complete |
| RF-4 | RedisJSON wrapper: JSON.SET, JSON.GET, partial updates | ✅ Complete |
| RF-5 | RedisBloom wrapper: idempotency checks with configurable error rate | ✅ Complete |
| RF-6 | RediSearch: ON JSON indexing, full-text + faceted search | ✅ Complete |
| RF-7 | Pub/Sub wrapper: ephemeral notifications | ✅ Complete |
| RF-8 | Streams wrapper: consumer groups, claiming, XAUTOCLAIM | ✅ Complete |
| RF-9 | Cache-aside repository decorator pattern | ✅ Complete |
| RF-10 | Stream worker: durable audit processing, graceful shutdown | ✅ Complete |
| RF-11 | HTTP handlers: all CRUD and search endpoints | ✅ Complete |
| RF-12 | App wiring: dependency injection, graceful shutdown | ✅ Complete |
| RF-13 | Sentinel & Cluster topologies (config-driven) | ✅ Complete |
| RF-14 | OpenTelemetry tracing hooks | ✅ Complete |
| RF-15 | Profiling drill (SLOWLOG, LATENCY, MEMORY) | ✅ Documentation ready |

**Tests:** `go test -race ./...` passes cleanly. Integration tests use `testcontainers-go` for real Redis Stack instances.

---

## Architecture

The architecture is intentionally conservative and explicit:

- `cmd/redisforge/main.go` — entrypoint and startup
- `internal/app/app.go` — dependency wiring, server lifecycle, graceful shutdown
- `internal/config` — environment-driven typed configuration
- `internal/logging` — structured slog logger with request context
- `internal/observability` — OpenTelemetry TracerProvider setup
- `internal/redisx` — Redis client and module wrappers (JSON, Bloom, Search, Pub/Sub, Streams)
- `internal/repo` — in-memory fallback store and cache-aside decorator
- `internal/handlers` — HTTP handlers for Items CRUD and search
- `internal/workers` — audit stream consumer with graceful shutdown
- `internal/domain` — Item type and sentinel errors

### Request Lifecycle

```
GET /v1/items/{id}
  → chi router
  → middleware: RequestID, RealIP, Logger, Recoverer, Timeout
  → handler: HandleGetItem
  → CacheItemRepo.GetByID
    → try Redis (JSONStore.GetItem)
    → cache miss → fallback (MemoryItemRepo.GetByID)
    → populate cache (JSONStore.SetItem)
  → writeJSON → HTTP 200
```

### Two Kinds of State

| State | Lifetime | Storage | Examples |
|---|---|---|---|
| Application | server lifetime | process memory | Redis client, config, logger, template cache |
| Request | one HTTP request | *http.Request.Context | current item ID, request ID, user context |

### Topology Support

The Redis client abstracts three topologies behind the same interface:

```go
// Single-node (default)
redis.NewClient(&redis.Options{Addr: "localhost:6379"})

// Sentinel (HA failover)
redis.NewFailoverClient(&redis.FailoverOptions{
    MasterName: "mymaster",
    SentinelAddrs: []string{"sentinel-1:26379", ...},
})

// Cluster (horizontal scale)
redis.NewClusterClient(&redis.ClusterOptions{
    Addrs: []string{"node-1:6379", "node-2:6379", ...},
})
```

Every wrapper (JSON, Bloom, Search, Streams) accepts `redis.UniversalClient`—all three topologies work unchanged.

---

## Tech Stack

| Component | Technology |
|---|---|
| Language | Go 1.25+ |
| HTTP Router | chi/v5 |
| Redis Client | go-redis/v9 (supports single, Sentinel, Cluster) |
| Structured Logging | log/slog |
| Config Parsing | caarlos0/env/v10 |
| Redis Modules | redis/redis-stack (JSON, Bloom, Search, Streams) |
| Tracing | OpenTelemetry with stdout exporter |
| Testing | testcontainers-go for integration tests |
| CI | Ready for GitHub Actions |

---

## Project Structure

```text
redisforge/
├── cmd/
│   └── redisforge/
│       └── main.go                # Entrypoint and startup
├── internal/
│   ├── app/
│   │   └── app.go                 # Application lifecycle and wiring
│   ├── config/
│   │   ├── config.go              # Environment config with validation
│   │   └── config_test.go
│   ├── domain/
│   │   ├── item.go                # Item domain type
│   │   └── errors.go              # Sentinel error types
│   ├── logging/
│   │   └── logger.go              # slog setup with request context
│   ├── observability/
│   │   └── tracing_hooks.go       # OpenTelemetry initialization
│   ├── redisx/
│   │   ├── client.go              # Redis client factory (single/Sentinel/Cluster)
│   │   ├── json.go                # RedisJSON wrapper (JSON.SET, JSON.GET, etc.)
│   │   ├── json_test.go
│   │   ├── bloom.go               # RedisBloom wrapper (idempotency checks)
│   │   ├── bloom_test.go
│   │   ├── search.go              # RediSearch wrapper (full-text + faceted search)
│   │   ├── pubsub.go              # Pub/Sub wrapper (ephemeral notifications)
│   │   └── streams.go             # Redis Streams wrapper (durable async work)
│   ├── repo/
│   │   ├── item.go                # ItemRepo interface
│   │   ├── item_memory.go         # In-memory fallback implementation
│   │   ├── item_cache.go          # Cache-aside decorator
│   │   └── item_cache_test.go
│   ├── handlers/
│   │   ├── http_helpers.go        # writeJSON, readJSON, error handling
│   │   ├── health.go              # /healthz endpoint
│   │   ├── items_create.go        # POST /v1/items
│   │   ├── items_get.go           # GET /v1/items/{id}
│   │   ├── items_update.go        # PUT /v1/items/{id}
│   │   ├── items_delete.go        # DELETE /v1/items/{id}
│   │   ├── items_list.go          # GET /v1/items
│   │   ├── items_search.go        # GET /v1/items/search
│   │   └── query.go               # Query string parsing
│   └── workers/
│       └── audit_workers.go       # Audit event stream consumer
├── deployments/
│   ├── docker-compose.yml         # Single-node Redis Stack
│   ├── redis-sentinel/
│   │   └── docker-compose.yml     # Sentinel HA topology
│   └── redis-cluster/             # (Future) Cluster topology
├── docs/
│   ├── redis-decisions.md         # Bloom, RediSearch, Streams configuration choices
│   └── profiling-results.md       # Performance baseline and tuning data
├── Makefile                       # run, build, test, lint, up, down
├── go.mod
└── go.sum
```

---

## Key Components

### `CacheItemRepo` — Decorator Pattern

The `CacheItemRepo` decorates any `ItemRepo` with Redis caching using the cache-aside pattern:

```go
type CacheItemRepo struct {
    fallback ItemRepo                // In-memory or Postgres
    cache    *redisx.JSONStore       // RedisJSON documents
    logger   *slog.Logger
}

func (r *CacheItemRepo) GetByID(ctx context.Context, id string) (domain.Item, error) {
    // Try cache first
    item, err := r.cache.GetItem(ctx, id)
    if err == nil {
        return item, nil
    }
    // Cache miss → read fallback → populate cache
    item, err = r.fallback.GetByID(ctx, id)
    if err != nil {
        return domain.Item{}, err
    }
    r.cache.SetItem(ctx, item) // best-effort backfill
    return item, nil
}
```

**Why this pattern?** Fallback (Postgres in ILA) is always source of truth. Reads populate cache lazily. Writes invalidate cache (never update-in-place) to avoid consistency bugs.

### Stream Worker with Graceful Shutdown

The `AuditWorker` demonstrates durable async processing:

```go
// Start consuming audit events
worker := workers.NewAuditWorker(stream, logger, hostname)
if err := worker.Start(ctx); err != nil {
    return err
}

// Later: graceful shutdown
cancel() // signal context
worker.Stop() // wait for in-flight batches
```

The worker handles:
- `XREADGROUP` for new messages
- `XAUTOCLAIM` every 5s to reclaim entries from dead consumers (idle > 30s)
- `XACK` after successful processing
- Graceful shutdown: finishes current batch before returning

### Redis Client Abstraction

The `redisx.Open` function returns `redis.UniversalClient`, which is satisfied by:
- `redis.Client` (single-node)
- `redis.FailoverClient` (Sentinel HA)
- `redis.ClusterClient` (horizontal scale)

Every wrapper (JSON, Bloom, Search, Streams, Pub/Sub) accepts the interface. Topology selection is config-driven—no code changes needed.

---

## Redis Modules & Use Cases

### RedisJSON — Document Storage with Partial Updates

**Use case in RedisForge:** Cache whole Item objects. Enable atomic updates like "increment score by 5" without round-tripping through application code.

```go
// Store: JSON.SET item:{id} $ {"id":"...","score":9.5,...}
store.SetItem(ctx, item)

// Read: JSON.GET item:{id} $
item, _ := store.GetItem(ctx, id)

// Partial update: JSON.NUMINCRBY item:{id} $.score 5
newScore, _ := store.IncrScore(ctx, id, 5)
```

**Configuration:** No capacity limit. Memory per Item ~800–1000 bytes (depends on field count and string lengths).

---

### RedisBloom — Idempotency Checks

**Use case in RedisForge:** Fast pre-check for duplicate idempotency keys before hitting the fallback database.

```go
// Create bloom filter: BF.RESERVE bf:idempotency 0.001 1000000
// Error rate = 0.1%, capacity = 1M items, memory ≈ 1.2 MB

// Check key: BF.EXISTS bf:idempotency "request-id-123"
// Returns: false → definitely new; true → might exist (check fallback)

// After creating: BF.ADD bf:idempotency "request-id-123"
```

**Key insight:** Bloom filters have no false negatives. If `Exists` returns `false`, the key is definitely new. If it returns `true`, you must confirm in the fallback (Postgres in ILA).

**Tuning:** See [docs/redis-decisions.md](docs/redis-decisions.md) for error-rate math and when to adjust capacity.

---

### RediSearch — Full-Text & Faceted Search

**Use case in RedisForge:** Search Items by name (full-text), category (exact facet), tags (multi-value filter), and score (range).

```go
// Create index: FT.CREATE idx:items ON JSON PREFIX 1 item:{ SCHEMA ...
search.EnsureIndex(ctx)

// Queries:
// - Full-text: "widget"
// - Facet: "@category:{tools}"
// - Range: "@score:[5 10]"
// - Combined: "widget @category:{tools} @score:[5 10]"
```

**Schema design (from internal/redisx/search.go):**
- `$.name` → TEXT with weight 2.0 (boosted for relevance)
- `$.category` → TAG (exact-match facet)
- `$.tags[*]` → TAG (multi-value tag filter)
- `$.score` → NUMERIC SORTABLE

---

### Streams — Durable Async Work

**Use case in RedisForge:** Audit event log with exactly-once semantics. Every Item mutation emits an event that survives restarts and is processed by exactly one consumer.

```go
// Append event: XADD audit-events ~ 100000 event "..."
stream.Append(ctx, "audit-events", fields)

// Consumer group: XGROUP CREATE audit-events audit-processors $
stream.EnsureGroup(ctx, "audit-events", "audit-processors")

// Read new messages: XREADGROUP GROUP ... STREAMS ... >
msgs, _ := stream.ReadGroup(ctx, "audit-events", "audit-processors", consumer, 10, 2*time.Second)
for _, msg := range msgs {
    process(msg)
    msg.Ack(ctx) // XACK
}

// Claim stale: XAUTOCLAIM ... MIN-IDLE-TIME 30s
stale, _, _ := stream.ClaimStale(ctx, "audit-events", "audit-processors", consumer, 10)
```

**Configuration:**
- MAXLEN ~ 100,000 (approximate trimming)
- Idle threshold: 30 seconds (reclaim crashed consumer's work)
- Claim interval: 5 seconds (in background loop)

---

## Pub/Sub vs Streams

| Property | Pub/Sub | Streams |
|---|---|---|
| **Durability** | None — fire-and-forget | Durable — survives restarts |
| **Message loss** | Lost if no subscriber listening | Persisted until MAXLEN trim |
| **Consumer groups** | No | Yes — multiple independent consumers |
| **Message replay** | No | Yes — XRANGE, XREVRANGE |
| **Delivery guarantee** | At-most-once (lose if unlucky) | At-least-once (with ACK) |
| **Backpressure** | None | MAXLEN enforces bounded queue |
| **Use in RedisForge** | Ephemeral item-changed broadcast | Durable audit event log |
| **Use in ILA** | Real-time incident SSE feed | Audit trail, notifications, escalations |
| **Example: Item updated** | `PUBLISH items:changes "{...}"` — message lost if nobody listening | `XADD audit-events ~ event "{...}"` — survives crashes, exactly-once processing |

**When to choose:**
- **Pub/Sub:** Notifications that don't need to survive. E.g., "dashboard updated", "user came online".
- **Streams:** Work that must be done reliably. E.g., audit logs, notifications that spawn escalations, job queues.

---

## Sentinel vs Cluster

| Property | Sentinel | Cluster |
|---|---|---|
| **Problem solved** | Availability (HA failover) | Horizontal scale + HA |
| **Sharding** | No — all data on one master | Yes — 16,384 slots across masters |
| **Master/replica setup** | 1 master + N replicas (N ≥ 1) | N masters (N ≥ 3), each with replicas |
| **Multi-key operations** | Always safe | Only within same slot (use hash tags) |
| **Client library** | `redis.NewFailoverClient` | `redis.NewClusterClient` |
| **Failover time** | ~10–30 seconds (sentinel quorum delay) | ~1–2 seconds (built-in) |
| **Data consistency** | Strong (single master) | Eventual (across slots) |
| **Max data size** | Limited by single machine (typically 256 GB–1 TB) | Unlimited (scales horizontally) |
| **Complexity** | Simpler setup, more moving parts | More complex, fully distributed |
| **Use when** | Dataset fits one node (< 500 GB) | Dataset needs horizontal split (> 500 GB) or extreme throughput |
| **Use in RedisForge** | HA demo: kill master, watch reconnect | Hash-tagged multi-key ops demo |
| **Use in ILA** | Start here for HA; upgrade to Cluster if hitting capacity | Use if incident volume > 500 GB/year |

**Topology config in RedisForge:**
```env
# Single-node (default)
REDIS_ADDR=localhost:6379

# Sentinel HA
REDIS_SENTINEL_ENABLED=true
REDIS_SENTINEL_MASTER_NAME=mymaster
REDIS_SENTINEL_ADDRS=sentinel-1:26379,sentinel-2:26379,sentinel-3:26379

# Cluster
REDIS_CLUSTER_ENABLED=true
REDIS_CLUSTER_ADDRS=node-1:6379,node-2:6379,node-3:6379
```

Topology is selected at runtime—no code changes needed. Every Redis operation (JSON, Bloom, Search, Streams) works transparently on all three.

---

## Local Development

### Prerequisites

- Go 1.25+
- Docker and Docker Compose

### Setup

```bash
# Clone the repository
git clone https://github.com/SShogun/redisforge.git
cd redisforge

# Start Redis Stack (single-node)
make up

# Run the application
make run
# Output: "redisforge listening" on port 8080

# In another terminal: create an Item
curl -X POST http://localhost:8080/v1/items \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Widget Pro",
    "category": "electronics",
    "score": 9.5,
    "tags": ["bestseller", "new"],
    "idempotency_key": "req-123"
  }'

# Run tests (with race detection)
make test

# Build a binary
make build

# Clean up
make down
```

### Environment Variables

Override defaults in your shell:

```env
REDIS_ADDR=localhost:6379          # Redis server address
REDIS_PASSWORD=                     # Redis auth password (empty = no auth)
REDIS_DB=0                          # Redis database number (single-node only)
REDIS_POOL_SIZE=10                  # Connection pool size
SERVER_PORT=8080                    # HTTP server port
ENV=development                     # Environment (dev/production affects logging format)
SERVICE_VERSION=0.0.1               # Service version string (in logs and traces)

# Sentinel configuration (optional)
REDIS_SENTINEL_ENABLED=false
REDIS_SENTINEL_MASTER_NAME=mymaster
REDIS_SENTINEL_ADDRS=sentinel-1:26379,sentinel-2:26379,sentinel-3:26379

# Cluster configuration (optional)
REDIS_CLUSTER_ENABLED=false
REDIS_CLUSTER_ADDRS=node-1:6379,node-2:6379,node-3:6379
```

---

## Testing

Tests use `testcontainers-go` to spin up real `redis/redis-stack` instances:

```bash
# Run all tests with race detection
make test

# Expected output:
# ok    github.com/SShogun/redisforge/internal/config   1.793s
# ok    github.com/SShogun/redisforge/internal/redisx   11.833s
# ok    github.com/SShogun/redisforge/internal/repo     1.886s
```

### Test Coverage

| Package | Tests | Focus |
|---|---|---|
| `internal/config` | 3 | Config loading, validation, defaults |
| `internal/redisx` | 10+ | JSON, Bloom, integration with testcontainers |
| `internal/repo` | 5+ | Cache-aside decorator, memory store |

Tests are lightweight and explicit. No mocks for Redis—real containers. This proves the code works against actual Redis.

---

## Development Roadmap

All phases are complete. Here's the project structure:

| Phase | Title | Scope |
|---|---|---|
| RF-0 | Bootstrap | Entrypoint, config, logging, compose |
| RF-1 | Config & Logging | Typed env config, slog setup |
| RF-2 | Domain & Errors | Item type, sentinel errors |
| RF-3 | Redis Client | Single-node, Sentinel, Cluster support |
| RF-4 | RedisJSON | Document storage, partial updates |
| RF-5 | RedisBloom | Idempotency with configurable error rate |
| RF-6 | RediSearch | Full-text + faceted search ON JSON |
| RF-7 | Pub/Sub | Ephemeral notifications |
| RF-8 | Streams | Consumer groups, claiming, graceful shutdown |
| RF-9 | Repository Decorator | Cache-aside pattern |
| RF-10 | Stream Worker | Durable audit processing |
| RF-11 | HTTP Handlers | CRUD, search, health endpoints |
| RF-12 | App Wiring | Dependency injection, lifecycle |
| RF-13 | HA & Scale | Sentinel and Cluster drills |
| RF-14 | Observability | OpenTelemetry tracing setup |
| RF-15 | Profiling | SLOWLOG, LATENCY, MEMORY analysis |

Each phase has a corresponding section in [docs/REDISFORGE_BUILD_GUIDE.md](docs/REDISFORGE_BUILD_GUIDE.md) with implementation code, tests, and verification steps.

### Phase 4 — HTTP and app wiring

- HTTP handlers
Each phase has a corresponding section in [docs/REDISFORGE_BUILD_GUIDE.md](docs/REDISFORGE_BUILD_GUIDE.md) with implementation code, tests, and verification steps.

---

## Learning Goals

This repository is deliberately built to practice:

- **Dependency injection** in Go without frameworks
- **Request lifecycle design** (middleware → handler → storage → response)
- **Redis topology abstraction** (single-node vs Sentinel vs Cluster using the same interface)
- **Cache-aside patterns** (lazy population, invalidation on write)
- **Durable async processing** with consumer groups and exactly-once semantics
- **Graceful shutdown** (finish in-flight requests and async work before exiting)
- **Configuration as a contract** (typed, validated, environment-driven)
- **Integration testing** with real dependencies (testcontainers-go)
- **Distributed tracing** with OpenTelemetry
- **Hash-tagged key design** for multi-key operations in Cluster mode

---

## Docs

Documentation is organized by topic:

| Document | Purpose |
|---|---|
| [redis-decisions.md](docs/redis-decisions.md) | Configuration choices: Bloom error rate, RediSearch schema, Streams MAXLEN, Sentinel vs Cluster |
| [profiling-results.md](docs/profiling-results.md) | Performance baseline: SLOWLOG, LATENCY, MEMORY analysis (templates for your profiling data) |

To understand the architecture, start with the **Build Guide**. To understand why we chose specific configurations, read **redis-decisions.md**. To baseline performance, use **profiling-results.md**.

---

<p align="center">
	<sub>Built with intent, not with scaffolding.</sub>
</p>
