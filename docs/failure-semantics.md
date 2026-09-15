# RedisForge Failure Semantics

This document states what RedisForge guarantees when Redis operations fail. It is intentionally narrower than a production outbox or transactional-event design: the project keeps item writes and Redis audit emission separate so the trade-offs remain visible.

## Audit event lifecycle

For successful item creates, updates, and deletes, the handler schedules a best-effort audit event with action `created`, `updated`, or `deleted`.

```text
HTTP write succeeds
    |
    +--> response can return immediately
    |
    `--> async audit emitter
            |
            +--> JSON serialize
            +--> XADD audit-events (5s timeout)
            |
            `--> audit worker consumer group
                    |
                    +--> process event
                    `--> XACK
```

### Before the event reaches Redis

The producer path is asynchronous and best effort.

- A successful item write is **not rolled back** when audit `XADD` fails.
- Producer append failures are logged with `event_id`, `item_id`, `action`, and the error.
- `audit_events_emitted_total{action,status}` counts producer append outcomes.
- `audit_emit_latency_ms{action,status}` measures serialization plus append latency.
- Each async append has a 5-second timeout.
- There is currently no producer retry queue, transactional outbox, or disk-backed handoff.
- Because producer goroutines are not drained during shutdown, an in-flight best-effort event can also be lost if the process exits before the append finishes.

Operationally, any sustained increase in:

```promql
rate(audit_events_emitted_total{status="error"}[5m])
```

means item writes may be succeeding without corresponding audit entries.

### After the event reaches Redis

Once `XADD` succeeds, the worker contract is **at least once**, not exactly once.

- The consumer reads new group entries with `XREADGROUP`.
- Delivered but unacknowledged entries remain in the Pending Entries List (PEL).
- Stale pending entries are recovered with cursor-complete `XAUTOCLAIM` scanning.
- The worker ACKs only after its processing step succeeds.
- ACK failures are treated as processing failures, so the entry remains recoverable.
- A crash after processing but before ACK can cause the same event to be delivered again.

Any future worker side effect must therefore be idempotent or explicitly tolerate duplicate delivery.

### Malformed audit entries

Malformed entries are not useful work and are ACKed to prevent a permanent poison-message loop. If that cleanup ACK fails, the failure is surfaced instead of being reported as success, so the pending entry can be reclaimed again.

RedisForge does not currently maintain a dead-letter stream containing malformed payloads. The current policy is **log + ACK/drop**, with ACK failure remaining visible.

## Item cache failures

`CacheItemRepo` treats RedisJSON as a cache in front of the fallback repository.

| Operation | Redis/cache failure behavior | Item operation result |
| --- | --- | --- |
| Create | Fallback create succeeds, cache write-through failure is logged | Create still succeeds |
| Get | Cache miss or cache read error falls through to fallback; successful fallback read attempts cache backfill | Read can still succeed |
| Update | Fallback update succeeds, cache invalidation failure is logged | Update still succeeds |
| Delete | Fallback delete succeeds; cache deletion is best effort | Delete still succeeds |

This is deliberate cache degradation behavior: Redis cache availability does not define item durability in the current architecture.

## Bloom-filter idempotency failures

The create handler uses the Bloom filter as a demonstration pre-check rather than a durable idempotency store.

- If `BF.EXISTS` returns `true`, the request is rejected as a duplicate.
- If the Bloom lookup itself errors, the handler currently **fails open** and continues the create path.
- After a successful create, `BF.ADD` is best effort; its error does not fail the HTTP request.
- A lost Bloom update can therefore allow a later request with the same idempotency key to create another item.

The Bloom filter must not be interpreted as a transactional exactly-once request guarantee.

## Search failures

RediSearch is on the request path for the search endpoint. An `FT.SEARCH` error is returned to the handler and surfaces as a failed request; there is no fallback full scan.

## What RedisForge does not claim

RedisForge does **not** currently claim:

- atomic item-write + audit-event commit
- exactly-once audit delivery or processing
- durable producer retries before `XADD`
- transactional idempotency based on the Bloom filter
- cache writes as the source of truth

Those would require different architecture, such as a transactional outbox, durable retry queue, or a database-backed idempotency record.

## Verification map

| Failure semantic | Proof |
| --- | --- |
| create/update/delete schedule the correct audit action | `internal/handlers/audit_events_test.go` |
| producer append errors are returned by the synchronous emitter path | `internal/audit/emitter_test.go` |
| pre-existing stream entries replay after group creation | `internal/workers/audit_workers_test.go` |
| stale PEL recovery traverses the `XAUTOCLAIM` cursor | `internal/workers/audit_workers_test.go` |
| ACK failures remain processing failures | `internal/workers/audit_workers_test.go` |
| real Redis module integration | Testcontainers coverage under `internal/redisx` and `internal/handlers` |

The `/metrics` endpoint is the operational proof surface for producer failures, worker processing, pending counts, RedisJSON latency, Bloom checks, and search latency.
