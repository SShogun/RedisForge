# RedisForge Demo Workflow

Goal: produce a short, repeatable demo that shows RedisForge as a useful Redis learning project, not just another CRUD API.

## Demo Story

In 60 to 90 seconds, show this arc:

```text
start stack
  -> create item
  -> fetch item through cache-aside path
  -> search with RediSearch
  -> show metrics or logs
  -> point viewers to docs/README.md
```

## Prerequisites

- Docker Desktop running
- Go 1.25+
- PowerShell on Windows
- Repo cloned locally

## Quick Demo

Start the app, Redis Stack, Prometheus, and Grafana:

```powershell
.\make.ps1 up
```

The API is now available on `http://localhost:8080`.

Create an item:

```powershell
Invoke-RestMethod -Uri "http://localhost:8080/v1/items" -Method Post -ContentType "application/json" -Body '{
  "name": "Redis Streams Notebook",
  "category": "learning",
  "score": 9.7,
  "tags": ["redis", "streams", "search"],
  "idempotency_key": "demo-001"
}'
```

Fetch items:

```powershell
Invoke-RestMethod "http://localhost:8080/v1/items"
```

Search:

```powershell
Invoke-RestMethod "http://localhost:8080/v1/items/search?q=Redis"
```

Show metrics:

```powershell
Invoke-RestMethod "http://localhost:8080/metrics"
```

Run tests:

```powershell
.\make.ps1 test
```

Stop the stack:

```powershell
.\make.ps1 down
```

## What To Show On Screen

Use four quick cuts:

1. README opening: show the purpose and docs table.
2. Code: open `internal/redisx/streams.go` or `internal/repo/item_cache.go`.
3. Terminal: create/search item successfully.
4. Docs: open `docs/implementation/redis-patterns.md`.

## Recording Tips

- Keep the first demo under 90 seconds.
- Use a large terminal font.
- Avoid explaining every Redis module in one video.
- Pick one hook per video: Streams recovery, Bloom idempotency, RediSearch, or cache-aside.
- End by showing `docs/README.md`, because that is the repo's learning path.

## Demo Hooks

| Hook | Angle |
| --- | --- |
| "Redis beyond SET/GET" | Show JSON, Search, Bloom, and Streams in one small service |
| "Cache-aside without hand-waving" | Show repository code and then the API call |
| "Streams vs Pub/Sub" | Explain durable audit events vs ephemeral notifications |
| "Redis as architecture practice" | Show code, docs, tests, and metrics together |
