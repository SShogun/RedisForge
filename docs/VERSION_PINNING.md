# Dependency Version Pinning Strategy

## go-redis/v9 Stability Commitment

**Current pinned version:** `v9.19.0`

### Why this version matters

`go-redis/v9` is pinned to ensure consistent behavior across critical subsystems:

#### 1. Sentinel Failover Stability
- `NewFailoverClient()` implementation has subtle timing differences between patch releases
- Failover detection intervals and quorum logic can shift with minor version bumps
- Pinning prevents unexpected HA behavior during automated failovers

#### 2. Cluster Hash-Slot Resolution
- Slot migration handling varies between releases
- Multi-key operations depend on stable CLUSTER SLOTS response parsing
- Uncontrolled updates can break operations on keys with hash tags (e.g., `item:{123}`)

#### 3. Context Cancellation Semantics
- How context.Done() propagates through blocking operations changed in v9.16+
- Pub/Sub and Stream operations rely on predictable context behavior
- Version stability ensures graceful shutdown works as expected

### Transitive Dependency Alignment

These must match the go-redis version:

```
github.com/redis/go-redis/v9         v9.19.0
github.com/redis/go-redis/extra/redisotel/v9  v9.19.0  ← must match
go.opentelemetry.io/otel             v1.26.0+  ← ensure compatibility
```

Mismatched versions can cause:
- Silent tracing failures (OTel instrumentation not registered)
- Panic on metrics collection (Prometheus type mismatches)
- Nil pointer dereferences in cluster failover paths

### Upgrade Procedure

Before bumping the version:

1. **Run Sentinel drill** (docs/REDISFORGE_BUILD_GUIDE.md, Phase 13)
   - Verify failover detection time and ACK behavior
   - Ensure replica recovery completes cleanly

2. **Run Cluster hash-tag drill**
   - Multi-key operations: `MGET`, `MSET` across different slots
   - Verify slot migration doesn't drop ACKs in Streams

3. **Run integration tests with `-race` flag**
   ```bash
   go test -race ./...
   ```
   - Catches timing-sensitive bugs in context propagation
   - Validates graceful shutdown paths

4. **Update profiling baseline** (docs/profiling-results.md)
   - Compare latency percentiles before/after
   - Document any performance regressions

5. **Commit with justification**
   ```
   go get -u github.com/redis/go-redis/v9
   go mod tidy
   # verify everything passes above, then commit with explanation of testing done
   ```

### Current Test Coverage

All subsystems tested with v9.19.0:

- ✅ Sentinel failover (integration test)
- ✅ Cluster hash slots (multi-key ops)
- ✅ Stream consumer groups and claim operations
- ✅ Pub/Sub subscription lifecycle
- ✅ Context cancellation in blocking operations
- ✅ JSON, Bloom, Search module operations

### Future Considerations

If go-redis/v10 is released:

1. It will require code changes (API breaking changes likely)
2. Redis Cluster Sharded Pub/Sub support may be added
3. Context handling for blocking operations may change
4. Plan 2–3 weeks for validation before upgrading
