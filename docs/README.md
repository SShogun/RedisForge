# RedisForge Docs

This folder is the RedisForge learning path. Use it when you want to revise Redis concepts, understand the implementation, or decide what to build next.

## Recommended Reading Order

1. [Implementation Architecture](implementation/architecture.md)
   Start here to understand how the HTTP API, repositories, Redis wrappers, workers, metrics, and config fit together.

2. [Redis Patterns](implementation/redis-patterns.md)
   Read this when you want the Redis-specific lessons: JSON, Bloom, Search, Streams, Pub/Sub, Sentinel, and Cluster.

3. [Build Guide](REDISFORGE_BUILD_GUIDE.md)
   Historical phase-by-phase record of how the project was built.

4. [Redis Decisions](redis-decisions.md)
   Deeper notes on configuration choices and tradeoffs.

5. [Profiling Results](profiling-results.md)
   Performance and tuning workspace for SLOWLOG, LATENCY, MEMORY, and benchmarks.

6. [Demo Workflow](demo_workflow.md)
   A short repeatable flow for recording demos and proving the project works.

7. [Project Journal](project-journal.md)
   Commit cadence, next public updates, and the habit loop that keeps the repo looking alive.

8. [Interview Questions](interview_questions.md) and [Interview Answers](redisforge_interview_answers.md)
   52 backend interview questions with RedisForge-specific answers. Use for revision.

## Docs By Intent

| Intent | Read |
| --- | --- |
| "I forgot how Redis Streams consumer groups work." | [implementation/redis-patterns.md](implementation/redis-patterns.md) |
| "I want to understand the codebase fast." | [implementation/architecture.md](implementation/architecture.md) |
| "I want to know why these Redis settings exist." | [redis-decisions.md](redis-decisions.md) |
| "I want to record a demo for GitHub/LinkedIn/X." | [demo_workflow.md](demo_workflow.md) and [social_templates.md](social_templates.md) |
| "I want to keep making real commits." | [project-journal.md](project-journal.md) |
| "I am preparing for Redis/backend interviews." | [interview_questions.md](interview_questions.md) and [redisforge_interview_answers.md](redisforge_interview_answers.md) |
| "I want to try Redis Cluster locally." | [../deployments/redis-cluster/README.md](../deployments/redis-cluster/README.md) |

## Feature Coverage Matrix

Every meaningful feature should leave behind three things: **code**, a **test**, and a **doc note**. This matrix tracks coverage.

| Feature | Code | Test | Doc Note |
| --- | --- | --- | --- |
| RedisJSON (CRUD + partial updates) | `redisx/json.go` | `redisx/json_test.go` | `redis-patterns.md` ✅ |
| RedisBloom (idempotency) | `redisx/bloom.go` | `redisx/bloom_test.go` + `handlers/items_create_test.go` | `redis-patterns.md` ✅ |
| RediSearch (full-text + faceted) | `redisx/search.go` | ⚠️ needs dedicated test | `redis-patterns.md` ✅ |
| Redis Streams (audit log) | `redisx/streams.go` + `workers/audit_workers.go` | ⚠️ needs dedicated test | `redis-patterns.md` ✅ |
| Pub/Sub (ephemeral notifications) | `redisx/pubsub.go` | ⚠️ needs dedicated test | `redis-patterns.md` ✅ |
| Cache-Aside Repository | `repo/item_cache.go` | `repo/item_cache_test.go` (4 tests + 4 benchmarks) | `redis-patterns.md` ✅ |
| Sentinel HA | `redisx/client.go` | Manual via `redis-sentinel/` compose | `redis-decisions.md` ✅ |
| Cluster (horizontal scale) | `redisx/client.go` | Manual via `redis-cluster/` compose | `redis-cluster/README.md` ✅ |
| Prometheus Metrics | `observability/metrics.go` | Verified via `/metrics` endpoint | `profiling-results.md` ✅ |
| OpenTelemetry Tracing | `observability/tracing_hooks.go` | Verified via trace output | `architecture.md` ✅ |
| Graceful Shutdown | `app/app.go` | Manual verification | `architecture.md` ✅ |
| Handler Idempotency (E2E) | `handlers/items_create.go` | `handlers/items_create_test.go` ✅ | `redis-patterns.md` ✅ |

> **Legend:** ✅ = covered, ⚠️ = gap identified for future commit

## Documentation Standard

Every meaningful feature should leave behind three things:

- Code that demonstrates the behavior.
- A test, benchmark, or manual verification step.
- A doc note explaining why the Redis choice was made.

This keeps RedisForge useful for both the maintainer and future visitors.

