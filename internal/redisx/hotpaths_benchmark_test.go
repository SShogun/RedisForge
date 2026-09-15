package redisx_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/SShogun/redisforge/internal/domain"
	"github.com/SShogun/redisforge/internal/redisx"
	"github.com/SShogun/redisforge/internal/testutil"
)

// BenchmarkRedisHotPaths measures application-observed latency for the Redis
// operations that dominate RedisForge's steady-state paths. Results include the
// Go client, Docker/network round-trip, Redis command execution, and decoding.
func BenchmarkRedisHotPaths(b *testing.B) {
	client := testutil.StartRedisStack(b)
	ctx := context.Background()

	jsonStore := redisx.NewJSONStore(client)
	bloom := redisx.NewBloomFilter(client, "bf:benchmark")
	searchStore := redisx.NewSearchStore(client)
	stream := redisx.NewStreamClient(client)

	if err := bloom.Reserve(ctx, 0.001, 10_000); err != nil {
		b.Fatalf("Bloom Reserve: %v", err)
	}
	if err := bloom.Add(ctx, "benchmark-key"); err != nil {
		b.Fatalf("Bloom Add: %v", err)
	}
	if err := searchStore.EnsureIndex(ctx); err != nil {
		b.Fatalf("EnsureIndex: %v", err)
	}

	seed := domain.Item{
		ID:       "bench-hot",
		Name:     "Benchmark Widget",
		Category: "benchmark",
		Score:    9.5,
		Tags:     []string{"benchmark", "hot"},
	}
	if err := jsonStore.SetItem(ctx, seed); err != nil {
		b.Fatalf("seed JSON item: %v", err)
	}

	for i := 0; i < 100; i++ {
		item := domain.Item{
			ID:       fmt.Sprintf("bench-search-%03d", i),
			Name:     fmt.Sprintf("Benchmark Search Item %03d", i),
			Category: "benchmark",
			Score:    float64(i % 10),
			Tags:     []string{"benchmark"},
		}
		if err := jsonStore.SetItem(ctx, item); err != nil {
			b.Fatalf("seed search item %d: %v", i, err)
		}
	}
	waitForSearchTotal(b, searchStore, "@category:{benchmark}", 101)

	b.Run("JSON_GET", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			if _, err := jsonStore.GetItem(ctx, seed.ID); err != nil {
				b.Fatalf("GetItem: %v", err)
			}
		}
	})

	b.Run("JSON_SET", func(b *testing.B) {
		b.ReportAllocs()
		item := seed
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			item.Score = float64(i % 100)
			if err := jsonStore.SetItem(ctx, item); err != nil {
				b.Fatalf("SetItem: %v", err)
			}
		}
	})

	b.Run("BLOOM_EXISTS", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			if _, err := bloom.Exists(ctx, "benchmark-key"); err != nil {
				b.Fatalf("Exists: %v", err)
			}
		}
	})

	b.Run("FT_SEARCH", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			if _, _, err := searchStore.Search(ctx, "Benchmark", 0, 10); err != nil {
				b.Fatalf("Search: %v", err)
			}
		}
	})

	b.Run("XADD", func(b *testing.B) {
		b.ReportAllocs()
		fields := map[string]interface{}{"event": `{"event_id":"benchmark","action":"updated"}`}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			if _, err := stream.Append(ctx, "benchmark-audit-events", fields); err != nil {
				b.Fatalf("Append: %v", err)
			}
		}
	})
}
