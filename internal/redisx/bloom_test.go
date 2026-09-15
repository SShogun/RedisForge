package redisx_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/SShogun/redisforge/internal/redisx"
	"github.com/SShogun/redisforge/internal/testutil"
)

func TestBloomFilterIntegration(t *testing.T) {
	client := testutil.StartRedisStack(t)
	ctx := context.Background()

	t.Run("new key is not found", func(t *testing.T) {
		bf := redisx.NewBloomFilter(client, "bf:test-new")
		if err := bf.Reserve(ctx, 0.001, 1_000); err != nil {
			t.Fatalf("Reserve: %v", err)
		}
		exists, err := bf.Exists(ctx, "brand-new-key")
		if err != nil {
			t.Fatalf("Exists: %v", err)
		}
		if exists {
			t.Fatal("brand-new key should not exist")
		}
	})

	t.Run("add then exists", func(t *testing.T) {
		bf := redisx.NewBloomFilter(client, "bf:test-add")
		if err := bf.Reserve(ctx, 0.001, 1_000); err != nil {
			t.Fatalf("Reserve: %v", err)
		}
		if err := bf.Add(ctx, "my-key"); err != nil {
			t.Fatalf("Add: %v", err)
		}
		exists, err := bf.Exists(ctx, "my-key")
		if err != nil {
			t.Fatalf("Exists: %v", err)
		}
		if !exists {
			t.Fatal("added key must exist in bloom filter")
		}
	})

	t.Run("reserve is idempotent", func(t *testing.T) {
		bf := redisx.NewBloomFilter(client, "bf:test-reserve")
		if err := bf.Reserve(ctx, 0.001, 1_000); err != nil {
			t.Fatalf("first Reserve: %v", err)
		}
		if err := bf.Reserve(ctx, 0.001, 1_000); err != nil {
			t.Fatalf("second Reserve should be idempotent: %v", err)
		}
	})

	t.Run("added keys have no false negatives", func(t *testing.T) {
		bf := redisx.NewBloomFilter(client, "bf:test-no-false-negatives")
		if err := bf.Reserve(ctx, 0.01, 100); err != nil {
			t.Fatalf("Reserve: %v", err)
		}

		for i := 0; i < 100; i++ {
			key := fmt.Sprintf("key-%d", i)
			if err := bf.Add(ctx, key); err != nil {
				t.Fatalf("Add(%q): %v", key, err)
			}
		}
		for i := 0; i < 100; i++ {
			key := fmt.Sprintf("key-%d", i)
			exists, err := bf.Exists(ctx, key)
			if err != nil {
				t.Fatalf("Exists(%q): %v", key, err)
			}
			if !exists {
				t.Fatalf("false negative for inserted key %q", key)
			}
		}
	})
}
