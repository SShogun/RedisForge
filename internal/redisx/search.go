package redisx

import (
	"context"
	"fmt"
	"strings"

	"github.com/redis/go-redis/v9"
)

const itemIndexName = "idx:items"

// Hash-tag note for Cluster mode:
// The RediSearch index is a singleton (not per-item) and spans the entire cluster.
// Individual item keys use "item:{id}" format to ensure hash-tag discipline.
// The index itself is synchronized across slots automatically by Redis Cluster.
// No hash-tag needed for the index key.

type SearchStore struct {
	client redis.UniversalClient
}

func NewSearchStore(client redis.UniversalClient) *SearchStore {
	return &SearchStore{
		client: client,
	}
}

func (s *SearchStore) EnsureIndex(ctx context.Context) error {
	err := s.client.Do(ctx,
		"FT.CREATE", itemIndexName,
		"ON", "JSON",
		"PREFIX", "1", "item:{",
		"SCHEMA",
		"$.name", "AS", "name", "TEXT", "WEIGHT", "2.0",
		"$.category", "AS", "category", "TAG",
		"$.tags[*]", "AS", "tags", "TAG",
		"$.score", "AS", "score", "NUMERIC", "SORTABLE",
	).Err()
	if err != nil {
		if strings.Contains(err.Error(), "Index already exists") {
			return nil
		}
		return fmt.Errorf("SearchStore.EnsureIndex: %w", err)
	}
	return nil
}

type SearchResult struct {
	ID      string
	RawJSON string
}

//   "widget"                   → full text search
//   "@category:{tools}"        → exact tag match
//   "@score:[5 10]"            → numeric range
//   "widget @category:{tools}" → combined

func (s *SearchStore) Search(ctx context.Context, query string, offset, limit int) ([]SearchResult, int64, error) {
	res, err := s.client.Do(ctx,
		"FT.SEARCH", itemIndexName, query,
		"LIMIT", offset, limit,
		"RETURN", "1", "$",
	).Slice()
	if err != nil {
		return nil, 0, fmt.Errorf("SearchStore.Search: %w", err)
	}
	if len(res) == 0 {
		return nil, 0, nil
	}

	total, _ := res[0].(int64)
	var results []SearchResult

	for i := 1; i < len(res); i += 2 {
		key, _ := res[i].(string)
		id := strings.TrimPrefix(strings.TrimSuffix(key, "}"), "item:{")

		var rawJSON string
		if i+1 < len(res) {
			fields, _ := res[i+1].([]interface{})
			if len(fields) >= 2 {
				rawJSON, _ = fields[1].(string)
			}
		}
		results = append(results, SearchResult{ID: id, RawJSON: rawJSON})
	}
	return results, total, nil
}
