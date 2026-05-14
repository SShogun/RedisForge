package redisx

import (
	"context"
	"fmt"

	"github.com/SShogun/redisforge/internal/observability"
	"github.com/redis/go-redis/v9"
)

type BloomFilter struct {
	client redis.UniversalClient
	name   string
}

// NewBloomFilter returns a BloomFilter backed by the given client.
// Hash-tag note for Cluster mode:
// This is a singleton Bloom filter (not per-item), so hash-tag discipline does not apply.
// Compare with itemKey() in json.go which uses {id} hash tags to ensure all per-item operations
// land on the same slot. The Bloom filter spans the cluster and Redis handles synchronization.
func NewBloomFilter(client redis.UniversalClient, name string) *BloomFilter {
	return &BloomFilter{
		client: client,
		name:   name,
	}
}

func (b *BloomFilter) Reserve(ctx context.Context, errorRate float64, capacity int64) error {
	err := b.client.Do(ctx, "BF.RESERVE", b.name, errorRate, capacity).Err()
	if err != nil && err.Error() == "ERR item exists" {
		return nil
	}
	if err != nil {
		return fmt.Errorf("BloomFilter.Reserve: %w", err)
	}
	return nil
}

func (b *BloomFilter) Add(ctx context.Context, key string) error {
	if err := b.client.Do(ctx, "BF.ADD", b.name, key).Err(); err != nil {
		return fmt.Errorf("BloomFilter.Add: %w", err)
	}
	return nil
}

func (b *BloomFilter) Exists(ctx context.Context, key string) (bool, error) {
	res, err := b.client.Do(ctx, "BF.EXISTS", b.name, key).Bool()
	if err == nil {
		observability.RecordBloomCheck(res)
	}
	if err != nil {
		return false, fmt.Errorf("BloomFilter.Exists: %w", err)
	}
	return res, nil
}

// false -> key is deffo new
// true -> key might exist; need a postgres lookup
