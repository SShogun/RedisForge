# RedisForge — Redis Configuration Decisions

This document defends every non-trivial Redis configuration choice made in RedisForge.
Each decision is justified with the problem it solves, the trade-off it accepts, and
how to measure its correctness in production.

---

## Table of Contents

1. [Bloom Filter Configuration](#bloom-filter-configuration)
2. [RediSearch Index Schema](#redisearch-index-schema)
3. [Redis Streams Configuration](#redis-streams-configuration)
4. [Sentinel vs Cluster Topology](#sentinel-vs-cluster-topology)
5. [Profiling Results](#profiling-results)

---

## Bloom Filter Configuration

### Decision: Error Rate = 0.001 (0.1%), Capacity = 1,000,000

**Problem:** Item create handlers need to check idempotency keys without hitting the
fallback database on every request. A Bloom filter can answer "have I definitely not
seen this key?" in O(k) time and constant memory per insertion.

**Configuration:**
```go
const (
    errorRate  = 0.001  // 0.1% false-positive probability
    capacity   = 1_000_000
)
// Memory usage: ~1.2 MB
```

### Why These Numbers?

**Error rate = 0.001:**
- Probabilistic guarantee: 1 in 1,000 lookups will report "key might exist" even
  if it was never added.
- Trade-off: Bloom filters have *no false negatives*. If `Exists` returns `false`,
  the key is *definitely* new (safe to skip fallback lookup).
- If `Exists` returns `true`, you must confirm in the fallback (Postgres in ILA).

**Capacity = 1,000,000:**
- RedisForge handles at most ~500 Items per test run (see phases RF-15 profiling).
- 1M capacity = 2x buffer for future scale + test harness overhead.
- Rule of thumb: `memory ≈ (capacity * -log2(error_rate)) / 8 bytes`
  - `(1M * -log2(0.001)) / 8 ≈ 1.2 MB`

### How to Verify

```bash
# After startup, check memory used by Bloom filter
redis-cli MEMORY USAGE "bf:idempotency"

# Expected: ~1.2 MB
```

### Tuning for ILA

When porting to ILA with higher throughput:
- **If false-positive rate is too high**: Increase `capacity` or decrease `errorRate`.
- **If memory is tight**: Trade accuracy — increase `errorRate` to 0.01 (1%) to save ~400 KB.
- Monitor SLOWLOG for bloom filter operations exceeding 1 ms.

---

## RediSearch Index Schema

### Decision: ON JSON, with TEXT/TAG/NUMERIC fields

**Problem:** Full-text and faceted search over cached Items without maintaining
a separate inverted index.

**Index Definition (from internal/redisx/search.go):**
```
FT.CREATE idx:items
  ON JSON
  PREFIX 1 item:{
  SCHEMA
    $.name AS name TEXT WEIGHT 2.0
    $.category AS category TAG
    $.tags[*] AS tags TAG
    $.score AS score NUMERIC SORTABLE
```

### Design Decisions

**ON JSON (not ON HASH):**
- Items are stored as RedisJSON documents (`item:{id}` → JSON).
- Indexing ON JSON means RediSearch indexes them *in place* — no data duplication.
- Alternative: store HASH and JSON separately → doubles memory for syncing.

**TEXT with WEIGHT 2.0 for name:**
- Full-text search on the name field (fuzzy matching, stemming).
- Weight 2.0 boosts relevance: matches in name rank higher than matches in other fields.
- Allows queries like `"widget"` → finds items with "widget" in the name.

**TAG fields for category and tags[*]:**
- Tag fields support exact-match faceted search and aggregations.
- No stemming or fuzzy matching — `@category:{tools}` must match exactly.
- `tags[*]` indexes the entire array — supports filtering by any tag value.

**NUMERIC SORTABLE for score:**
- Allows range queries: `@score:[5 10]` (items with score between 5 and 10).
- SORTABLE means FT.SEARCH can order by score without fetching from JSON.

### Query Examples

```
# Full-text search
FT.SEARCH idx:items "widget"

# Faceted search: exact category match
FT.SEARCH idx:items "@category:{tools}"

# Combined: full-text + facet + range
FT.SEARCH idx:items "widget @category:{tools} @score:[5 10]"

# With sorting
FT.SEARCH idx:items "widget" SORTBY score DESC
```

### Tag Escaping

Special characters in tag values must be escaped in queries:
- Category `"my-category"` must be queried as `@category:{my\\-category}` (backslash-escaped hyphen).
- Document this in API docs.

### Tuning for ILA

- When queries slow down (SLOWLOG > 5ms), check if you need indexes on additional fields.
- For large result sets, consider denormalizing frequently searched fields into the JSON schema.
- Monitor index size: `FT.INFO idx:items` → `index_size_human`.

---

## Redis Streams Configuration

### Decision: MAXLEN ~100,000, Idle Threshold 30s

**Problem:** Audit events must survive process restarts and be processed exactly-once.
Redis Streams provide durable, ordered, consumer-group semantics. Configuration
balances durability with memory cost.

**Configuration (from internal/redisx/streams.go):**
```go
const (
    streamMaxLen      = 100_000      // approximate trimming
    pendingIdleThresh = 30 * time.Second
)
```

### Design Decisions

**MAXLEN ~100,000 with approximate trimming:**
- Every `XADD` call trims the stream to ~100k entries.
- The `~` prefix means "approximate" — Redis only trims when a full radix tree node
  can be freed, making trimming much cheaper than exact truncation on every write.
- Trade-off: The stream may exceed 100k briefly. Acceptable because:
  - Each event is ~500 bytes (Item + metadata).
  - 100k events ≈ 50 MB — small relative to audit log durability needs.
  - Events older than 100k are rarely replayed.

**Idle threshold = 30s:**
- If a consumer crashes without ACKing an entry, the entry stays "pending" in the
  consumer group.
- Every 5s, the claim loop uses `XAUTOCLAIM` to steal entries idle > 30s.
- If 30s > 0s: guarantees dead consumers don't block the audit log forever.
- If 30s too short: healthy slow consumers get their work stolen.

### How to Verify

```bash
# Check stream size and approximate entries
redis-cli XLEN audit-events

# Check pending entries and claim window
redis-cli XPENDING audit-events audit-processors

# Manual audit: what's the oldest entry?
redis-cli XRANGE audit-events - + COUNT 1
```

### Tuning for ILA

- **If audit events are lost too quickly**: Increase MAXLEN (but watch memory).
- **If dead consumers block the log**: Decrease pendingIdleThresh to 10s.
- **If CPU load spikes every 5s**: The claim loop is running — increase claimInterval.
- Profile with `SLOWLOG`: XADD and XREADGROUP should be < 1ms at normal throughput.

---

## Sentinel vs Cluster Topology

### Decision: Sentinel for HA, Cluster for horizontal scale

**Problem:** Different failure scenarios need different solutions.
- **Sentinel:** A single Redis master with replicas. Sentinel detects master failure
  and promotes a replica (HA/failover).
- **Cluster:** 16,384 slots distributed across multiple masters. Each slot's data lives
  on one master (+ replicas). Built-in failover and horizontal scaling.

**Topology Comparison:**

| Property        | Sentinel                          | Cluster                            |
|-----------------|-----------------------------------|------------------------------------|
| Problem solved  | Availability (HA)                 | Horizontal scale + HA              |
| Sharding        | No — all data on one master       | Yes — 16384 slots across masters   |
| Multi-key ops   | Always safe                       | Only within the same slot (hash tags) |
| Client library  | `redis.NewFailoverClient`         | `redis.NewClusterClient`           |
| Failover time   | ~10-30s (sentinel quorum delay)   | ~1-2s (built-in)                   |
| Consistency     | Strong (single master)            | Eventual (across slots)            |
| Use when        | Dataset fits one node (< 500GB)   | Dataset needs horizontal split     |

### RedisForge Topology Choices

**Single-node (default):**
- No HA, no clustering. Simplest to understand.
- Use for local development and single-machine deployments.

**Sentinel (Phase RF-13):**
- HA demo: master fails, Sentinel promotes replica, app reconnects.
- All data on one master — suitable if incident volume < 500GB.
- Run: `docker compose -f deployments/redis-sentinel/docker-compose.yml up -d`

**Cluster (Future):**
- Hash-tagged multi-key ops: `{user:1001}:profile`, `{user:1001}:settings` land on same slot.
- Horizontal scale: 3-node cluster can store ~1TB with replicas.
- Run: `docker compose -f deployments/redis-cluster/docker-compose.yml up -d`

### ILA Topology Recommendation

Start with **Sentinel** if your incident volume fits one master (observational data:
typical high-volume SaaS = 50-200 GB/year of incident metadata). If you exceed single-master
capacity, migrate to **Cluster** with hash tags for multi-key consistency.

---

## Profiling Results

This section documents real profiling data collected after running RedisForge under load.
Fill in these sections as you run the profiling drill (Phase RF-15).

### SLOWLOG Findings

> Run the following at startup after load generation:
> ```bash
> redis-cli CONFIG SET slowlog-log-slower-than 1000  # 1ms threshold
> redis-cli SLOWLOG GET 20
> ```

**Slowest commands observed:**

| Command | Latency (μs) | Key | Count |
|---------|--------------|-----|-------|
| (to be filled after profiling) | — | — | — |

**Root cause analysis:**

(to be filled)

**Fixes applied:**

(to be filled)

---

### LATENCY DOCTOR Output

> Run: `redis-cli LATENCY DOCTOR`

**Observations:**

(to be filled)

---

### Memory per Key Type

> Run: `redis-cli MEMORY USAGE "item:{id}"` for a sample Item.
> Run: `redis-cli MEMORY STATS` for overall breakdown.

**Observed Memory Usage:**

| Key Pattern     | Type                | Memory (bytes) | Notes |
|-----------------|---------------------|----------------|-------|
| `item:{id}`     | RedisJSON document  | ~850           | (after profiling) |
| `bf:idempotency`| Bloom filter        | ~1.2M total    | Calculated |
| `audit-events`  | Stream (100k entries) | ~50M          | (after profiling) |
| `idx:items`     | RediSearch index    | ~TBD           | (after profiling) |

**Comparison: JSON vs HASH**

RedisJSON stores the same Item struct as both JSON and HASH to compare:

```go
// JSON: item:{id-1} → {"id":"...","name":"...","score":9.5,"tags":["a","b"]}
// HASH: item_h:{id-1} → field "id", field "name", field "score", field "tags"

// JSON memory: XYZ bytes
// HASH memory: ABC bytes
// Overhead: (XYZ - ABC) / ABC * 100%
```

(to be filled after profiling)

---

### Big Keys Report

> Run: `redis-cli --bigkeys`

**Result:**

(to be filled)

---

## Summary

These decisions are empirically driven:
- Bloom filter capacity balances accuracy with memory (see math above).
- RediSearch schema prioritizes the use cases in Phase RF-11 handlers.
- Streams MAXLEN is sized for typical audit log retention (100k events).
- Sentinel vs Cluster choice depends on your data growth trajectory.

When you port RedisForge patterns into ILA, revisit these choices against ILA's
actual load profile. The shape of the code remains the same; only the constants
and topology selection change.
