# RedisForge Hardening Plan: Agent-Executable Milestones

**Purpose.** Turn RedisForge from a strong Redis learning/demo repository into a production-shaped portfolio service by improving one bounded, reviewable unit at a time.

This file is the **execution queue**. The architectural baseline and non-negotiable constraints live in [architecture-limitations.md](architecture-limitations.md). An automated agent must read both files before changing code.

The target remains intentionally focused:

```text
HTTP request
  -> PostgreSQL transaction
       -> item mutation
       -> durable idempotency receipt when applicable
       -> outbox event
  -> response

Outbox relay -> Redis Stream
  -> audit consumer -> durable audit effect + receipt -> ACK
  -> search projector -> version-checked RedisJSON projection -> receipt -> ACK

GET -> Redis cache when valid -> PostgreSQL on miss
Search -> RediSearch over versioned RedisJSON projection
```

Redis remains central to the project, but it is no longer asked to pretend that an in-memory fallback is durable state.

---

## Execution policy

### One daily unit

A scheduled agent run may **implement at most one task ID** from this file.

The agent should:

1. Read [architecture-limitations.md](architecture-limitations.md) and this file.
2. Inspect the current `master` branch and recent history.
3. Starting from the first task in queue order, evaluate whether its acceptance criteria are already satisfied by the current repository.
4. Skip already-satisfied tasks, recording the evidence used.
5. Select the first incomplete task whose dependencies are satisfied.
6. Implement only that task.
7. Run targeted verification, then the applicable repository verification baseline.
8. Create a local commit only when there is no known failing verification.
9. Stop and produce the required report.

The agent must **not** continue into the next task during the same scheduled run, even if the selected task finishes quickly.

### Branch naming

Use:

```text
agentzero/redisforge/<task-id-lowercase>-YYYYMMDD
```

Example:

```text
agentzero/redisforge/rf-m1-p02-20261008
```

Base the branch on the current reviewed `master`. Do not stack a new daily task on an unreviewed prior Agent Zero branch.

### Local commit policy

Default behavior:

- create a **local commit** after implementation and verification;
- never push;
- never merge;
- never force-update a branch;
- never create a release or deployment.

If targeted verification fails because of the implementation, do **not** create the commit. Leave the branch/worktree for inspection and report the failure.

If a CI-equivalent check cannot run solely because of a documented local environment limitation (for example Docker unavailable), the agent may still create the local commit if all runnable targeted checks pass, but it must mark that check `UNVERIFIED`. Remote CI remains the merge gate.

### Definition of complete

A task is complete only when:

- its acceptance criteria are demonstrably satisfied;
- directly relevant tests exist and pass;
- no known regression remains;
- changed behavior is documented where necessary;
- applicable verification is green or explicitly `UNVERIFIED` only for environment limitations;
- the final report contains exact evidence.

"Code was written" is not completion.

---

# Milestone RF-M0 — Contracts and Safe Process Behavior

Goal: remove ambiguity before introducing a durable store. This milestone establishes explicit guarantees and fixes lifecycle/API behavior that should not depend on PostgreSQL.

## RF-M0-P01 — Guarantee matrix and ADR baseline

**Dependencies:** none

**Scope**

- Create or update concise ADR/decision documentation for:
  - source of truth;
  - idempotency semantics;
  - audit delivery/retention;
  - search consistency;
  - Redis topology/module requirements.
- Add a user-visible guarantee/failure matrix linked from the README or docs index.
- Keep claims consistent with the current code; target-state guarantees must be clearly labelled as planned.

**Acceptance**

- Every advertised guarantee names its owner, failure behavior, and evidence/test strategy.
- Current behavior is not described as durable if it is not durable.
- At-least-once is not described as exactly-once.
- No invented latency/SLO number is introduced.

**Verification**

- Review affected Markdown links.
- Search README/docs for contradictory claims about durability, audit permanence, search consistency, and idempotency.
- Run repository code checks only if code/config changed.

## RF-M0-P02 — Propagate HTTP server startup/runtime failure

**Dependencies:** RF-M0-P01

**Scope**

- Change application lifecycle wiring so listener/bind/start failure reaches the top-level runner.
- Preserve graceful shutdown for normal termination.
- Add focused tests for immediate server failure and normal cancellation/shutdown.

**Acceptance**

- A port/listener failure causes a nonzero application failure path rather than being logged and ignored.
- Normal shutdown remains bounded and clean.
- No unrelated server redesign.

