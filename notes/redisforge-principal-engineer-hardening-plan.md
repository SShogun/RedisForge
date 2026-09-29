# RedisForge Hardening Plan: From Redis Demo to Principal-Level Portfolio Project

**Purpose.** This is a proposed implementation plan, not a claim that RedisForge is production-ready. The goal is one coherent, production-shaped service whose data, event, cache, and failure guarantees are explicit and demonstrated. Prefer correctness and evidence over adding more infrastructure.

## Current baseline (repository evidence)

- `internal/app/app.go` wires HTTP handlers to `CacheItemRepo` with a newly constructed `MemoryItemRepo` fallback. Item state in that map is lost when the process exits; RedisJSON is used as a cache.
- `internal/repo/item_cache.go` writes through on create, falls back on cache miss, and invalidates on update/delete. `internal/redisx/search.go` indexes RedisJSON keys; an update can therefore remove an item from search until a read repopulates the cache.
- Create uses RedisBloom as an idempotency pre-check. A positive is rejected, lookup errors fail open, and add failures are ignored. This is not a durable idempotency record.
- Audit emission is asynchronous and best effort. The stream uses approximate `MAXLEN ~ 100000`; the worker currently logs and ACKs events. It does not write to a durable audit sink.
- `/healthz` returns OK without checking dependencies. `app.Run` logs an HTTP server error from its goroutine but can continue waiting for a shutdown signal.
- CI already runs formatting, `go vet`, race tests, a Redis hot-path benchmark smoke test, and a build. Compose files use `latest` tags for key images.

These are useful teaching choices, but they should not be presented as durable production guarantees.

## Recommended target architecture

For a portfolio project that demonstrates service engineering, make **PostgreSQL the item system of record** and keep Redis central as the cache, search projection, and event-delivery technology:

```text
HTTP request
  -> Postgres transaction: item + idempotency receipt + outbox event
  -> response
Outbox relay -> Redis Stream (bounded delivery buffer)
Stream consumers -> idempotent effects + per-consumer receipts -> ACK
Outbox/projector -> versioned RedisJSON search projection -> RediSearch
GET -> Postgres on cache miss; RedisJSON remains an optimization
```

The existing `ItemRepo` boundary makes a Postgres implementation a natural extension. Keep `MemoryItemRepo` for unit tests or an explicitly labelled demo mode. Do not dual-write the item to memory and Postgres and call that highly available.

If the project must stay Redis-only, choose the other coherent design: make RedisJSON authoritative, define Redis persistence/restore and failover guarantees, and remove the in-memory fallback from the production path. Do not maintain two competing sources of truth.

## Implementation plan

### 0. Define the contract and decision record

Before adding dependencies, write short ADRs for: source of truth; idempotency semantics; audit delivery/retention; search consistency; and Redis topology/version. Define what happens for DB down, Redis down, duplicate keys, stale versions, slow consumers, malformed events, and shutdown.

**Deliverables:** a guarantee table in the README/docs and a target architecture diagram. State explicitly that delivery is at least once, not exactly once; set retention and consistency expectations rather than implying permanence or immediate consistency.

**Gate:** every user-visible guarantee has an owner, a failure behavior, and a test or operational signal.

### 1. Durable items and real idempotency

1. Add a Postgres-backed `ItemRepo`, migrations, connection/pool configuration, and readiness checks. Treat memory mode as test/demo only. Because current item data is process-local, migration of live data is not required; document that the old demo data is disposable.
2. In one DB transaction, write the item, its version/timestamps, an idempotency receipt, and the corresponding outbox event. Use database uniqueness/transactions—not a Bloom result—as the correctness boundary.
3. Store a request hash and the original response/status for each idempotency key. A retry with the same key and same request returns the original result; reuse with a different request returns a documented conflict. Define key scope and retention now; if authentication/tenancy is later added, scope keys by caller/tenant.
4. Keep Bloom only as an optional optimization/learning example. A Bloom positive must be confirmed against durable state; a Bloom outage or stale filter must not create duplicate items or reject a valid new key.

