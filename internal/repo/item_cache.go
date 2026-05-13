package repo

import (
	"context"
	"errors"
	"log/slog"

	"github.com/SShogun/redisforge/internal/domain"
	"github.com/SShogun/redisforge/internal/redisx"
)

// CacheItemRepo decorates an ItemRepo with a RedisJSON cache.
type CacheItemRepo struct {
	fallback ItemRepo
	cache    *redisx.JSONStore
	logger   *slog.Logger
}

func NewCacheItemRepo(fallback ItemRepo, cache *redisx.JSONStore, logger *slog.Logger) *CacheItemRepo {
	return &CacheItemRepo{fallback: fallback, cache: cache, logger: logger}
}

func (r *CacheItemRepo) Create(ctx context.Context, item domain.Item) (domain.Item, error) {
	created, err := r.fallback.Create(ctx, item)
	if err != nil {
		return domain.Item{}, err
	}
	// Write-through to cache
	if err := r.cache.SetItem(ctx, created); err != nil {

		r.logger.WarnContext(ctx, "cache write-through failed", "item_id", created.ID, "err", err)
	}
	return created, nil
}

func (r *CacheItemRepo) GetByID(ctx context.Context, id string) (domain.Item, error) {
	// Cache-first
	item, err := r.cache.GetItem(ctx, id)
	if err == nil {
		return item, nil
	}
	if !errors.Is(err, domain.ErrNotFound) {

		r.logger.WarnContext(ctx, "cache read failed", "item_id", id, "err", err)
	}

	// Cache miss — hit the fallback
	item, err = r.fallback.GetByID(ctx, id)
	if err != nil {
		return domain.Item{}, err
	}

	// Populate cache for next read
	if cacheErr := r.cache.SetItem(ctx, item); cacheErr != nil {
		r.logger.WarnContext(ctx, "cache backfill failed", "item_id", id, "err", cacheErr)
	}
	return item, nil
}

func (r *CacheItemRepo) Update(ctx context.Context, item domain.Item) (domain.Item, error) {
	updated, err := r.fallback.Update(ctx, item)
	if err != nil {
		return domain.Item{}, err
	}

	if err := r.cache.DeleteItem(ctx, updated.ID); err != nil {
		r.logger.WarnContext(ctx, "cache invalidation failed", "item_id", updated.ID, "err", err)
	}
	return updated, nil
}

func (r *CacheItemRepo) Delete(ctx context.Context, id string) error {
	if err := r.fallback.Delete(ctx, id); err != nil {
		return err
	}
	_ = r.cache.DeleteItem(ctx, id)
	return nil
}

func (r *CacheItemRepo) List(ctx context.Context, offset, limit int) ([]domain.Item, error) {
	// List is never cached — caching lists is a cardinality footgun.

	return r.fallback.List(ctx, offset, limit)
}
