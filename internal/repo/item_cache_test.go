package repo

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/SShogun/redisforge/internal/domain"
)

type stubRepo struct {
	createFn func(context.Context, domain.Item) (domain.Item, error)
	getFn    func(context.Context, string) (domain.Item, error)
	updateFn func(context.Context, domain.Item) (domain.Item, error)
	deleteFn func(context.Context, string) error
	listFn   func(context.Context, int, int) ([]domain.Item, error)
}

func (s stubRepo) Create(ctx context.Context, item domain.Item) (domain.Item, error) {
	return s.createFn(ctx, item)
}

func (s stubRepo) GetByID(ctx context.Context, id string) (domain.Item, error) {
	return s.getFn(ctx, id)
}

func (s stubRepo) Update(ctx context.Context, item domain.Item) (domain.Item, error) {
	return s.updateFn(ctx, item)
}

func (s stubRepo) Delete(ctx context.Context, id string) error {
	return s.deleteFn(ctx, id)
}

func (s stubRepo) List(ctx context.Context, offset, limit int) ([]domain.Item, error) {
	return s.listFn(ctx, offset, limit)
}

type fakeCache struct {
	items       map[string]domain.Item
	getErr      error
	setErr      error
	deleteErr   error
	setCalls    int
	deleteCalls int
}

func (f *fakeCache) SetItem(_ context.Context, item domain.Item) error {
	if f.items == nil {
		f.items = make(map[string]domain.Item)
	}
	f.items[item.ID] = item
	f.setCalls++
	return f.setErr
}

func (f *fakeCache) GetItem(_ context.Context, id string) (domain.Item, error) {
	if f.getErr != nil {
		return domain.Item{}, f.getErr
	}
	if item, ok := f.items[id]; ok {
		return item, nil
	}
	return domain.Item{}, domain.ErrNotFound
}

