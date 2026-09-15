package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/SShogun/redisforge/internal/handlers"
	"github.com/SShogun/redisforge/internal/redisx"
	"github.com/SShogun/redisforge/internal/repo"
	"github.com/SShogun/redisforge/internal/testutil"
)

func TestHandleCreateItem_Idempotency(t *testing.T) {
	client := testutil.StartRedisStack(t)

	ctx := context.Background()
	bloom := redisx.NewBloomFilter(client, "bf:idempotency_test")
	if err := bloom.Reserve(ctx, 0.001, 1000); err != nil {
		t.Fatalf("bloom.Reserve failed: %v", err)
	}

	stream := redisx.NewStreamClient(client)
	inMemoryRepo := repo.NewMemoryItemRepo()

	handler := handlers.HandleCreateItem(inMemoryRepo, stream, bloom)

	payload := map[string]interface{}{
		"name":            "Idempotency Test Widget",
		"category":        "tests",
		"score":           42.0,
		"tags":            []string{"test"},
		"idempotency_key": "unique-req-123",
	}
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	// First request succeeds and records the idempotency key.
	req1 := httptest.NewRequest(http.MethodPost, "/v1/items", bytes.NewReader(body))
	req1.Header.Set("Content-Type", "application/json")
	rr1 := httptest.NewRecorder()
	handler.ServeHTTP(rr1, req1)
	if rr1.Code != http.StatusCreated {
		t.Fatalf("first request: expected %d, got %d", http.StatusCreated, rr1.Code)
	}

	// Second request with the same key is rejected.
	req2 := httptest.NewRequest(http.MethodPost, "/v1/items", bytes.NewReader(body))
	req2.Header.Set("Content-Type", "application/json")
	rr2 := httptest.NewRecorder()
	handler.ServeHTTP(rr2, req2)
	if rr2.Code != http.StatusConflict {
		t.Fatalf("second request: expected %d, got %d", http.StatusConflict, rr2.Code)
	}
}
