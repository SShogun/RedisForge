package redisx_test

import (
	"context"
	"testing"
	"time"

	"github.com/SShogun/redisforge/internal/domain"
	"github.com/SShogun/redisforge/internal/redisx"
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

func TestJSONStore_SetGet(t *testing.T) {
	addr := startRedisStack(t)
	client := redis.NewClient(&redis.Options{Addr: addr})
	store := redisx.NewJSONStore(client)
	ctx := context.Background()

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
	got, err := store.GetItem(ctx, "test-1")
	if err != nil {
		t.Fatalf("GetItem: %v", err)
	}
	if got.Name != "Widget" {
		t.Errorf("expected Widget, got %s", got.Name)
	}
}

func TestJSONStore_AppendTag(t *testing.T) {
	addr := startRedisStack(t)
	client := redis.NewClient(&redis.Options{Addr: addr})
	store := redisx.NewJSONStore(client)
	ctx := context.Background()

	item := domain.Item{ID: "tag-test", Tags: []string{"alpha"}}
	_ = store.SetItem(ctx, item)
	_ = store.AppendTag(ctx, "tag-test", "beta")

	got, _ := store.GetItem(ctx, "tag-test")
	if len(got.Tags) != 2 || got.Tags[1] != "beta" {
		t.Errorf("expected [alpha beta], got %v", got.Tags)
	}
}

func TestJSONStore_GetNotFound(t *testing.T) {
	addr := startRedisStack(t)
	client := redis.NewClient(&redis.Options{Addr: addr})
	store := redisx.NewJSONStore(client)

	_, err := store.GetItem(context.Background(), "does-not-exist")
	if err != domain.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

// lets understand this file
/*
	testcontainers is a go library that launches docker containers from your tests so you can run
	integration tests against readl services like Redis, Postgres, etc instead of using mocks

	we use a startRedisStack container
	waits for the server to be ready,
	uses the container host:port to create a real Redis client for tests.
*/
