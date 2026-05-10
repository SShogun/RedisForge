package redisx_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/SShogun/redisforge/internal/redisx"
	"github.com/redis/go-redis/v9"
)

func TestBloom_NewKeyIsNotFound(t *testing.T) {
	addr := startRedisStack(t) // reuse helper from json_test.go
	client := redis.NewClient(&redis.Options{Addr: addr})
	bf := redisx.NewBloomFilter(client, "bf:test")
	ctx := context.Background()

	_ = bf.Reserve(ctx, 0.001, 1_000_000)

	exists, err := bf.Exists(ctx, "brand-new-key")
	if err != nil {
		t.Fatalf("Exists: %v", err)
	}
	if exists {
		t.Error("brand new key should not exist")
	}
}

func TestBloom_AddThenExists(t *testing.T) {
	addr := startRedisStack(t)
	client := redis.NewClient(&redis.Options{Addr: addr})
	bf := redisx.NewBloomFilter(client, "bf:test2")
	ctx := context.Background()

	_ = bf.Reserve(ctx, 0.001, 1_000_000)
	_ = bf.Add(ctx, "my-key")

	exists, _ := bf.Exists(ctx, "my-key")
	if !exists {
		t.Error("added key must exist in bloom filter")
	}
}

// TestBloom_FalsePositive intentionally creates a tiny filter so
// the false-positive rate is extremely high. This proves you understand
// the failure mode: Exists can return true for a key never added.
func TestBloom_FalsePositive(t *testing.T) {
	addr := startRedisStack(t)
	client := redis.NewClient(&redis.Options{Addr: addr})
	bf := redisx.NewBloomFilter(client, "bf:tiny")
	ctx := context.Background()

	// Tiny filter: capacity=5, error_rate=0.99 → almost guaranteed false positives
	_ = bf.Reserve(ctx, 0.99, 5)

	// Fill it with 5 entries
	for i := 0; i < 5; i++ {
		_ = bf.Add(ctx, fmt.Sprintf("key-%d", i))
	}

	// Now check a key we never added — with 0.99 error rate it will
	// almost certainly return true (false positive).
	fpCount := 0
	for i := 100; i < 200; i++ {
		exists, _ := bf.Exists(ctx, fmt.Sprintf("unseen-%d", i))
		if exists {
			fpCount++
		}
	}
	t.Logf("False positives detected: %d / 100 (expected many with errorRate=0.99)", fpCount)
	// We don't assert a specific number — we log it to prove the concept.
	// In your docs/redis-decisions.md, record the observed false-positive rate.
}