**Verification**

```bash
go test ./internal/app/... -count=1
go vet ./...
go test -race -count=1 ./...
go build ./cmd/redisforge
```

## RF-M0-P03 — Harden JSON request boundaries

**Dependencies:** RF-M0-P02

**Scope**

- Bound request body size.
- Reject trailing JSON/tokens.
- Make malformed JSON behavior deterministic.
- Add/strengthen validation for existing Item fields without inventing new product semantics.

**Acceptance**

- Oversized bodies fail with the documented status.
- Multiple JSON values/trailing tokens fail.
- Existing valid payloads still work.
- Tests cover boundary cases.

**Verification**

```bash
go test ./internal/handlers/... -count=1
go vet ./...
go test -race -count=1 ./...
go build ./cmd/redisforge
```

## RF-M0-P04 — Document update semantics before persistence migration

**Dependencies:** RF-M0-P03

**Scope**

- Decide and document whether existing update behavior is full replacement or partial update.
- Define the intended optimistic-concurrency contract using item version and HTTP `ETag`/`If-Match` or an explicitly justified equivalent.
- Add tests only for behavior that already exists; implementation of durable concurrency may occur in RF-M1.

**Acceptance**

- API semantics are unambiguous before the Postgres repository is introduced.
- The document identifies the exact conflict response expected once durable concurrency is implemented.

**Verification**

- Documentation/link review.
- No broad code change is required for this task.

---

# Milestone RF-M1 — Durable Items, Idempotency, and Transactional Outbox

Goal: make PostgreSQL the authoritative item store and establish one transactional correctness boundary.

## RF-M1-P01 — Add PostgreSQL runtime and migration foundation

**Dependencies:** RF-M0 complete

**Scope**

- Add a PostgreSQL driver/pool appropriate for the project.
- Add typed DB configuration.
- Add migrations for the initial durable schema:
  - items;
  - idempotency receipts;
  - outbox events.
- Add Postgres to local Compose with a pinned version and health check.
- Do not wire handlers to Postgres yet.

**Acceptance**

- Clean environment can start Postgres deterministically.
- Migrations apply from empty state and are repeatable according to the chosen migration tool.
- Secrets/credentials have safe local defaults and are not logged.

**Verification**

- Migration-specific tests/commands.
- `go mod tidy` because dependencies change.
- `go vet ./...`
- `go test -race -count=1 ./...`
- `go build ./cmd/redisforge`

## RF-M1-P02 — Implement PostgreSQL ItemRepo CRUD

**Dependencies:** RF-M1-P01

**Scope**

- Implement the existing `ItemRepo` contract using PostgreSQL.
- Make version/timestamp ownership explicit in the durable layer.
- Add repository contract/integration tests for create/get/list/update/delete.
- Do not change handler semantics beyond what the repository boundary requires.

**Acceptance**

- Item state survives process/repository reconstruction.
- CRUD semantics match the documented contract.
- Update/delete not-found behavior is deterministic.
- Integration tests use real PostgreSQL.

**Verification**

```bash
go test ./internal/repo/... -count=1
go vet ./...
go test -race -count=1 ./...
go build ./cmd/redisforge
```

## RF-M1-P03 — Wire Postgres as production source of truth

**Dependencies:** RF-M1-P02

**Scope**

- Production application wiring uses Postgres-backed `ItemRepo`.
- `MemoryItemRepo` remains available only for tests or explicit demo/unit usage.
- Remove any production-path implication that RedisJSON or memory is authoritative.
- GET cache miss must resolve from durable state.

**Acceptance**

- Restarting the app does not erase authoritative item data.
- GET/List agree after restart.
- Cache failure does not corrupt durable state.
- Existing Redis learning features remain available where appropriate.

**Verification**

- App/repository integration tests.
- Restart-oriented integration test if practical.
- CI-equivalent verification.

## RF-M1-P04 — Implement durable idempotency receipts

**Dependencies:** RF-M1-P03

**Scope**

- Store idempotency key scope, request hash, original status/result reference, and retention metadata.
- Define same-key/same-request replay behavior.
- Define same-key/different-request conflict behavior.
- Bloom may remain only as an optimization; it cannot be the correctness boundary.

**Acceptance**

- Same key + same request can return a stable prior result.
- Same key + different request returns the documented conflict.
- Receipt survives restart.
- Bloom outage/false positive cannot incorrectly decide correctness.

