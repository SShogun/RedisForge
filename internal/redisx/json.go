package redisx

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/SShogun/redisforge/internal/domain"
	"github.com/redis/go-redis/v9"
)

type JSONStore struct {
	client redis.UniversalClient
}

func NewJSONStore(client redis.UniversalClient) *JSONStore {
	return &JSONStore{
		client: client,
	}
}

func itemKey(id string) string {
	return fmt.Sprintf("item:{%s}", id)
}

// converts JSON.SET item:{123}
func (s *JSONStore) SetItem(ctx context.Context, item domain.Item) error {
	data, err := json.Marshal(item)
	if err != nil {
		return fmt.Errorf("JSONStore.SetItem: marshal: %w", err)
	}

	if err := s.client.Do(ctx, "JSON.SET", itemKey(item.ID), "$", string(data)).Err(); err != nil {
		return fmt.Errorf("JSONStore.SetItem: %w", err)
	}
	return nil
}
func (s *JSONStore) GetItem(ctx context.Context, id string) (domain.Item, error) {
	raw, err := s.client.Do(ctx, "JSON.GET", itemKey(id), "$").Text()
	// raw gets returned as an json array then converted into string using .Text() function
	if err == redis.Nil {
		return domain.Item{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.Item{}, fmt.Errorf("JSONStore.GetItem: %w", err)
	}
	// json.Unmarshal takes only byte input, so we convert raw to bytes using []byte(raw)
	var items []domain.Item
	if err := json.Unmarshal([]byte(raw), &items); err != nil {
		return domain.Item{}, fmt.Errorf("JSONStore.GetItem: unmarshal: %w", err)
	}
	if len(items) == 0 {
		return domain.Item{}, domain.ErrNotFound
	}
	return items[0], nil
}
func (s *JSONStore) AppendTag(ctx context.Context, id, tag string) error {
	tagJSON, _ := json.Marshal(tag)
	if err := s.client.Do(ctx, "JSON.ARRAPPEND",
		// just adds the given tagJSON into the end of the array (tag)
		itemKey(id), "$.tags", string(tagJSON)).Err(); err != nil {
		return fmt.Errorf("JSONStore.AppendTag: %w", err)
	}
	return nil
}
func (s *JSONStore) IncrScore(ctx context.Context, id string, delta float64) (float64, error) {
	res, err := s.client.Do(ctx, "JSON.NUMINCRBY", itemKey(id), "$.score", delta).Text()
	if err != nil {
		return 0, fmt.Errorf("JSONStore.IncrScore: %w", err)
	}
	var vals []float64
	if err := json.Unmarshal([]byte(res), &vals); err != nil || len(vals) == 0 {
		return 0, fmt.Errorf("JSONStore.IncrScore: parse result: %w", err)
	}
	return vals[0], nil
}
func (s *JSONStore) DeleteItem(ctx context.Context, id string) error {
	if err := s.client.Do(ctx, "JSON.DEL", itemKey(id), "$").Err(); err != nil {
		return fmt.Errorf("JSONStore.DeleteItem: %w", err)
	}
	return nil
}

/*
JSON.SET item-key $ JSONObject/JSONArray -> adds/updates the item-key with the JSONObject/JSONArray
JSON.GET item-key $ -> gets the entire following item-key
JSON.ARRAPPEND item-key $.tags tag -> adds the tag at the end of the JSONArray
JSON.INCRSCORE item-key $.score delta -> adds delta to the score field
JSON.DEL item-key $ -> deleted the complete item-key
*/