func (f *fakeCache) DeleteItem(_ context.Context, id string) error {
	delete(f.items, id)
	f.deleteCalls++
	return f.deleteErr
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestCacheItemRepo_GetByID_HitsCache(t *testing.T) {
	ctx := context.Background()
	cache := &fakeCache{items: map[string]domain.Item{"item-1": {ID: "item-1", Name: "cached"}}}
	repo := NewCacheItemRepo(stubRepo{
		getFn: func(context.Context, string) (domain.Item, error) {
			t.Fatal("fallback repo should not be called on cache hit")
			return domain.Item{}, nil
		},
		createFn: func(context.Context, domain.Item) (domain.Item, error) { return domain.Item{}, nil },
		updateFn: func(context.Context, domain.Item) (domain.Item, error) { return domain.Item{}, nil },
		deleteFn: func(context.Context, string) error { return nil },
		listFn:   func(context.Context, int, int) ([]domain.Item, error) { return nil, nil },
	}, cache, testLogger())

	got, err := repo.GetByID(ctx, "item-1")
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Name != "cached" {
		t.Fatalf("expected cached item, got %#v", got)
	}
}

func TestCacheItemRepo_GetByID_BackfillsCache(t *testing.T) {
	ctx := context.Background()
	fallback := NewMemoryItemRepo()
	created, err := fallback.Create(ctx, domain.Item{ID: "item-2", Name: "source"})
	if err != nil {
		t.Fatalf("seed fallback: %v", err)
	}
	cache := &fakeCache{getErr: domain.ErrNotFound}
	repo := NewCacheItemRepo(fallback, cache, testLogger())

	got, err := repo.GetByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Name != "source" {
		t.Fatalf("expected fallback item, got %#v", got)
	}
	if cache.setCalls != 1 {
		t.Fatalf("expected cache backfill, got %d set calls", cache.setCalls)
	}
}

func TestCacheItemRepo_Update_InvalidatesCache(t *testing.T) {
	ctx := context.Background()
	fallback := NewMemoryItemRepo()
	created, err := fallback.Create(ctx, domain.Item{ID: "item-3", Name: "before"})
	if err != nil {
		t.Fatalf("seed fallback: %v", err)
	}
	cache := &fakeCache{items: map[string]domain.Item{created.ID: created}}
	repo := NewCacheItemRepo(fallback, cache, testLogger())

	updated, err := repo.Update(ctx, domain.Item{ID: created.ID, Name: "after", Version: created.Version})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Name != "after" {
		t.Fatalf("expected updated item, got %#v", updated)
	}
	if cache.deleteCalls != 1 {
		t.Fatalf("expected cache invalidation, got %d delete calls", cache.deleteCalls)
	}
}

func TestCacheItemRepo_Create_WritesThrough(t *testing.T) {
	ctx := context.Background()
	fallback := NewMemoryItemRepo()
	cache := &fakeCache{}
	repo := NewCacheItemRepo(fallback, cache, testLogger())

	created, err := repo.Create(ctx, domain.Item{ID: "item-4", Name: "write-through"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if created.ID != "item-4" {
		t.Fatalf("expected created item, got %#v", created)
	}
	if cache.setCalls != 1 {
		t.Fatalf("expected cache write-through, got %d set calls", cache.setCalls)
	}
}

// Benchmarks: Cache hit vs miss performance

// BenchmarkCacheItemRepo_GetByID_CacheHit measures performance when item is in cache.
// This is the fast path: immediate return without fallback store access.
func BenchmarkCacheItemRepo_GetByID_CacheHit(b *testing.B) {
	ctx := context.Background()
	item := domain.Item{ID: "bench-hit", Name: "Cached", Category: "perf", Score: 9.9}
	cache := &fakeCache{items: map[string]domain.Item{item.ID: item}}
	fallback := stubRepo{
		getFn: func(context.Context, string) (domain.Item, error) {
			b.Fatal("cache hit should not access fallback")
			return domain.Item{}, nil
		},
		createFn: func(context.Context, domain.Item) (domain.Item, error) { return domain.Item{}, nil },
		updateFn: func(context.Context, domain.Item) (domain.Item, error) { return domain.Item{}, nil },
		deleteFn: func(context.Context, string) error { return nil },
		listFn:   func(context.Context, int, int) ([]domain.Item, error) { return nil, nil },
	}
	repo := NewCacheItemRepo(fallback, cache, testLogger())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = repo.GetByID(ctx, item.ID)
	}
}

// BenchmarkCacheItemRepo_GetByID_CacheMiss measures performance on cache miss.
// This requires fallback store lookup, which is slower than cache hit.
func BenchmarkCacheItemRepo_GetByID_CacheMiss(b *testing.B) {
	ctx := context.Background()
	item := domain.Item{ID: "bench-miss", Name: "Fallback", Category: "perf", Score: 8.8}

	// Seed fallback store but leave cache empty
	fallback := NewMemoryItemRepo()
	_, _ = fallback.Create(ctx, item)

	cache := &fakeCache{getErr: domain.ErrNotFound}
	repo := NewCacheItemRepo(fallback, cache, testLogger())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = repo.GetByID(ctx, item.ID)
	}
}

// BenchmarkCacheItemRepo_GetByID_BackfillPath measures cache miss with backfill.
// This includes fallback lookup + cache write, the most expensive single path.
func BenchmarkCacheItemRepo_GetByID_BackfillPath(b *testing.B) {
	ctx := context.Background()
	item := domain.Item{ID: "bench-backfill", Name: "ToCache", Category: "perf", Score: 7.7}

	fallback := NewMemoryItemRepo()
	_, _ = fallback.Create(ctx, item)

	cache := &fakeCache{}
	repo := NewCacheItemRepo(fallback, cache, testLogger())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = repo.GetByID(ctx, item.ID)
	}
}

// BenchmarkCacheItemRepo_GetByID_Contention measures cache hit under parallel access.
// Simulates real-world load where multiple goroutines query the same popular item.
func BenchmarkCacheItemRepo_GetByID_Contention(b *testing.B) {
	ctx := context.Background()
	item := domain.Item{ID: "bench-contention", Name: "Popular", Category: "perf", Score: 9.0}
	cache := &fakeCache{items: map[string]domain.Item{item.ID: item}}
	fallback := stubRepo{
		getFn: func(context.Context, string) (domain.Item, error) {
			return domain.Item{}, nil
		},
		createFn: func(context.Context, domain.Item) (domain.Item, error) { return domain.Item{}, nil },
		updateFn: func(context.Context, domain.Item) (domain.Item, error) { return domain.Item{}, nil },
		deleteFn: func(context.Context, string) error { return nil },
		listFn:   func(context.Context, int, int) ([]domain.Item, error) { return nil, nil },
	}
	repo := NewCacheItemRepo(fallback, cache, testLogger())

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = repo.GetByID(ctx, item.ID)
		}
	})
}
