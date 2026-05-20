# RedisForge Build Guide

This document serves as a historical record of how RedisForge was built, phase by phase. It is intended to be used as a reference for engineers who want to understand the architectural decisions, the implementation details, and the testing strategies used in this project.

## Phase 0: Bootstrap
- Initialized the Go module (`go mod init github.com/SShogun/redisforge`).
- Set up the basic `cmd/redisforge/main.go` entrypoint.
- Created the initial `deployments/docker-compose.yml` to spin up `redis-stack`.

## Phase 1: Config & Logging
- Used `caarlos0/env/v10` for strict, typed, environment-variable-driven configuration.
- Set up `log/slog` for structured JSON logging to ensure production readiness from day one.

## Phase 2: Domain & Errors
- Defined the core `domain.Item` struct.
- Defined sentinel errors (`domain.ErrNotFound`, `domain.ErrConflict`) to decouple the storage layer from the transport layer.

## Phase 3: Redis Client Abstraction
- Created the `redisx` package.
- Utilized `redis.UniversalClient` to support Single-Node, Sentinel, and Cluster topologies transparently based on the injected configuration.

## Phase 4: RedisJSON
- Implemented `JSONStore` wrapping `JSON.SET`, `JSON.GET`, and partial updates (`JSON.NUMINCRBY`).
- Kept the client interface agnostic to the underlying Redis module.

## Phase 5: RedisBloom
- Implemented idempotency checks using `BF.RESERVE`, `BF.ADD`, and `BF.EXISTS`.
- Tuned the error rate to 0.1% to prevent unnecessary database round-trips for duplicate requests.

## Phase 6: RediSearch
- Implemented `FT.CREATE` to index `JSON` documents.
- Enabled full-text and faceted search by defining a specific schema for `Item` tags and categories.

## Phase 7: Pub/Sub
- Added ephemeral real-time notifications for system events that do not require durability.

## Phase 8: Redis Streams
- Created a robust Streams wrapper to act as an audit log.
- Implemented Consumer Groups (`XGROUP CREATE`) and guaranteed at-least-once delivery with `XACK`.

## Phase 9: Repository Decorator
- Implemented the Cache-Aside pattern.
- Created `CacheItemRepo` which decorates an underlying `ItemRepo` (e.g., PostgreSQL/Memory) with `RedisJSON` caching. 

## Phase 10: Stream Worker
- Created `AuditWorker` to consume from the audit stream.
- Implemented graceful shutdown and background `XAUTOCLAIM` loops to recover lost messages from dead consumers.

## Phase 11: HTTP Handlers
- Created the RESTful HTTP API using `chi/v5`.
- Mapped HTTP requests to the underlying caching and storage layers.

## Phase 12: App Wiring
- Wired all dependencies together in `internal/app/app.go`.
- Ensured proper context propagation and shutdown sequence.

## Phase 13: HA & Scale
- Verified Sentinel and Cluster configurations.
- Abstracted topology specifics completely away from the business logic.

## Phase 14: Observability
- Integrated OpenTelemetry hooks and Prometheus metrics.
- Added a `docker-compose` stack that provisions Grafana and Prometheus.
- Created custom dashboards for Cache Hits, Latency, and Stream processing rates.

## Phase 15: Profiling & Load Testing
- Developed `scripts/benchmark.ps1` to act as a traffic generator.
- Used the benchmark script to populate Grafana and prove the sub-millisecond efficiency of RedisJSON and RediSearch.
