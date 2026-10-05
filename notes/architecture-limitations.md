# RedisForge: Architecture Baseline, Guarantees, and Execution Guardrails

This file is the **current-state contract** for RedisForge. It exists so future implementation work starts from the repository that actually exists rather than from an imagined target architecture.

The companion execution queue is [redisforge-principal-engineer-hardening-plan.md](redisforge-principal-engineer-hardening-plan.md).

## Current critical path: `POST /v1/items`

`internal/app/app.go` wires chi handlers to a `CacheItemRepo` whose fallback is a newly constructed `MemoryItemRepo`.

1. `HandleCreateItem` validates the request. With an idempotency key, `BF.EXISTS` may reject it as a duplicate; a Bloom error fails open.
2. `CacheItemRepo.Create` writes to memory first, then best-effort writes the item as RedisJSON (`item:{id}`). The memory repo assigns version/timestamps.
3. After create, Bloom `BF.ADD` is best effort. The handler starts `EmitAsync` and returns `201` without waiting for audit `XADD`.
4. `Emitter` uses a five-second background timeout to append to `audit-events`. The stream uses approximate `MAXLEN ~ 100000` trimming.
5. `AuditWorker` creates `audit-processors` at stream ID `0`, reads new entries via `XREADGROUP`, and scans stale pending entries via cursor-complete `XAUTOCLAIM` (30-second minimum idle, every five seconds). Processing logs the event then ACKs it; an ACK failure leaves it recoverable.

## Current guarantees and limits

| Area | Current guarantee | Current limitation |
| --- | --- | --- |
| Item durability | CRUD succeeds against the in-process repository and may populate RedisJSON | The authoritative fallback is memory and is recreated on startup; RedisJSON can outlive memory, so GET/List/search can diverge after restart |
| Search | RediSearch indexes RedisJSON keys | Update invalidates the cache key, so an item can disappear from search until a later read backfills it; there is no authoritative fallback scan |
| Idempotency | Bloom may reject a previously seen key | Bloom false positives can reject valid requests; errors fail open; there is no durable request/response receipt |
| Audit publication | Successful `XADD` gives a recoverable stream entry while retained | Producer failure or shutdown can lose best-effort emits; bounded trimming is not permanent retention |
| Audit processing | Consumer-group recovery can redeliver retained pending entries | The worker logs and ACKs only; it has no durable audit sink or exactly-once guarantee |
| Health | `/healthz` proves the HTTP process can answer | It does not prove Redis or any future database dependency is ready |
| Redis topology | Standalone, Sentinel, and Cluster clients are configurable | Startup also depends on RedisJSON, RedisBloom, and RediSearch module availability |
| Process lifecycle | Graceful shutdown paths exist | HTTP listener/start failures are not yet treated as a top-level startup failure in the desired hardened contract |

These limitations are not bugs to hide. They are the baseline the hardening plan is intended to replace deliberately and with evidence.

## Chosen target direction

For the hardening work, the default architecture decision is:

- **PostgreSQL becomes the item system of record.**
- A single PostgreSQL transaction owns item mutation, durable idempotency receipt, and outbox event creation.
- Redis remains important as the cache, search projection, Streams delivery buffer, topology-learning surface, and observability target.
- Search is an explicitly versioned projection of durable item state.
- Stream delivery is **at least once**. Consumers must be replay-safe and idempotent.
- `MemoryItemRepo` remains test/demo infrastructure only; it must not be presented as a production durability layer.

If this target is intentionally changed later, update this file first and explain the trade-off before implementation begins.

## Non-negotiable invariants for hardening work

1. **One source of truth.** Do not introduce a second durable item authority or an ambiguous dual-write path.
2. **No correctness dependency on Bloom.** Bloom can optimize; it cannot decide durable idempotency alone.
3. **No DB + message dual-write gap.** Item mutation and outbox creation must share one database transaction.
4. **Duplicates are expected.** Relay and consumer logic must be safe if the same `event_id` appears more than once.
5. **Old projection events cannot overwrite newer state.** Item versions must make replay monotonic.
6. **ACK follows durable effect.** A consumer must not acknowledge an event before its required durable effect/receipt is established.
7. **Retention is explicit.** A bounded Redis Stream is a delivery buffer, not a permanent audit store.
8. **Failure semantics stay observable.** Do not silently fall back in a way that changes user-visible guarantees.
9. **No invented performance claims.** Benchmarks must record workload, environment, commands, and measured output.
10. **No resume-driven infrastructure.** Kafka, Kubernetes, extra Redis modules, or similar additions require a demonstrated need.

## Agent execution guardrails

Automated implementation is allowed only inside the companion hardening plan and must obey these rules.

### The agent may

- inspect the repository and recent Git history;
- create or switch to a local work branch;
- implement **one bounded task** from the hardening plan;
- add or update tests required by that task;
- update directly affected documentation when behavior changes;
- run local verification commands;
- create a **local commit** after verification if configured to do so;
- produce a final evidence report and proposed/final commit message.

### The agent must not

- push to a remote;
- merge, rebase, force-push, tag, release, or deploy;
- open or merge a pull request unless explicitly asked by the user;
- start a later task because the current one is blocked;
- redesign the target architecture to make a task easier;
- weaken or delete a failing test merely to obtain green;
- change unrelated files for cleanup, style, dependency upgrades, or refactors;
- add infrastructure not required by the selected task;
- claim verification that did not actually run.

### Mandatory stop conditions

Stop and report instead of continuing when any of the following is true:

- the worktree contains unrelated uncommitted user changes;
- the previous automated change is still unreviewed/unmerged and continuing would stack unrelated work on it;
- the selected task's prerequisite is not satisfied on the current base branch;
- implementation requires a product/architecture decision not already made in these notes;
- a required dependency, Docker service, credential, or local capability is unavailable;
- a failing test appears unrelated to the task and the root cause is not safely attributable;
- completing the task would require broadening scope beyond its stated acceptance criteria.

## Repository verification baseline

RedisForge CI currently runs the following on Ubuntu:

```bash
test -z "$(gofmt -l .)"
go vet ./...
go test -race -count=1 ./...
go test -count=1 ./internal/redisx -run '^$' -bench '^BenchmarkRedisHotPaths$' -benchtime=1x -benchmem
go build ./cmd/redisforge
```

Every automated task must run the smallest relevant tests first, then as much of the CI-equivalent baseline as the local environment supports.

Rules:

- If a command cannot run, report **UNVERIFIED** with the exact reason; do not convert it into a pass.
- Do not run `go mod tidy` unless the selected task changes dependencies or module metadata.
- Docker/Testcontainers failures caused by Docker being unavailable are environment limitations, not proof of correctness.
- Local Windows `gofmt -l` results can be affected by CRLF worktree behavior; CI on Ubuntu is the final formatting authority.
- CI success after the user pushes remains the remote merge gate.

## Evidence required from every automated implementation

The final report must contain:

| Field | Required content |
| --- | --- |
| Task | Exact task ID and title from the hardening plan |
| Base | Base branch and starting commit |
| Changed files | Exact paths |
| Behavior changed | Concise before/after explanation |
| Tests added/changed | What property they prove |
| Verification | Each command and PASS / FAIL / UNVERIFIED |
| Risks | Remaining uncertainty or follow-up |
| Diff summary | What the user should inspect first |
| Commit | Local commit SHA if created, otherwise proposed commit message |
| Next task | The next queue item only if this task is actually complete |

## Revalidation rule

Before executing a hardening task, the agent must compare this baseline against the current repository. If the repository has already evolved past a statement here, treat the code and tests as evidence, update this file as part of the relevant task, and do not blindly recreate already-completed work.
