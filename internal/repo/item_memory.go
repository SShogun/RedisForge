package repo

// item_memory.go is an in-memory ItemRepo used in tests and as the
import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/SShogun/redisforge/internal/domain"
)

// MemoryItemRepo is a thread-safe in-memory store.
type MemoryItemRepo struct {
	mu    sync.RWMutex
	items map[string]domain.Item
}

func NewMemoryItemRepo() *MemoryItemRepo {
	return &MemoryItemRepo{items: make(map[string]domain.Item)}
}

func (r *MemoryItemRepo) Create(ctx context.Context, item domain.Item) (domain.Item, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.items[item.ID]; exists {
		return domain.Item{}, domain.ErrDuplicate
	}
	item.Version = 1
	item.CreatedAt = time.Now().UTC()
	item.UpdatedAt = item.CreatedAt
	r.items[item.ID] = item
	return item, nil
}

func (r *MemoryItemRepo) GetByID(ctx context.Context, id string) (domain.Item, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	item, ok := r.items[id]
	if !ok {
		return domain.Item{}, domain.ErrNotFound
	}
	return item, nil
}

func (r *MemoryItemRepo) Update(ctx context.Context, item domain.Item) (domain.Item, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	existing, ok := r.items[item.ID]
	if !ok {
		return domain.Item{}, domain.ErrNotFound
	}
	// Optimistic concurrency: version must match
	if existing.Version != item.Version {
		return domain.Item{}, fmt.Errorf("%w: expected version %d, got %d",
			domain.ErrConflict, existing.Version, item.Version)
	}
	item.Version++
	item.UpdatedAt = time.Now().UTC()
	r.items[item.ID] = item
	return item, nil
}

func (r *MemoryItemRepo) Delete(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.items[id]; !ok {
		return domain.ErrNotFound
	}
	delete(r.items, id)
	return nil
}

func (r *MemoryItemRepo) List(ctx context.Context, offset, limit int) ([]domain.Item, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	all := make([]domain.Item, 0, len(r.items))
	for _, v := range r.items {
		all = append(all, v)
	}
	if offset >= len(all) {
		return []domain.Item{}, nil
	}
	end := offset + limit
	if end > len(all) {
		end = len(all)
	}
	return all[offset:end], nil
}