**Verification**

- Focused idempotency repository/service tests.
- Restart test.
- CI-equivalent verification.

## RF-M1-P05 — Make create + receipt + outbox one transaction

**Dependencies:** RF-M1-P04

**Scope**

- For create requests, atomically write:
  - item;
  - idempotency receipt when supplied;
  - outbox event with stable `event_id`, aggregate/item ID, item version, schema version, correlation/request ID where available.
- Eliminate best-effort asynchronous audit emission from the create correctness path.

**Acceptance**

- Transaction rollback leaves no partial item/receipt/outbox state.
- Successful commit always has the corresponding outbox row.
- Duplicate request concurrency creates one logical item/result.
- No direct DB + Redis dual write is introduced.

**Verification**

- Transaction rollback test.
- Concurrent identical-request test.
- Same-key/different-payload test.
- CI-equivalent verification.

## RF-M1-P06 — Extend transactional outbox to update/delete and durable concurrency

**Dependencies:** RF-M1-P05

**Scope**

- Update/delete create outbox events in the same item transaction.
- Implement the previously documented optimistic-concurrency contract.
- Ensure item versions advance monotonically.
- Define delete version/tombstone information needed by projections.

**Acceptance**

- Stale update/delete is rejected deterministically.
- Successful mutation and outbox event are atomic.
- Failed mutation produces no outbox event.
- Tests prove version monotonicity and stale-write conflict.

**Verification**

- Handler + repository integration tests.
- Concurrency tests.
- CI-equivalent verification.

**RF-M1 gate:** production CRUD is durable, idempotency correctness is transactional, and every committed mutation has a durable outbox event.

---

# Milestone RF-M2 — Reliable Outbox Relay and Idempotent Consumers

Goal: make event delivery recoverable and duplicate-safe without claiming exactly-once behavior.

## RF-M2-P01 — Add bounded outbox relay

**Dependencies:** RF-M1 complete

**Scope**

- Claim outbox rows in bounded batches.
- Publish stable event envelopes to Redis Streams.
- Use bounded retry/backoff.
- Preserve enough state to recover after process restart.
- Do not mark consumer completion at publication time.

**Acceptance**

- Relay does not lose a committed outbox row because Redis is temporarily unavailable.
- Batch size/backoff are bounded/configurable.
- Stable `event_id` is preserved on retry.

**Verification**

- Relay unit/integration tests including Redis unavailable then recovery.
- CI-equivalent verification.

## RF-M2-P02 — Prove publish/crash duplicate safety

**Dependencies:** RF-M2-P01

**Scope**

- Handle the crash window after successful `XADD` but before publisher state is recorded.
- Add failure injection around publication bookkeeping.
- Keep duplicates legal and observable.

**Acceptance**

- The same outbox event can be republished after crash without corrupting downstream state.
- Tests demonstrate the duplicate window instead of hiding it.
- Documentation states at-least-once delivery.

**Verification**

- Failure-injection integration tests.
- CI-equivalent verification.

## RF-M2-P03 — Persist audit effect and per-consumer receipt

**Dependencies:** RF-M2-P02

**Scope**

- Replace log-only audit consumption with a durable audit record/effect.
- Store a unique per-consumer `event_id` receipt in the same DB transaction as the audit effect.
- ACK only after the durable transaction succeeds.

**Acceptance**

- Re-delivery does not duplicate the durable audit effect.
- Crash after effect/receipt but before ACK is safe.
- ACK failure leaves a recoverable message.

**Verification**

- Duplicate-delivery test.
- Crash-before-ACK test.
- Existing stream recovery tests remain green.
- CI-equivalent verification.

## RF-M2-P04 — Add bounded retry and dead-letter policy

**Dependencies:** RF-M2-P03

**Scope**

- Track attempts/retry state for poison/malformed events.
- Define dead-letter persistence/publication.
- Persist or successfully publish dead-letter evidence before dropping/ACKing the original.
- Add bounded-cardinality metrics for retry/dead-letter outcomes.

**Acceptance**

- Poison event cannot loop forever without visibility.
- Original event is not ACKed before dead-letter handling is durable enough for the documented contract.
- Metrics/logs identify the failure category without high-cardinality IDs as labels.

**Verification**

- Malformed-event and repeated-failure tests.
- Metrics tests where practical.
- CI-equivalent verification.

