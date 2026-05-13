package repo

import (
	"context"

	"github.com/SShogun/redisforge/internal/domain"
)

// List of all functions
type ItemRepo interface {
	Create(ctx context.Context, item domain.Item) (domain.Item, error)
	GetByID(ctx context.Context, id string) (domain.Item, error)
	Update(ctx context.Context, item domain.Item) (domain.Item, error)
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, offset, limit int) ([]domain.Item, error)
}
