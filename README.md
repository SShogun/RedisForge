# RedisForge

<p align="center">
  <a href="https://github.com/SShogun/redisforge/actions/workflows/ci.yml"><img src="https://github.com/SShogun/redisforge/actions/workflows/ci.yml/badge.svg" alt="CI" /></a>
  <img src="https://img.shields.io/badge/Go-1.25%2B-00ADD8?style=for-the-badge&logo=go&logoColor=white" alt="Go version" />
  <img src="https://img.shields.io/badge/Redis-Stack-DC382D?style=for-the-badge&logo=redis&logoColor=white" alt="Redis Stack" />
  <img src="https://img.shields.io/badge/Focus-Redis%20Architecture-2E7D32?style=for-the-badge" alt="Focus" />
</p>

RedisForge is a production-shaped Go service for learning Redis deeply.

It uses one intentionally small domain, **Items**, to make Redis architecture visible: RedisJSON, RedisBloom, RediSearch, Streams, Pub/Sub, Sentinel, Cluster-ready clients, cache-aside repositories, graceful shutdown, metrics, and integration tests with real Redis Stack containers.

If Redis feels messy, this repo is meant to be the place you reopen and revise from. If you are reviewing it from GitHub, it should read like a practical Redis field guide with code you can run.

## Why Star This

- Learn Redis patterns through working Go code, not isolated snippets.
- Compare RedisJSON, Bloom filters, RediSearch, Streams, and Pub/Sub in one service.
- See where Redis fits in a backend: config, handlers, repositories, workers, metrics, tests, and deployment files.
- Use the docs as revision notes for interviews, system design, production debugging, and future Redis projects.
- Follow the project journal and roadmap-style docs to see how a real open-source repo evolves over time.

## What You Can Learn Here

| Topic | Where to Start |
| --- | --- |
| Project tour | [docs/README.md](docs/README.md) |
| How the service is wired | [docs/implementation/architecture.md](docs/implementation/architecture.md) |
| Redis module choices | [docs/implementation/redis-patterns.md](docs/implementation/redis-patterns.md) |
| Phase-by-phase build history | [docs/REDISFORGE_BUILD_GUIDE.md](docs/REDISFORGE_BUILD_GUIDE.md) |
| Redis configuration tradeoffs | [docs/redis-decisions.md](docs/redis-decisions.md) |
| Profiling and tuning | [docs/profiling-results.md](docs/profiling-results.md) |
| Demo and social launch flow | [docs/demo_workflow.md](docs/demo_workflow.md) |
| Commit cadence and repo growth | [docs/project-journal.md](docs/project-journal.md) |

## Architecture At A Glance

```text
HTTP API
  -> chi middleware
  -> item handlers
  -> cache-aside repository
  -> RedisJSON cache
  -> in-memory fallback store

Writes also emit:
  -> Redis Streams audit event
  -> background audit worker with consumer groups

Search uses:
  -> RediSearch index over RedisJSON documents

Idempotency uses:
  -> RedisBloom pre-check before create

Operations expose:
  -> /healthz
  -> /metrics
  -> OpenTelemetry tracing hooks
```

The important design choice: the domain stays tiny so Redis remains the main thing you are studying.

## Feature Map

| Redis Feature | What RedisForge Uses It For |
| --- | --- |
| RedisJSON | Store Item documents and support partial updates |
| RedisBloom | Idempotency pre-checks with no false negatives |
| RediSearch | Full-text search, category filters, tag filters, score ranges |
| Streams | Durable audit log with consumer groups and stale-message claiming |
| Pub/Sub | Ephemeral real-time notifications |
| Sentinel | High-availability topology support |
| Cluster client | Horizontal-scale topology support and hash-tag discipline |
| SLOWLOG/LATENCY/MEMORY | Profiling drills and production-style tuning notes |

## Project Structure