## RF-M2-P05 — Protect retention and replay window

**Dependencies:** RF-M2-P04

**Scope**

- Revisit stream `MAXLEN` behavior.
- Keep outbox rows long enough for every required consumer receipt plus the documented retention window.
- Add bounded replay/sweeper behavior for published-but-unprocessed events.
- Add oldest-unprocessed/backlog visibility.

**Acceptance**

- A trimmed stream payload alone cannot permanently lose a committed event while its durable replay window is still promised.
- Sweeper/replay uses stable event IDs and remains duplicate-safe.
- Retention assumptions are documented.

**Verification**

- Trim-before-consumption test.
- Replay-after-trim test.
- Slow-consumer/backlog test where practical.
- CI-equivalent verification.

## RF-M2-P06 — Full event failure matrix

**Dependencies:** RF-M2-P05

**Scope**

Add explicit failure-injection coverage for:

- crash after DB commit;
- crash after `XADD`;
- crash after consumer effect before receipt/ACK;
- crash after receipt before ACK;
- Redis outage and recovery;
- stream trimming before first consumption;
- stale pending claim/recovery;
- malformed/poison event.

**Acceptance**

- Tests demonstrate no lost committed event inside the documented retention model.
- Duplicates remain safe.
- Ordering claims are limited to what is actually implemented.
- The failure-semantics documentation matches test evidence.

**Verification**

- Dedicated failure-matrix test suite.
- Full CI-equivalent verification.

**RF-M2 gate:** committed outbox events remain recoverable through the documented window, consumers are idempotent, and failure behavior is tested rather than implied.

---

# Milestone RF-M3 — Versioned Cache and Search Projection

Goal: make Redis cache/search behavior consistent with durable item state while keeping search explicitly eventually consistent.

## RF-M3-P01 — Separate GET cache from search projection

**Dependencies:** RF-M2 complete

**Scope**

- Use distinct Redis keyspaces/prefixes for read cache and search projection.
- Remove the current coupling where cache invalidation can make search disappear.
- Document ownership and TTL/retention of each keyspace.

**Acceptance**

- Updating/invalidating GET cache cannot remove the searchable projection by accident.
- Existing search index definitions target the projection namespace.

**Verification**

- Redis integration tests.
- CI-equivalent verification.

## RF-M3-P02 — Add version-checked search projector

**Dependencies:** RF-M3-P01

**Scope**

- Consume outbox/stream item events.
- Apply create/update projection only when event version is newer than the stored projection version.
- Record projection consumer receipt after a replay-safe Redis update.

**Acceptance**

- Duplicate event is harmless.
- Older event cannot overwrite newer projection state.
- Crash after Redis projection write but before receipt is safe on replay.

**Verification**

- Ordered, duplicate, out-of-order, and crash-window tests.
- CI-equivalent verification.

## RF-M3-P03 — Make delete projection replay-safe

**Dependencies:** RF-M3-P02

**Scope**

- Implement versioned tombstone or equivalent delete protection.
- Ensure an old create/update event cannot resurrect a newer delete.
- Bound tombstone retention according to replay guarantees.

**Acceptance**

- Delete remains authoritative over older replayed events.
- Newer legitimate recreate semantics, if supported, are explicit and version-safe.

**Verification**

- Delete/replay/out-of-order integration tests.
- CI-equivalent verification.

## RF-M3-P04 — Add bounded rebuild/reindex command

**Dependencies:** RF-M3-P03

**Scope**

- Rebuild search projection from PostgreSQL in bounded pages/batches.
- Make reruns safe.
- Avoid loading all items into memory.
- Document operational usage.

**Acceptance**

- Empty/corrupt/missing projection can be restored from durable data.
- Re-running rebuild converges without duplicating/corrupting state.

**Verification**

- Rebuild integration test from intentionally damaged projection.
- CI-equivalent verification.

## RF-M3-P05 — Add projection reconciliation and explicit search failure semantics

**Dependencies:** RF-M3-P04

**Scope**

- Add bounded reconciliation signal/metric for missing or stale projection data.
- Define search behavior when Redis/Search is unavailable.
- Measure/report projection lag without high-cardinality labels.

**Acceptance**

- Operators can detect stale/missing projection.
- Search dependency failure returns a documented actionable error rather than silently pretending authoritative completeness.
- CRUD remains correct because Postgres is authoritative.

