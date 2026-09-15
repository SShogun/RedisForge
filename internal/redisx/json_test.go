package redisx_test

import (
	"context"
	"errors"
	"testing"

	"github.com/SShogun/redisforge/internal/domain"
	"github.com/SShogun/redisforge/internal/redisx"
	"github.com/SShogun/redisforge/internal/testutil"
)

func TestJSONStoreIntegration(t *testing.T) {
	client := testutil.StartRedisStack(t)
	store := redisx.NewJSONStore(client)
	ctx := context.Background()

	t.Run("set and get", func(t *testing.T) {
		item := domain.Item{
			ID:       "test-1",
			Name:     "Widget",
			Category: "tools",
			Score:    9.5,
			Tags:     []string{"new"},
			Version:  1,
		}

		if err := store.SetItem(ctx, item); err != nil {
			t.Fatalf("SetItem: %v", err)
		}
		got, err := store.GetItem(ctx, item.ID)
		if err != nil {
			t.Fatalf("GetItem: %v", err)
		}
		if got.Name != item.Name || got.Score != item.Score {
			t.Fatalf("unexpected item: %#v", got)
		}
	})

	t.Run("append tag", func(t *testing.T) {
		item := domain.Item{ID: "tag-test", Tags: []string{"alpha"}}
		if err := store.SetItem(ctx, item); err != nil {
			t.Fatalf("SetItem: %v", err)
		}
		if err := store.AppendTag(ctx, item.ID, "beta"); err != nil {
			t.Fatalf("AppendTag: %v", err)
		}

		got, err := store.GetItem(ctx, item.ID)
		if err != nil {
			t.Fatalf("GetItem: %v", err)
		}
		if len(got.Tags) != 2 || got.Tags[1] != "beta" {
			t.Fatalf("expected [alpha beta], got %v", got.Tags)
		}
	})

	t.Run("increment score", func(t *testing.T) {
		item := domain.Item{ID: "score-test", Score: 10}
		if err := store.SetItem(ctx, item); err != nil {
			t.Fatalf("SetItem: %v", err)
		}

		score, err := store.IncrScore(ctx, item.ID, 2.5)
		if err != nil {
			t.Fatalf("IncrScore: %v", err)
		}
		if score != 12.5 {
			t.Fatalf("expected score 12.5, got %v", score)
		}
	})

	t.Run("delete", func(t *testing.T) {
		item := domain.Item{ID: "delete-test", Name: "delete me"}
		if err := store.SetItem(ctx, item); err != nil {
			t.Fatalf("SetItem: %v", err)
		}
		if err := store.DeleteItem(ctx, item.ID); err != nil {
			t.Fatalf("DeleteItem: %v", err)
		}
		if _, err := store.GetItem(ctx, item.ID); !errors.Is(err, domain.ErrNotFound) {
			t.Fatalf("expected ErrNotFound after delete, got %v", err)
		}
	})

	t.Run("not found", func(t *testing.T) {
		if _, err := store.GetItem(ctx, "does-not-exist"); !errors.Is(err, domain.ErrNotFound) {
			t.Fatalf("expected ErrNotFound, got %v", err)
		}
	})
}
