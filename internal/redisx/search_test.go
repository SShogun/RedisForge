package redisx_test

import (
	"context"
	"testing"
	"time"

	"github.com/SShogun/redisforge/internal/domain"
	"github.com/SShogun/redisforge/internal/redisx"
	"github.com/SShogun/redisforge/internal/testutil"
)

func TestSearchStoreIntegration(t *testing.T) {
	client := testutil.StartRedisStack(t)
	ctx := context.Background()
	jsonStore := redisx.NewJSONStore(client)
	searchStore := redisx.NewSearchStore(client)

	if err := searchStore.EnsureIndex(ctx); err != nil {
		t.Fatalf("EnsureIndex: %v", err)
	}
	if err := searchStore.EnsureIndex(ctx); err != nil {
		t.Fatalf("EnsureIndex should be idempotent: %v", err)
	}

	items := []domain.Item{
		{ID: "search-1", Name: "Steel Hammer", Category: "tools", Score: 9.5, Tags: []string{"sale", "metal"}},
		{ID: "search-2", Name: "Precision Screwdriver", Category: "tools", Score: 8.0, Tags: []string{"new"}},
		{ID: "search-3", Name: "Desk Lamp", Category: "home", Score: 7.0, Tags: []string{"sale"}},
	}
	for _, item := range items {
		if err := jsonStore.SetItem(ctx, item); err != nil {
			t.Fatalf("seed %s: %v", item.ID, err)
		}
	}
	waitForSearchTotal(t, searchStore, "*", int64(len(items)))

	tests := []struct {
		name    string
		query   string
		wantIDs map[string]bool
	}{
		{name: "full text", query: "Hammer", wantIDs: map[string]bool{"search-1": true}},
		{name: "category tag", query: "@category:{tools}", wantIDs: map[string]bool{"search-1": true, "search-2": true}},
		{name: "array tag", query: "@tags:{sale}", wantIDs: map[string]bool{"search-1": true, "search-3": true}},
		{name: "score range", query: "@score:[8 10]", wantIDs: map[string]bool{"search-1": true, "search-2": true}},
		{name: "combined", query: "Hammer @category:{tools}", wantIDs: map[string]bool{"search-1": true}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, total, err := searchStore.Search(ctx, tt.query, 0, 10)
			if err != nil {
				t.Fatalf("Search(%q): %v", tt.query, err)
			}
			if total != int64(len(tt.wantIDs)) {
				t.Fatalf("Search(%q): expected total %d, got %d", tt.query, len(tt.wantIDs), total)
			}
			if len(results) != len(tt.wantIDs) {
				t.Fatalf("Search(%q): expected %d results, got %d", tt.query, len(tt.wantIDs), len(results))
			}
			for _, result := range results {
				if !tt.wantIDs[result.ID] {
					t.Fatalf("Search(%q): unexpected result ID %q", tt.query, result.ID)
				}
				if result.RawJSON == "" {
					t.Fatalf("Search(%q): result %q missing returned JSON", tt.query, result.ID)
				}
			}
		})
	}
}

func waitForSearchTotal(tb testing.TB, store *redisx.SearchStore, query string, want int64) {
	tb.Helper()
	ctx := context.Background()
	deadline := time.Now().Add(5 * time.Second)
	var lastTotal int64
	var lastErr error
	for time.Now().Before(deadline) {
		_, total, err := store.Search(ctx, query, 0, 100)
		lastTotal = total
		lastErr = err
		if err == nil && total >= want {
			return
		}
		time.Sleep(25 * time.Millisecond)
	}
	tb.Fatalf("search index did not reach %d results: total=%d err=%v", want, lastTotal, lastErr)
}