**Verification**

- Redis unavailable tests for CRUD vs search.
- Metric/reconciliation tests.
- CI-equivalent verification.

**RF-M3 gate:** cache is an optimization, search is a replayable versioned projection, and Redis failure cannot corrupt authoritative item state.

---

# Milestone RF-M4 — Readiness, Shutdown, Reproducibility, and Observability

Goal: make the service operable and make failures diagnosable.

## RF-M4-P01 — Split liveness and readiness

**Dependencies:** RF-M3 complete

**Scope**

- Keep liveness about process viability.
- Add readiness that probes dependencies required for the advertised service.
- Reflect Postgres and Redis/Search requirements separately where semantics differ.

**Acceptance**

- Liveness does not flap solely because Redis is unavailable.
- Readiness becomes false when a required dependency is unavailable.
- Tests cover dependency transitions.

**Verification**

- Handler/app tests.
- Integration test with dependency outage.
- CI-equivalent verification.

## RF-M4-P02 — Coordinate shutdown across HTTP, relay, workers, telemetry, and stores

**Dependencies:** RF-M4-P01

**Scope**

Define and implement bounded shutdown order:

1. stop accepting new requests;
2. finish/cancel in-flight request work by deadline;
3. stop outbox relay;
4. drain/stop stream consumers;
5. flush telemetry;
6. close Redis/Postgres.

**Acceptance**

- No new untracked async audit goroutines remain.
- Shutdown has explicit deadlines and returns errors when shutdown fails.
- Tests prove bounded exit.

**Verification**

- Lifecycle tests including cancellation and stuck worker simulation.
- `go test -race`.
- CI-equivalent verification.

## RF-M4-P03 — Pin reproducible service images and health-gated Compose startup

**Dependencies:** RF-M4-P02

**Scope**

- Pin Redis Stack, Postgres, Prometheus, and Grafana versions/digests where appropriate.
- Add Compose health checks.
- Use dependency health ordering instead of arbitrary startup timing.
- Document clean-checkout startup.

**Acceptance**

- No reproducibility-critical service uses `latest`.
- Clean startup waits for required dependencies.
- Existing demo workflows still work.

**Verification**

- `docker compose config`.
- Clean `up`/health smoke test.
- Existing integration tests.

## RF-M4-P04 — Add production-shaped metrics and tracing configuration

**Dependencies:** RF-M4-P03

**Scope**

Add/configure bounded-cardinality telemetry for:

- request latency/errors;
- DB pool pressure;
- cache outcomes;
- outbox backlog/oldest age;
- stream lag/pending/claim outcomes;
- dead-letter count;
- search projection lag/reconciliation.

Make tracing exporter/sampling configurable; keep stdout convenient locally and support OTLP/Collector usage.

**Acceptance**

- No item IDs, raw queries, idempotency keys, or other high-cardinality user data are metric labels.
- Telemetry failures do not corrupt request correctness.
- Service/resource attributes are meaningful.

**Verification**

- Unit tests for metric registration where applicable.
- App startup smoke.
- CI-equivalent verification.

## RF-M4-P05 — Operator runbook and failure dashboards

**Dependencies:** RF-M4-P04

**Scope**

- Update Grafana/dashboard assets for the new signals.
- Add a concise runbook for:
  - DB unavailable;
  - Redis/Search unavailable;
  - outbox backlog growth;
  - stream consumer lag;
  - dead-letter growth;
  - projection lag/rebuild.
- Avoid invented SLOs; derive thresholds only from measured baseline or clearly label placeholders.

**Acceptance**

- Each alert/signal tells an operator what to inspect next.
- Runbook commands match the repository.
- Dashboard panels use bounded-cardinality metrics.

**Verification**

- Config/provisioning validation where available.
- Manual docs/dashboard review.

**RF-M4 gate:** service startup/shutdown and degraded modes are explicit, environments are reproducible, and the important failure queues are visible.

---

# Milestone RF-M5 — Verification and Portfolio Evidence

Goal: turn the implementation into evidence that can survive technical review.

## RF-M5-P01 — Expand CI verification matrix

**Dependencies:** RF-M4 complete

**Scope**

- Pin integration-test service images.
- Ensure migrations run in CI.
- Add/organize deterministic integration suites for:
  - repository contracts;
  - transactional rollback;
  - concurrent idempotency;
  - event recovery/retention;
  - projection replay/rebuild;
  - lifecycle/shutdown.