```text
redisforge/
|-- cmd/redisforge/              # application entrypoint
|-- internal/
|   |-- app/                     # dependency wiring and lifecycle
|   |-- config/                  # typed environment configuration
|   |-- domain/                  # Item model and sentinel errors
|   |-- handlers/                # HTTP handlers for CRUD and search
|   |-- logging/                 # slog setup
|   |-- observability/           # metrics and tracing hooks
|   |-- redisx/                  # Redis client plus JSON/Bloom/Search/Streams wrappers
|   |-- repo/                    # in-memory store and cache-aside decorator
|   `-- workers/                 # audit stream worker
|-- deployments/
|   |-- docker-compose.yml       # app + Redis Stack + monitoring stack
|   |-- prometheus/              # Prometheus config
|   |-- grafana/                 # Grafana provisioning and dashboards
|   `-- redis-sentinel/          # Sentinel HA demo topology
|-- docs/
|   |-- implementation/          # architecture and Redis pattern notes
|   |-- README.md                # docs index and learning path
|   |-- REDISFORGE_BUILD_GUIDE.md
|   |-- redis-decisions.md
|   |-- profiling-results.md
|   |-- demo_workflow.md
|   `-- project-journal.md
|-- scripts/                     # benchmark and demo scripts
|-- Makefile                     # Linux/macOS tasks
`-- make.ps1                     # Windows tasks
```

## Quick Start

Prerequisites:

- Go 1.25+
- Docker Desktop
- PowerShell on Windows or make on Linux/macOS

Start the app, Redis Stack, Prometheus, and Grafana:

```powershell
.\make.ps1 up
```

The API is now available on `http://localhost:8080`.

Create an item:

```powershell
Invoke-RestMethod -Uri "http://localhost:8080/v1/items" -Method Post -ContentType "application/json" -Body '{
  "name": "Widget Pro",
  "category": "electronics",
  "score": 9.5,
  "tags": ["bestseller", "new"],
  "idempotency_key": "req-123"
}'
```

Search items:

```powershell
Invoke-RestMethod "http://localhost:8080/v1/items/search?q=Widget"
```

Check health and metrics:

```powershell
Invoke-RestMethod "http://localhost:8080/healthz"
Invoke-RestMethod "http://localhost:8080/metrics"
```

Stop the stack:

```powershell
.\make.ps1 down
```

Linux/macOS equivalents:

```bash
make up
make test
make down
```

## API Surface

| Method | Path | Purpose |
| --- | --- | --- |
| GET | `/healthz` | Health check |
| GET | `/metrics` | Prometheus metrics |
| POST | `/v1/items` | Create an item and emit an audit event |
| GET | `/v1/items` | List items |
| GET | `/v1/items/{id}` | Fetch one item through cache-aside lookup |
| PUT | `/v1/items/{id}` | Update item and refresh Redis state |
| DELETE | `/v1/items/{id}` | Delete item |
| GET | `/v1/items/search?q=...` | Search through RediSearch |

## Tests

```powershell
.\make.ps1 test
```

The tests use `testcontainers-go` where Redis behavior matters, so the Redis wrappers are validated against real Redis Stack instead of mocks.

## Development Rhythm

This repo is intended to grow in public. Good future commits are small, reviewable, and educational:

- Add one Redis lesson at a time.
- Add a failing test before fixing subtle behavior.
- Improve one doc after each implementation change.
- Record benchmark numbers when performance claims change.
- Keep [docs/project-journal.md](docs/project-journal.md) updated so visitors can see progress over time.

Suggested commit style:

```text
docs: add redis streams recovery notes
test: cover bloom duplicate idempotency path
feat: expose cache hit ratio metric
perf: record search latency baseline
```

## Status

RedisForge currently implements the planned learning phases from bootstrap through profiling notes:

- HTTP API, config, logging, graceful shutdown
- Redis client abstraction for single-node, Sentinel, and Cluster modes
- RedisJSON, RedisBloom, RediSearch, Streams, and Pub/Sub wrappers
- Cache-aside repository pattern
- Audit stream worker
- Prometheus metrics and OpenTelemetry hooks
- Integration tests and benchmark/demo scripts

## Repository Goal

RedisForge should be useful in two modes:

1. **Revision mode:** reopen the repo when Redis concepts feel scattered, then follow the docs and implementation notes until the system clicks again.
2. **Visitor mode:** land on GitHub, understand the value in under a minute, run the project quickly, and save or star it as a Redis learning reference.

That is the bar for every future change.
