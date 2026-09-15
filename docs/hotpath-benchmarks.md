# RedisForge Hot-Path Benchmarks

This note defines the reproducible benchmark harness for RedisForge's Redis-facing hot paths. It complements the historical load-test snapshots in `profiling-results.md`; it does not replace them or invent a portable latency target.

## What is measured

`BenchmarkRedisHotPaths` runs against a real, pinned Redis Stack container and measures the application-observed cost of:

| Benchmark | Redis command | Why it is hot |
| --- | --- | --- |
| `JSON_GET` | `JSON.GET` | Cache-aside reads |
| `JSON_SET` | `JSON.SET` | Cache write-through and benchmark seeding |
| `BLOOM_EXISTS` | `BF.EXISTS` | Create-request idempotency pre-check |
| `FT_SEARCH` | `FT.SEARCH` | Search endpoint queries |
| `XADD` | `XADD` | Audit-event append path |

The benchmark includes Go client overhead, serialization/deserialization, the Docker/network round trip, Redis command execution, and RedisForge instrumentation. It is therefore an **application-observed integration benchmark**, not a raw Redis server microbenchmark.

The Testcontainers client uses **RESP2**, matching `redisx.Open` in single-node, Sentinel, and Cluster modes. This matters because module command response shapes differ between RESP2 and RESP3; integration tests and benchmarks should exercise the protocol the application actually uses.

## Run it

From the repository root:

```bash
go test -count=1 ./internal/redisx \
  -run '^$' \
  -bench '^BenchmarkRedisHotPaths$' \
  -benchtime=100x \
  -benchmem
```

For a longer local sample:

```bash
go test -count=1 ./internal/redisx \
  -run '^$' \
  -bench '^BenchmarkRedisHotPaths$' \
  -benchtime=3s \
  -benchmem
```

The test harness starts `redis/redis-stack-server:7.4.0-v8` with Testcontainers, seeds representative data, waits for RediSearch indexing, and cleans the container up automatically.

## CI behavior

CI runs the same benchmark with `-benchtime=1x` as a **runtime smoke test**. That step proves the benchmark paths and container integration still execute, but it is intentionally not a performance gate.

GitHub-hosted runners vary in CPU scheduling, host load, and Docker/network behavior. Failing a PR because `ns/op` moved by a small percentage on shared runners would create false alarms.

## How to compare results

Only compare benchmark numbers collected under equivalent conditions. Record at least:

- date and commit SHA
- Go version
- operating system and architecture
- CPU model / core allocation
- Docker version and resource limits
- Redis Stack image
- benchmark `-benchtime`

Use `ns/op` for latency direction and `B/op` + `allocs/op` for client-side allocation changes. A useful optimization claim should name the benchmark, before/after commits, environment, and sample settings.

## What this benchmark does not prove

It does not prove production capacity, p99 behavior under concurrency, failover behavior, or Redis Cluster/Sentinel performance. Those require separate load and topology tests.

It also does not turn the historical numbers in `profiling-results.md` into universal thresholds. That document records an earlier local load session; this harness exists so future measurements can be repeated against a pinned dependency and a known workload.
