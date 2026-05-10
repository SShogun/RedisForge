package redisx

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

type BloomFilter struct {
	client redis.UniversalClient
	name   string
}

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
	res, err := b.client.Do(ctx, "BF.EXISTS", b.name, key).Int()
	if err != nil {
		return false, fmt.Errorf("BloomFilter.Exists: %w", err)
	}
	return res == 1, nil
}

// false -> key is deffo new
// true -> key might exist; need a postgres lookup
