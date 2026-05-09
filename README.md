# RedisForge

<p align="center">
	<img src="https://img.shields.io/badge/Go-1.22%2B-00ADD8?style=for-the-badge&logo=go&logoColor=white" alt="Go version" />
	<img src="https://img.shields.io/badge/Redis-Stack-DC382D?style=for-the-badge&logo=redis&logoColor=white" alt="Redis Stack" />
	<img src="https://img.shields.io/badge/Status-Bootstrap%20Stage-2E7D32?style=for-the-badge" alt="Status" />
</p>

RedisForge is a production-shaped Go service for learning Redis patterns through one small domain: Items. The codebase begins as a bootstrap app and is designed to grow into a Redis-first service that exercises RedisJSON, RedisBloom, RediSearch, Streams, Pub/Sub, Sentinel, and Cluster topologies without hiding the Redis decisions behind a large business domain.

The project currently contains the bootstrap foundation:

- A runnable Go entrypoint that prints a startup message.
- Environment-driven config loading with sane defaults.
- Structured logging built on `slog`.
- A Redis Stack compose file for local development.
- Tests for the config package.

The full build recipe lives in [docs/REDISFORGE_BUILD_GUIDE.md](docs/REDISFORGE_BUILD_GUIDE.md). That guide is the implementation map for the next phases of the project.

---

## Table of Contents

