# RedisForge Project Journal

This file exists to keep RedisForge visibly alive and useful. Update it when you make meaningful project progress: code changes, docs improvements, benchmark runs, demos, bug fixes, or design notes.

## Commit Cadence

Aim for small commits that prove real movement:

| Cadence | Good Commit Type |
| --- | --- |
| Daily or every few sessions | small docs note, test improvement, benchmark note, demo cleanup |
| Weekly | one focused feature, one Redis explanation, one profiling update |
| Monthly | roadmap cleanup, architecture review, README polish, release notes |

## Commit Message Examples

```text
docs: explain streams pending-entry recovery
docs: add visitor learning path
test: cover cache-aside miss backfill
feat: add redis stream lag metric
perf: record baseline search latency
chore: refresh demo workflow for windows
```

## Public Progress Ideas

- Record a 60 second demo of create -> search -> metrics.
- Add one "Redis lesson learned" note after each implementation change.
- Open issues for future Redis experiments before implementing them.
- Keep benchmarks honest by committing both the script and the result note.
- Add screenshots or terminal recordings when a feature becomes visual.

## Project Log

### 2026-05-20

- Ran load generation benchmarks using `scripts/benchmark.ps1`.
- Populated `docs/profiling-results.md` and `docs/redis-decisions.md` with real metrics (p50/p99 latency, JSON vs HASH memory comparison).
- Verified JSON overhead (~73%) is justified by the performance gains in partial updates and searching.
- Added Mermaid sequence diagrams for Cache-Aside and Streams flows in `docs/implementation/redis-patterns.md`.
- Added `stream_pending_count` Prometheus gauge metric and `GetPendingCount` method.
- Created `internal/handlers/items_create_test.go` — handler-level idempotency integration test using Testcontainers.
- Expanded Cluster hash-tag documentation with concrete slot-mapping examples.
- Created `deployments/redis-cluster/` — full 6-node (3 master + 3 replica) Cluster demo with README.
- Replaced all text-based architecture flows with Mermaid diagrams in `docs/implementation/architecture.md`.
- Added Feature Coverage Matrix to `docs/README.md` — audits code/test/doc status for every feature.
- Updated `docs/REDISFORGE_BUILD_GUIDE.md` with Phase 16 (Cluster) and Phase 17 (Polish).
- Refreshed `docs/redisforge_interview_answers.md` to reflect current project state (all features implemented).
- Updated `docs/README.md` reading order and intent table with interview guides and cluster demo.

### 2026-05-14

- Reworked the repository presentation around RedisForge as a Redis learning and revision resource.
- Added a docs index so visitors can choose architecture, Redis patterns, build history, profiling, demos, or commit cadence.
- Added implementation notes for architecture and Redis patterns.
- Added this project journal to support steady public commits and visible progress.

## Next Good Commits

- `test: add RediSearch integration test`
- `test: add Streams consumer group integration test`
- `test: add Pub/Sub integration test`
- `perf: capture benchmark output after cache warmup`
- `docs: add screenshots or terminal recordings for demo`
- `chore: record a 60-second demo video`

