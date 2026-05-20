# Redis Cluster Demo

This directory contains a Docker Compose file that boots a **6-node Redis Cluster** (3 masters, 3 replicas) for local learning and experimentation.

## Why Cluster?

| Topology | What It Solves | Limitation |
|----------|---------------|------------|
| **Single** | Simplicity | No HA, no scale |
| **Sentinel** | High availability (automatic failover) | Single master, no horizontal scale |
| **Cluster** | HA **and** horizontal scale | Key design must respect hash slots |

RedisForge's Go client (`internal/redisx/client.go`) already supports all three via `redis.UniversalClient`. This demo lets you experience Cluster first-hand.

## Quick Start

### 1. Start the 6 nodes

```powershell
cd deployments/redis-cluster
docker compose up -d
```

### 2. Form the cluster

After the containers are running (~5 seconds), tell Redis to create the cluster topology:

```powershell
docker compose exec redis-node-1 redis-cli --cluster create `
  redis-node-1:6379 redis-node-2:6379 redis-node-3:6379 `
  redis-node-4:6379 redis-node-5:6379 redis-node-6:6379 `
  --cluster-replicas 1 --cluster-yes
```

This distributes the 16,384 hash slots across 3 masters and assigns 1 replica to each.

### 3. Verify

```powershell
# Cluster health
docker compose exec redis-node-1 redis-cli cluster info

# Node topology (which nodes are masters, which are replicas)
docker compose exec redis-node-1 redis-cli cluster nodes
```

You should see `cluster_state:ok` and 3 master + 3 slave entries.

### 4. Connect RedisForge

Set these environment variables to point the app at the cluster:

```powershell
$env:REDIS_CLUSTER_ENABLED = "true"
$env:REDIS_CLUSTER_ADDRS = "localhost:7001,localhost:7002,localhost:7003,localhost:7004,localhost:7005,localhost:7006"
```

Then start the app normally. The `go-redis` ClusterClient will discover the full topology from any seed node.

### 5. Teardown

```powershell
docker compose down -v
```

---

## Key Concepts to Explore

### Hash Slots

Redis Cluster divides the keyspace into **16,384 slots**. Each master owns a range of slots. When you write a key, Redis hashes the key name to determine which slot (and therefore which master) owns it.

```
SET item:abc123 "hello"
→ CRC16("item:abc123") % 16384 = slot 9712
→ routed to whichever master owns slot 9712
```

### Hash Tags

If a key contains `{...}`, only the substring inside the braces is hashed. This lets you force related keys onto the same slot:

```
item:{user-42}:json   → hashes "user-42" → slot X
item:{user-42}:audit  → hashes "user-42" → slot X  (same!)
```

This is **required** for multi-key operations like `MGET`, pipelines touching multiple keys, or Lua scripts operating on multiple keys. Without hash tags, those operations would fail with a `CROSSSLOT` error.

### What Happens When a Master Dies?

Try it yourself:

```powershell
# Kill master node 1
docker compose stop redis-node-1

# Check cluster state — it will promote the replica
docker compose exec redis-node-2 redis-cli cluster nodes

# Bring it back — it rejoins as a replica
docker compose start redis-node-1
```

---

## Important: Redis Stack Modules in Cluster

> **Note:** This demo uses plain `redis:7-alpine` instead of `redis/redis-stack`.
>
> Redis Stack modules (RedisJSON, RedisBloom, RediSearch) do **not** support native Redis Cluster mode. In production, you would either:
> 1. Use **Redis Enterprise** which handles module sharding transparently
> 2. Use **Sentinel** for HA with modules on a single logical master
> 3. Use Cluster for raw Redis commands and a separate single-node Redis Stack for module operations
>
> RedisForge's main `docker-compose.yml` uses Redis Stack (single-node) for full module support. This cluster demo is specifically for learning Cluster topology, hash slots, and failover behavior.

---

## Connecting with redis-cli

You can connect to any node and the cluster will redirect you:

```powershell
# Connect to node 1 with cluster-mode following redirects
docker compose exec redis-node-1 redis-cli -c

# Try writing a key
127.0.0.1:6379> SET mykey "hello"
# Redis may redirect: -> Redirected to slot [14687] located at redis-node-3:6379

# Check which slot a key maps to
127.0.0.1:6379> CLUSTER KEYSLOT mykey
(integer) 14687

# Check hash tag behavior
127.0.0.1:6379> CLUSTER KEYSLOT {user:1}:profile
(integer) 8106
127.0.0.1:6379> CLUSTER KEYSLOT {user:1}:settings
(integer) 8106
```
