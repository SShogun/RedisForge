# Social Templates

Use these when sharing RedisForge publicly. Keep the post specific, technical, and easy to verify.

## X Thread

```text
1/ Redis is deeper than SET/GET.

I built RedisForge to make the important Redis patterns visible in one small Go service:

- RedisJSON
- RedisBloom
- RediSearch
- Streams
- Pub/Sub
- Sentinel-ready clients
- metrics + tests

2/ The domain is intentionally tiny: Items.

That keeps the real topic visible:

- cache-aside reads
- idempotency checks
- JSON document indexing
- durable audit events
- topology-aware Redis clients

3/ The part I care about most:

The repo is also a Redis revision guide.

When Redis concepts get messy, I can reopen the implementation docs and walk through the system from HTTP handler to Redis module.

4/ Start here:

README: <repo link>
Docs path: docs/README.md
Implementation notes: docs/implementation/redis-patterns.md

If you are learning Redis for backend/system design, this should be useful.
```

## LinkedIn Post

```text
Redis is one of those tools that looks simple until you actually need to design with it.

I built RedisForge as a small production-shaped Go service for revising Redis architecture patterns in one place.

It covers:

- RedisJSON for document caching
- RedisBloom for idempotency pre-checks
- RediSearch for full-text and faceted search
- Redis Streams for durable audit processing
- Pub/Sub for ephemeral notifications
- Sentinel/Cluster-ready client wiring
- Prometheus metrics, OpenTelemetry hooks, and integration tests

The domain is intentionally small: Items.

That keeps the focus on Redis decisions instead of hiding everything behind business logic. The repo also has docs for implementation notes, Redis tradeoffs, profiling, demos, and project progress.

Repo: <repo link>
Start with: docs/README.md
```

## Short Launch Caption

```text
Built RedisForge: a Go + Redis Stack project for learning Redis architecture beyond SET/GET.

It covers RedisJSON, Bloom filters, RediSearch, Streams, Pub/Sub, Sentinel-ready clients, metrics, tests, and implementation docs.

Repo: <repo link>
```

## Hashtags

```text
#Redis #GoLang #BackendEngineering #OpenSource #SystemDesign
```

## Content Calendar Ideas

| Post | Topic | Asset |
| --- | --- | --- |
| 1 | RedisForge launch | README screenshot + terminal demo |
| 2 | Cache-aside pattern | `internal/repo/item_cache.go` snippet |
| 3 | Redis Streams recovery | `XAUTOCLAIM` explanation |
| 4 | RediSearch over JSON | search query example |
| 5 | Bloom filter idempotency | false-positive tradeoff note |
| 6 | Sentinel vs Cluster | docs comparison table |

