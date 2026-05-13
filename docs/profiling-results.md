# RedisForge — Profiling Results

This document records performance profiling data collected under realistic load conditions.
All measurements were taken against `redis/redis-stack:latest` in the standard docker-compose
configuration (Phase RF-0). Results help validate configuration decisions from
[docs/redis-decisions.md](redis-decisions.md) and inform tuning for production.

---

## Table of Contents

1. [Test Setup](#test-setup)
2. [Load Generation](#load-generation)
3. [SLOWLOG Analysis](#slowlog-analysis)
4. [Latency Analysis](#latency-analysis)
5. [Memory Analysis](#memory-analysis)
6. [Throughput & Concurrency](#throughput--concurrency)
7. [Big Keys Report](#big-keys-report)
8. [Comparative Analysis: JSON vs HASH](#comparative-analysis-json-vs-hash)
9. [Observations & Recommendations](#observations--recommendations)

---

## Test Setup

**Date of profiling:** (to be filled)

**Environment:**
- Host: (to be filled — local/EC2/etc.)
- Redis version: redis/redis-stack:latest (contains JSON, Bloom, Search, Streams)
- RedisForge version: (git commit or tag)
- Load generator: (custom script, wrk, artillery, etc.)

**Configuration used:**
```env
REDIS_ADDR=localhost:6379
REDIS_POOL_SIZE=10
SERVER_PORT=8080
ENV=development
```

**Redis configuration:**
```
maxmemory: unlimited
slowlog-log-slower-than: 1000 (microseconds)
slowlog-max-len: 128
```

---

## Load Generation

### Test Scenario

**Goal:** Simulate realistic RedisForge usage: create Items, list them, search, and
audit events flow through the Stream.

**Load profile:**

| Operation | Requests | Rate | Duration |
|-----------|----------|------|----------|
| POST /v1/items (create) | (to be filled) | (to be filled) | (to be filled) |
| GET /v1/items/{id} (get) | (to be filled) | (to be filled) | (to be filled) |
| PUT /v1/items/{id} (update) | (to be filled) | (to be filled) | (to be filled) |
| GET /v1/items (list) | (to be filled) | (to be filled) | (to be filled) |
| GET /v1/items/search (search) | (to be filled) | (to be filled) | (to be filled) |

**Example load script:**
```bash
#!/bin/bash
# Generate 500 items
for i in $(seq 1 500); do
  curl -s -X POST http://localhost:8080/v1/items \
    -H "Content-Type: application/json" \
    -d "{
      \"name\":\"item-$i\",
      \"category\":\"category-$((i % 10))\",
      \"score\":$((RANDOM % 100)),
      \"tags\":[\"tag-$((i % 5))\",\"test\"],
      \"idempotency_key\":\"key-$i\"
    }" > /dev/null
  [ $((i % 50)) -eq 0 ] && echo "Created $i items..."
done

# Generate 100 searches
for i in $(seq 1 100); do
  curl -s -X GET "http://localhost:8080/v1/items/search?q=item&limit=20" > /dev/null
done

# Generate 100 reads
for i in $(seq 1 100); do
  ID=$(redis-cli KEYS "item:{*}" | head -1)
  curl -s -X GET "http://localhost:8080/v1/items/$ID" > /dev/null
done
```

### Metrics Collected

- **Throughput:** Requests per second
- **Latency:** p50, p95, p99, max (milliseconds)
- **Error rate:** % of requests failing
- **Redis commands:** Slowest, most frequent, total count

---

## SLOWLOG Analysis

**Command run:**
```bash
redis-cli CONFIG SET slowlog-log-slower-than 1000  # 1ms threshold
redis-cli SLOWLOG GET 20
```

### Slowest Commands Observed

| Rank | Command | Args | Duration (μs) | Key(s) | Count |
|------|---------|------|---------------|--------|-------|
| 1 | (to be filled) | — | — | — | — |
| 2 | (to be filled) | — | — | — | — |
| 3 | (to be filled) | — | — | — | — |

### Analysis

**Expected slowest operations (before profiling):**
- `FT.SEARCH` on large result sets (Bloom filter queries)
- `JSON.GET` on large Items with many tags
- `XREADGROUP` when backlog is deep (rebalancing consumers)

**Observed bottlenecks:**

(to be filled after profiling)

**Root causes & fixes applied:**

(to be filled)

---

## Latency Analysis

**Command run:**
```bash
redis-cli LATENCY RESET
redis-cli LATENCY DOCTOR
```

### LATENCY DOCTOR Output

```
(paste full output here)
```

### Histogram: Command Latency

| Command | p50 (μs) | p95 (μs) | p99 (μs) | Max (μs) | Samples |
|---------|----------|----------|----------|----------|---------|
| JSON.SET | — | — | — | — | — |
| JSON.GET | — | — | — | — | — |
| BF.EXISTS | — | — | — | — | — |
| FT.SEARCH | — | — | — | — | — |
| XADD | — | — | — | — | — |
| XREADGROUP | — | — | — | — | — |
| PING | — | — | — | — | — |

### Interpretation

- **p50 (median):** Expected case. Should be < 500 μs for all commands.
- **p95/p99 (tail latency):** Worst 5% and 1% of requests. Indicates variability.
- **Max:** Single worst observation. Use to detect outliers (GC pauses, swaps, etc.).

**Health thresholds:**
- ✅ All commands < 1000 μs (1 ms): healthy
- ⚠️  p95 > 1 ms: investigate indexing or cache misses
- ❌ p99 > 5 ms: likely memory pressure or network issues

---

## Memory Analysis

### Overall Memory Usage

**Commands run:**
```bash
redis-cli MEMORY STATS
redis-cli MEMORY USAGE "item:{sample-id}"
redis-cli MEMORY USAGE "bf:idempotency"
```

### Memory Breakdown

| Component | Type | Count | Memory (bytes) | % of Total |
|-----------|------|-------|----------------|-----------|
| Items (JSON documents) | RedisJSON | (to be filled) | (to be filled) | — |
| Bloom filter | BF | 1 | ~1.2M | (to be filled) |
| Search index | FT | 1 | (to be filled) | (to be filled) |
| Audit stream | STREAM | 1 | (to be filled) | (to be filled) |
| Pub/Sub subscriptions | (internal) | (to be filled) | (to be filled) | — |

**Total Redis memory used:** (to be filled)

### Memory per Key Type

**Sample JSON Item (`item:{id}`):**

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "name": "Widget Premium",
  "category": "electronics",
  "score": 87.5,
  "tags": ["bestseller", "new", "sale"],
  "version": 2,
  "created_at": "2026-05-13T10:30:00Z",
  "updated_at": "2026-05-13T11:45:30Z"
}
```

**Observed memory:** (to be filled) bytes

**Breakdown (estimate):**
- JSON structure overhead: ~80 bytes
- Field names + values: ~X bytes
- Per-field pointers: ~Y bytes

---

### JSON vs HASH Comparison

**Hypothesis:** RedisJSON is slightly larger per item due to key names in the JSON,
but partial-path updates (`JSON.NUMINCRBY`, `JSON.ARRAPPEND`) are cheaper than
fetching-updating-replacing a HASH.

**Test (after profiling):**
```bash
# Store same Item as JSON
JSON.SET item:{1} $ '{"id":"1","name":"test","score":10,"tags":[]}'

# Store same Item as HASH
HSET item_h:{1} id "1" name "test" score "10" tags "[]"

# Compare memory
MEMORY USAGE item:{1}
MEMORY USAGE item_h:{1}

# Compare update latency: increment score by 5
# JSON: JSON.NUMINCRBY item:{1} $.score 5
# HASH: HGETALL -> modify -> HSET (3 commands)
```

**Results:**

| Storage | Memory | Update latency (μs) | Notes |
|---------|--------|---------------------|-------|
| JSON | (to be filled) | (to be filled) | Partial-path updates |
| HASH | (to be filled) | (to be filled) | Fetch-modify-replace |

**Conclusion:** (to be filled)

---

### Bloom Filter Overhead

**Configuration (from internal/redisx/bloom.go):**
```
capacity = 1,000,000
errorRate = 0.001 (0.1%)
```

**Theoretical memory:** ~1.2 MB (from math in redis-decisions.md)

**Observed memory:** (to be filled) bytes

**Accuracy test:**
- Items added: (to be filled)
- False positives observed: (to be filled) %
- Actual error rate: (to be filled)

---

## Throughput & Concurrency

### HTTP Request Metrics

(From `make run` while load is applied)

| Endpoint | Method | Avg Latency (ms) | p99 Latency (ms) | RPS | Errors |
|----------|--------|------------------|------------------|-----|--------|
| /healthz | GET | (to be filled) | (to be filled) | (to be filled) | — |
| /v1/items | POST | (to be filled) | (to be filled) | (to be filled) | — |
| /v1/items/{id} | GET | (to be filled) | (to be filled) | (to be filled) | — |
| /v1/items/{id} | PUT | (to be filled) | (to be filled) | (to be filled) | — |
| /v1/items/{id} | DELETE | (to be filled) | (to be filled) | (to be filled) | — |
| /v1/items | GET | (to be filled) | (to be filled) | (to be filled) | — |
| /v1/items/search | GET | (to be filled) | (to be filled) | (to be filled) | — |

### Goroutine Leak Check

**Before load:**
```bash
# In a test: runtime.NumGoroutine()
```
(to be filled)

**During load:**
(to be filled)

**After shutdown:**
(to be filled)

**Conclusion:** Goroutines returned to baseline? (Yes/No)

---

## Big Keys Report

**Command run:**
```bash
redis-cli --bigkeys
```

### Output

```
(paste full output here)

Sample output:
-------- Summary -------
Sampled 1234 keys in the current database
Total key length in bytes is 12345 (avg len 9.99)
Total value length in bytes is 5432101 (avg len 4404.23)
Biggest string found 'key:123' has 65536 bytes
Biggest list found 'list:1' has 10000 elements
Biggest set found 'set:1' has 100000 members
Biggest zset found 'zset:1' has 100000 members
Biggest hash found 'hash:1' has 50000 fields
```

### Analysis

**Largest keys by memory:**

| Rank | Key | Type | Size | Growth rate |
|------|-----|------|------|-------------|
| 1 | (to be filled) | — | — | — |
| 2 | (to be filled) | — | — | — |
| 3 | (to be filled) | — | — | — |

**Expected largest:**
- `audit-events` stream (MAXLEN ~ 100k entries)
- `idx:items` search index (proportional to Item count)
- Individual `item:{id}` JSON documents

---

## Comparative Analysis: JSON vs HASH

This section directly validates the design decision from Phase RF-4 to use RedisJSON.

### Test Procedure

1. Create 100 Items as JSON and 100 as HASH
2. Measure memory for each
3. Measure latency: full read + increment score + write

### Results

**Memory per Item:**

| Format | Total bytes | % overhead vs HASH |
|--------|-------------|-------------------|
| JSON | (to be filled) | (to be filled) |
| HASH | (to be filled) | (to be filled) |

**Update latency (100 iterations):**

| Operation | JSON (μs) | HASH (μs) | Ratio |
|-----------|-----------|-----------|-------|
| Read full item | (to be filled) | (to be filled) | (to be filled) |
| Increment score | (to be filled) | (to be filled) | (to be filled) |
| Append tag | (to be filled) | (to be filled) | (to be filled) |

**Conclusion:**

(to be filled — does JSON's overhead justify faster partial updates?)

---

## Observations & Recommendations

### What Worked Well

(to be filled — list things that performed as expected)

### Bottlenecks Encountered

(to be filled)

### Configuration Tuning Applied

(to be filled — document any changes to Bloom capacity, Stream MAXLEN, etc.)

### Recommendations for ILA Port

When you port RedisForge patterns into ILA with higher scale:

1. **Memory budget:** If total Redis memory approaches 80% of available:
   - Increase `streamMaxLen` or implement older entry cleanup
   - Consider Cluster topology for horizontal scaling

2. **Latency targets:** If p99 latency exceeds 10 ms:
   - Check if `FT.SEARCH` queries are hitting unindexed fields
   - Verify Bloom filter error rate is not causing excess fallback lookups

3. **Concurrency:** If error rate increases under concurrency:
   - Increase `REDIS_POOL_SIZE` in config
   - Monitor goroutine count for leaks (should return to baseline after shutdown)

4. **Index maintenance:** As Item count grows:
   - `FT.INFO idx:items` to monitor index size
   - Consider whether category/tags need additional indexes

---

## Profiling Artifacts

(Links or file locations for raw profiling data, if retained)

- SLOWLOG snapshot: (file path)
- LATENCY output: (file path)
- Load test script output: (file path)
- Memory snapshot: (file path)

---

## Sign-Off

**Profiled by:** (your name)

**Date:** (to be filled)

**Approved for production:** (Yes/No — after review)

**Notes:** (any caveats or follow-up work)