**Prove:** concurrent identical requests create one item and return a stable result; changed payload under the same key conflicts; restart does not erase the receipt; DB rollback leaves neither item nor outbox event.

### 2. Reliable outbox relay and meaningful stream worker

1. Add a bounded outbox dispatcher that claims rows in small batches, retries transient errors with bounded backoff, and publishes a stable `event_id`. Track delivery receipts per consumer; publication is not completion. Preserve per-item order by serializing an aggregate or gating version N+1 on N. Full-snapshot events may apply only a newer version; deltas require sequential versions and gap repair. Do not claim global ordering unless implemented.
2. Expect the relay to publish and crash before recording success, so duplicates can occur. Each consumer must make its effect idempotent. The audit consumer writes the audit row and its unique `(consumer,event_id)` receipt in one DB transaction, then ACKs. For the Redis search projector, apply a version-checked idempotent update first, then record its receipt; replay after a crash is safe, and rebuild/reconciliation repairs the cross-store gap.
3. Add bounded retry/attempt state and a dead-letter policy for poison messages. Persist or successfully publish the dead-letter record before ACKing/dropping the original. Keep malformed-event policy explicit and observable.
4. Revisit `MAXLEN ~ 100000`. A cap can trim payloads while consumer-group references remain, so a pending ID does not by itself guarantee recoverable event data. Keep outbox rows until every required consumer has a durable receipt and the documented retention window passes. A bounded sweeper republishes published-but-unprocessed events; stable event IDs make this safe. Alternatively, use a pinned Redis version and a retention policy proven to protect the required work. Alert on oldest unprocessed age and backlog before retention is threatened.
5. Version event envelopes (`schema_version`), bound payload size, include correlation/request IDs and item version, and avoid secrets/needless personal data.

