package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/SShogun/redisforge/internal/handlers"
	"github.com/SShogun/redisforge/internal/redisx"
	"github.com/SShogun/redisforge/internal/repo"
	"github.com/redis/go-redis/v9"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

func startRedisStack(t *testing.T) string {
	t.Helper()
	ctx := context.Background()
	req := testcontainers.ContainerRequest{
		Image:        "redis/redis-stack:latest",
		ExposedPorts: []string{"6379/tcp"},
		WaitingFor:   wait.ForLog("Ready to accept connections").WithStartupTimeout(30 * time.Second),
	}
	c, err := testcontainers.GenericContainer(ctx,
		testcontainers.GenericContainerRequest{ContainerRequest: req, Started: true})
	if err != nil {
		t.Fatalf("start redis-stack: %v", err)
	}
	t.Cleanup(func() { c.Terminate(ctx) })
	host, _ := c.Host(ctx)
	port, _ := c.MappedPort(ctx, "6379")
	return host + ":" + port.Port()
}

func TestHandleCreateItem_Idempotency(t *testing.T) {
	addr := startRedisStack(t)
	client := redis.NewClient(&redis.Options{Addr: addr})
	defer client.Close()

	ctx := context.Background()
	bloom := redisx.NewBloomFilter(client, "bf:idempotency_test")
	// Initialize bloom filter
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
	body, _ := json.Marshal(payload)

	// First Request - Should Succeed (201 Created)
	req1, _ := http.NewRequest(http.MethodPost, "/v1/items", bytes.NewReader(body))
	req1.Header.Set("Content-Type", "application/json")
	rr1 := httptest.NewRecorder()

	handler.ServeHTTP(rr1, req1)

	if status := rr1.Code; status != http.StatusCreated {
		t.Errorf("first request: expected %v, got %v", http.StatusCreated, status)
	}

	// Second Request - Should Fail due to idempotency key (409 Conflict)
	req2, _ := http.NewRequest(http.MethodPost, "/v1/items", bytes.NewReader(body))
	req2.Header.Set("Content-Type", "application/json")
	rr2 := httptest.NewRecorder()

	handler.ServeHTTP(rr2, req2)

	if status := rr2.Code; status != http.StatusConflict {
		t.Errorf("second request: expected %v (Conflict), got %v", http.StatusConflict, status)
	}
}