- [Why This Project Exists](#why-this-project-exists)
- [Product Vision](#product-vision)
- [Current Status](#current-status)
- [Architecture](#architecture)
- [Tech Stack](#tech-stack)
- [Project Structure](#project-structure)
- [Key Components](#key-components)
- [Local Development](#local-development)
- [Testing](#testing)
- [Development Roadmap](#development-roadmap)
- [Docs](#docs)

---

## Why This Project Exists

RedisForge is meant to teach the part of backend engineering that simple JSON APIs let you postpone:

- Redis module selection and tradeoffs.
- Cache-aside repository design.
- Document storage and partial-path updates.
- Probabilistic idempotency checks.
- Durable async processing with Streams.
- Ephemeral notifications with Pub/Sub.
- Failover and cluster-aware key design.

The goal is to keep the domain small enough that the Redis architecture stays visible instead of being buried under application noise.

---

## Product Vision

RedisForge is an internal-style Redis systems playground for one domain: Items.

The intended flow is:

1. A request enters the service.
2. Configuration and logging are already wired.
3. Redis is used to support caching, search, and async work.
4. The application grows into explicit support for RedisJSON, Bloom filters, search indexes, pub/sub, and streams.
5. The same service is then used to demonstrate Sentinel and Cluster behavior.

This is not a public SaaS app and not a toy CRUD sample. It is a compact backend exercise that stays close to production patterns.

---

## Current Status

> **Bootstrap stage**

The repository currently has the minimum foundation in place and is ready for the next wiring phase.

| Area | Status |
|---|---|
| Entrypoint | Complete |
| Config loading | Complete |
| Structured logging | Complete |
| Local Redis Stack compose file | Complete |
| Redis client layer | Planned |
| RedisJSON wrapper | Planned |
| RedisBloom wrapper | Planned |
| RediSearch wrapper | Planned |
| Pub/Sub and Streams wrappers | Planned |
| Repository layer | Planned |
| HTTP handlers | Planned |
| App wiring | Planned |

The authoritative implementation plan for the remaining work is in [docs/REDISFORGE_BUILD_GUIDE.md](docs/REDISFORGE_BUILD_GUIDE.md).

---

## Architecture

The current architecture is intentionally small:

- `cmd/redisforge/main.go` starts the program.
- `internal/config` loads typed runtime configuration from environment variables.
- `internal/logging` builds the structured application logger.
- `deployments/docker-compose.yml` starts Redis Stack for local development.

The planned architecture follows a consistent seam pattern:

- `internal/redisx` owns Redis client setup and module-specific wrappers.
- `internal/repo` owns the storage abstraction and cache decorator.
- `internal/workers` owns long-running stream consumers.
- `internal/handlers` owns HTTP request handling.
- `internal/app` wires the whole application together.

The build guide documents that shape in detail and explains the reasoning behind each layer.

---

## Tech Stack

- Go 1.22
- `github.com/caarlos0/env/v10` for config parsing
- `log/slog` for structured logging
- Redis Stack for local Redis module development
- Docker Compose for local infrastructure

---

## Project Structure

```text
RedisForge/
├── cmd/
│   └── redisforge/
│       └── main.go              # Entrypoint and startup message
├── deployments/
│   └── docker-compose.yml       # Local Redis Stack environment
├── docs/
│   └── REDISFORGE_BUILD_GUIDE.md # Full implementation recipe
├── internal/
│   ├── config/
│   │   ├── config.go            # Typed config loading and validation
│   │   └── config_test.go       # Config tests
│   └── logging/
│       └── logger.go            # Structured logger setup
├── Makefile                     # Run, build, test, and compose targets
├── go.mod
└── go.sum
```

This tree reflects the current repository state. The build guide describes the future `internal/redisx`, `internal/repo`, `internal/workers`, `internal/handlers`, and `internal/app` packages that will be added next.

---

## Key Components

### `cmd/redisforge/main.go`

The current entrypoint only proves the application can start. It is intentionally small so the wiring steps stay easy to follow.

### `internal/config`

This package owns all runtime configuration. It provides defaults for tests and validates required values at startup.

### `internal/logging`

This package creates the application logger and keeps request-scoped logging helpers ready for later middleware integration.

### `deployments/docker-compose.yml`

This compose file starts Redis Stack locally so the project can be developed against the same module set used in the build guide.

### `docs/REDISFORGE_BUILD_GUIDE.md`

This is the main project recipe. It explains the intended domain model, Redis module usage, repository pattern, stream worker model, and deployment drills.

---

## Local Development

### Requirements

- Go 1.22+
- Docker and Docker Compose

### Start Redis Stack

```bash
make up
```

### Run the application

```bash
make run
```

### Build a binary

```bash
make build
```

### Run tests

```bash
make test
```

### Optional linting

```bash
make lint
```

### Environment variables

The current defaults are:

```text
REDIS_ADDR=localhost:6379
REDIS_PASSWORD=
REDIS_DB=0
REDIS_POOL_SIZE=10
SERVER_PORT=8080
ENV=development
SERVICE_VERSION=0.0.1
```

---

## Testing

The repository already includes config tests. Run the full suite with:

```bash
make test
```

As RedisForge grows, the testing strategy in the build guide adds container-backed integration tests for RedisJSON, RedisBloom, RediSearch, and Streams.

---

## Development Roadmap

### Phase 0 — Bootstrap

- Runnable Go entrypoint
- Config loading and validation
- Structured logging
- Redis Stack compose file

### Phase 1 — Redis client wiring

- Open a configured Redis client
- Support single-node, Sentinel, and Cluster topologies
- Add tracing hooks

### Phase 2 — Redis module wrappers

- RedisJSON document storage
- RedisBloom idempotency checks
- RediSearch indexing and search
- Pub/Sub and Streams wrappers

### Phase 3 — Repository and workers

- Cache-aside repository decorator
- In-memory fallback store
- Redis Streams audit worker

### Phase 4 — HTTP and app wiring

- HTTP handlers
- Router composition
- Graceful shutdown

### Phase 5 — Hardening and observability

- Sentinel failover drill
- Cluster-aware key design
- OpenTelemetry and profiling drills

---

## Docs

- [Build Guide](docs/REDISFORGE_BUILD_GUIDE.md) - full implementation recipe and phase plan

---

<p align="center">
	<sub>Built with intent, not with scaffolding.</sub>
</p>
