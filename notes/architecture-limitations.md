# RedisForge: Architecture and Limits

## Critical path: `POST /v1/items`

`internal/app/app.go` wires chi handlers to a `CacheItemRepo` whose fallback is a new `MemoryItemRepo`.

1. `HandleCreateItem` validates name/category. With an idempotency key, `BF.EXISTS` may reject it as a duplicate; a Bloom error fails open.
2. `CacheItemRepo.Create` writes to memory first, then best-effort writes the item as RedisJSON (`item:{id}`). The memory repo assigns version/timestamps.
3. After create, Bloom `BF.ADD` is best effort. The handler starts `EmitAsync` and returns `201` without waiting for audit `XADD`.
4. `Emitter` uses a five-second background timeout to append to `audit-events`. The stream uses approximate `MAXLEN ~ 100000` trimming.
5. `AuditWorker` creates `audit-processors` at stream ID `0`, reads new entries via `XREADGROUP`, and scans stale pending entries via cursor-complete `XAUTOCLAIM` (30-second minimum idle, every five seconds). Processing logs the event then ACKs it; an ACK failure leaves it recoverable.

## Important limits

- **Item durability:** the app's fallback is in-memory and is recreated on startup. RedisJSON is a cache, not the durable source of truth. Redis can therefore retain cached items after memory data is lost; GET and List may disagree after a restart.
- **Search/cache coupling:** RediSearch indexes RedisJSON cache keys. Update invalidates the cache entry, so the item can disappear from search until a later GET backfills it. Search has no fallback scan.
- **Audit durability:** producer failures and in-flight emits at shutdown can lose events. The stream is bounded, and the worker currently only logs and ACKs -- it does not write to an external audit store. At-least-once recovery applies only while the stream retains an entry's payload; it is not permanent retention or exactly-once processing.
- **Idempotency:** Bloom positives (including false positives) are rejected; lookup errors fail open and add errors are ignored. This is not a durable idempotency record.
- **Health/topology:** `/healthz` always returns `ok` and does not probe Redis. Standalone, Sentinel, and Cluster clients are configurable, but startup also requires RedisJSON, RedisBloom, and RediSearch modules.

## Verification context

Local inspection used Go 1.27 on Windows. `go vet ./...` and `go build` passed. The race command could not run because local CGO is disabled; non-race integration tests requiring Testcontainers failed because Docker is unavailable here. The CI workflow runs on Ubuntu. `gofmt -l` on this Windows checkout reports CRLF worktree files; Git records them as LF, so that local output does not establish a Linux CI formatting failure.