- Add focused fuzz targets for parsing/validation boundaries where valuable.

**Acceptance**

- CI fails when a promised guarantee is broken.
- Fuzz targets are bounded enough for CI smoke use.
- Existing benchmark smoke remains meaningful.

**Verification**

- Run full CI-equivalent command set locally where possible.
- Inspect workflow YAML for deterministic service setup.

## RF-M5-P02 — Define reproducible benchmark methodology

**Dependencies:** RF-M5-P01

**Scope**

Document and script:

- hardware/runtime/image versions;
- data cardinality;
- warm/cold cache conditions;
- request mix;
- concurrency;
- run duration/count;
- exact command;
- metrics captured: throughput, p50/p95/p99, errors, CPU/memory, Redis memory, outbox/projection lag.

**Acceptance**

- Another developer can reproduce the run.
- No measured result is added yet unless actually executed and preserved.
- Synthetic/local results are labelled accurately.

**Verification**

- Script dry run/help output.
- Documentation review.

## RF-M5-P03 — Record baseline and hardened benchmark evidence

**Dependencies:** RF-M5-P02

**Scope**

- Run the documented workload.
- Preserve raw output/artifacts or machine-readable summaries.
- Compare relevant before/after behavior.
- Explain regressions/trade-offs rather than cherry-picking a single metric.

**Acceptance**

- Results include environment and exact command.
- No claim exceeds the measured evidence.
- Reliability metrics are reported alongside latency/throughput when relevant.

**Verification**

- Re-run a sample to confirm the script/result pipeline.
- Cross-check README numbers against raw evidence.

## RF-M5-P04 — Final architecture, README, demo, and resume evidence package

**Dependencies:** RF-M5-P03

**Scope**

Update final project-facing material:

- architecture diagram;
- ADR/guarantee matrix;
- failure semantics;
- project journal;
- README claims;
- create -> retry -> update/search -> kill worker -> recover demo;
- benchmark methodology/results;
- dashboard/runbook references.

**Acceptance**

- README describes what the code currently proves, not what the plan intended.
- Demo exercises at least one failure/recovery path.
- Resume-level wording is supported by passing tests and recorded evidence.

A valid eventual claim, only after the evidence exists, is approximately:

> Designed and validated a durable Go service using PostgreSQL transactions and Redis projections/Streams; implemented request idempotency and a transactional outbox relay; demonstrated duplicate-safe recovery, bounded retention, readiness/shutdown behavior, and measured latency under a reproducible workload.

**Verification**

- Fresh clone/read-through.
- Link check/manual docs audit.
- Full CI green on the final branch.

**RF-M5 gate:** the project has evidence for its guarantees and measured claims, not merely an impressive dependency list.

---

# Daily Agent Zero report contract

Every scheduled run must end with a report in this exact structure:

```text
REDISFORGE DAILY HARDENING REPORT

Task:
  <task id> — <title>

Selection:
  Base branch: master
  Starting commit: <sha>
  Branch: <local branch>
  Why this task was selected: <dependency/queue evidence>
  Earlier tasks skipped as already complete: <ids + evidence, or none>

Result:
  Status: READY FOR USER REVIEW | BLOCKED | FAILED
  Behavior changed:
    - ...
  Files changed:
    - ...

Evidence:
  Targeted tests:
    <command> -> PASS/FAIL/UNVERIFIED
  Repository checks:
    <command> -> PASS/FAIL/UNVERIFIED
  Environment limitations:
    - ...

Review first:
  1. <highest-risk diff>
  2. <next>
  3. <next>

Known risks / follow-up:
  - ...

Git:
  Local commit created: yes/no
  Commit SHA: <sha or n/a>
  Commit message:
    <type(scope): summary>

Next queue item:
  <next task id only if current task is complete>
```

Do not replace the evidence section with a generic statement such as "all tests pass."

---

# User review and merge gate

After an Agent Zero run reports `READY FOR USER REVIEW`, the intended human flow is:

```text
inspect diff/local commit
        |
        v
make any manual corrections
        |
        v
push agent branch yourself
        |
        v
GitHub CI
   |          |
 green      red
   |          |
 review     fix/re-run
   v
 merge
   |
   v
next scheduled day may select the next queue item
```

Agent Zero must not treat a local commit as merged progress. The current `master` branch is the source of truth for whether the queue may advance.
