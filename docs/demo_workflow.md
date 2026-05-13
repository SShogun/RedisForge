Demo workflow — RedisForge

Goal
- Produce a short, repeatable demo you can record and post on LinkedIn and X showing RedisForge features: item create/search, cache hit/miss, streams/audit, and observability metrics.

Prerequisites
- Windows with Docker Desktop running
- Git, Go 1.25+, PowerShell
- From repo root: Docker Compose files in `deployments/` are used to bring up Redis stack

Quick steps (1–2 minute demo)
1. Start Redis stack (Sentinel/Cluster if desired):

```powershell
cd .\deployments\
# choose the compose you want (simple or sentinel/cluster)
docker compose up -d
```

2. Build and run RedisForge (local):

```powershell
cd "${PWD}"
go build -o bin/redisforge ./cmd/redisforge
# run in one terminal
.
# start with default config (use env or flags as needed)
./bin/redisforge
```

3. Exercise the API (create, get, search):

```powershell
# create an item
curl -X POST http://localhost:8080/items -H "Content-Type: application/json" -d '{"id":"demo-1","name":"Demo Item","category":"demo","tags":["social","demo"]}'

# get item (cache hit)
curl http://localhost:8080/items/demo-1

# search (RediSearch)
curl 'http://localhost:8080/items/search?q=Demo'
```

4. Show observability (Prometheus metrics) and tracing
- Open `http://localhost:8080/metrics` to show Prometheus metrics (cache_hit/miss, redis latencies)
- If using Jaeger/OTel, show traces in the tracing UI (optional)

5. Demonstrate Streams/Audit (if enabled)
- Trigger an operation that writes to streams, then show consumer processing logs

6. Run a benchmark (optional)

```powershell
# run repository benchmarks (cache hit vs miss)
go test -bench=BenchmarkCacheItemRepo ./internal/repo -run=^$
```

Recording tips
- Keep the recording to 60–90 seconds for X/Twitter; 2–3 minutes for LinkedIn
- Start with a one-line caption on the editor showing the repo and goal
- Show terminal commands and quick results (curl output, metrics page, a log line for stream processing)
- Call out one metric (cache_hits_total) and explain the improvement

Assets to attach in post
- Short GIF (30s) showing: start server → create item → get item (cache hit) → open /metrics
- 1–2 screenshots: metrics dashboard and search result
- Link to repo and short instructions in thread or comment

Customization
- Replace endpoints/ports if your config differs
- Use `deployments/redis-sentinel/` compose for high-availability demo
- Use `--record` flag on terminal recording tools (e.g., ShareX, Peek, OBS) to create GIFs

Next steps
- Provide the narrative text you want shown on X/LinkedIn and I’ll produce ready-to-post templates and a short caption pack.