The [transactional outbox pattern](https://docs.aws.amazon.com/prescriptive-guidance/latest/cloud-design-patterns/transactional-outbox.html) addresses the database-plus-message dual-write gap; duplicates still require idempotent consumers. Redis documents stream trimming and consumer-group reference behavior in the [XADD reference](https://redis.io/docs/latest/commands/xadd/).

**Prove:** inject crashes after DB commit, after `XADD`, after each consumer effect but before its receipt/ACK, and after ACK. Test stream trimming before consumption and per-item ordering. Show no lost committed events, safe duplicates, bounded retries, and replay through the chosen retention window.

### 3. Make cache and search projections consistent

Separate the GET cache from the searchable projection (for example, distinct Redis key prefixes). Apply committed item changes from the outbox using monotonic item versions, so an old event cannot overwrite a newer projection. Track a receipt per projection consumer; Redis updates must be replay-safe if the process crashes before the receipt commits. Represent deletes with versioned tombstones or an equivalent safe delete flow.

Define search as eventually consistent with a measured/declared projection-lag target, or choose a stronger synchronous contract and accept its write availability trade-off. Add a bounded reindex/rebuild command from Postgres and a reconciliation metric for missing/stale projections. Keep Redis failure behavior explicit: cache failure may degrade latency; search can return an actionable dependency error if no fallback scan is intended.

**Prove:** create/update/delete appear correctly in search; replaying an old event cannot revert newer data; rebuilding the index restores it; cache failure does not corrupt authoritative data.

### 4. Harden HTTP contracts and process lifecycle

- Surface listener/bind failure as a process startup failure instead of only logging it from a goroutine.
- Split liveness from readiness. Liveness should mean the process can run; readiness should reflect the dependencies required for the advertised service. Keep Redis optional for CRUD only if the implementation really degrades without it; search has different requirements.
- Define shutdown order and deadlines: stop accepting requests, finish in-flight DB transactions, stop the outbox relay, drain/stop workers, flush telemetry, then close DB/Redis. Prefer the durable outbox over untracked `EmitAsync` goroutines.
- Bound request bodies; reject trailing JSON; validate score, tags, and string sizes. Decide whether partial update is `PATCH` or whether `PUT` becomes full replacement. Expose optimistic concurrency with ETag/`If-Match` (or a clearly documented version field) rather than relying only on an internal read/update race.
- If deployed beyond local development, add authentication/authorization and rate limits before exposing item data. Keep credentials out of logs and metrics.

**Prove:** port conflict exits nonzero; readiness changes with required dependencies; shutdown leaves no untracked work; oversized/invalid/trailing-body requests fail consistently; stale `If-Match` gets the documented conflict response.

### 5. Add operational signals and reproducible environments

- Pin Redis Stack, Postgres, Prometheus, and Grafana versions (or image digests); remove `latest` from reproducibility-critical paths. Add Compose health checks and wait for required services to become healthy. See [Compose startup ordering](https://docs.docker.com/compose/how-tos/startup-order/).
- Make tracing export configurable: stdout for local use, OTLP for a collector-backed demo. Add HTTP spans and meaningful service attributes; avoid always-on full sampling as the only mode. OpenTelemetry's [Go exporter guidance](https://opentelemetry.io/docs/languages/go/exporters/) recommends a Collector in production environments.
- Add bounded-cardinality metrics for request latency/errors, DB pool pressure, cache outcome, outbox oldest age/backlog, stream lag/pending/claim outcomes, dead-letter count, and search projection lag. Never label metrics with item IDs, raw query text, or idempotency keys.
- Define SLOs from an explicit workload and deployment envelope. Do not paste invented latency/throughput goals into the README.

**Prove:** a clean checkout starts the stack deterministically; readiness gates startup; a dashboard and alert/runbook explain what an operator should do when the outbox or stream backlog grows.

### 6. Turn verification into the portfolio evidence

Extend the existing CI rather than replacing it. Pin the test service images and run migrations plus integration tests in CI. Add deterministic tests for repository contracts, transaction rollback, concurrent idempotency, stream recovery/retention, projection rebuild, and shutdown. Add fuzz targets for JSON/query parsing and validation; Go supports coverage-guided fuzzing in the standard toolchain ([Go fuzzing](https://go.dev/doc/security/fuzz/)).

Use the existing benchmark harness as a reproducible experiment: document hardware/runtime/image versions, data cardinality, warm/cold cache, request mix, concurrency, and command. Measure throughput, p50/p95/p99 latency, errors, CPU/memory, Redis memory, and projection/outbox lag. Compare baseline and change; publish raw output or a script. Do not report synthetic runs as production measurements.

**Final evidence package:** architecture + ADRs, guarantee/failure matrix, integration and failure-injection results, benchmark methodology/results, dashboard screenshot, and a short create → retry → update/search → kill worker → recover demo. Update README claims only after the evidence exists.

## Suggested order and stop gates

1. ADRs/contracts and server-startup error handling.
2. Durable system of record, idempotency receipt, migrations, and transactional outbox.
3. Idempotent consumer, retry/DLQ, and retention/replay proof.
4. Search/cache projection consistency and rebuild.
5. Readiness/shutdown, telemetry, pinned Compose, and runbooks.
6. CI failure matrix, reproducible load test, README/demo.

Do not start a later phase while the prior phase's acceptance tests are red. Do not add Kafka, Kubernetes, or more Redis modules merely for resume keywords; add them only if a measured requirement justifies the operational cost.

## Resume-level completion bar

A strong eventual claim would be: *“Designed and validated a durable Go service using Postgres transactions and Redis projections/Streams; implemented request idempotency and an outbox relay; demonstrated duplicate-safe recovery, bounded retention, readiness/shutdown behavior, and measured latency under a reproducible workload.”*

Use that wording only after the stated tests and measurements actually pass. The differentiator is the verified guarantee and trade-off reasoning, not the technology count.

## Audit record

Before the initial file write, the plan was reviewed in five improvement rounds of ten checks each: (1) repository grounding and scope, (2) data/event guarantees, (3) crash/retry/retention paths, (4) security/operations/migration, and (5) verification, benchmark quality, and resume claims. A follow-up semantic audit found that a single “published” flag did not fully express recovery across multiple consumers and stream trimming. The plan was refined to require per-consumer receipts, replayable outbox rows, and replay-safe search projection updates, then checked against another 50-point review matrix. Numeric SLOs and unmeasured performance claims remain intentionally excluded.
