package testutil

import (
	"context"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

const RedisStackImage = "redis/redis-stack-server:7.4.0-v8"

// StartRedisStack starts the pinned Redis Stack image used by integration tests
// and benchmarks. The client deliberately uses RESP2 to match redisx.Open across
// single-node, Sentinel, and Cluster modes.
func StartRedisStack(tb testing.TB) *redis.Client {
	tb.Helper()

	startupCtx, cancelStartup := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancelStartup()

	req := testcontainers.ContainerRequest{
		Image:        RedisStackImage,
		ExposedPorts: []string{"6379/tcp"},
		WaitingFor:   wait.ForLog("Ready to accept connections").WithStartupTimeout(30 * time.Second),
	}
	container, err := testcontainers.GenericContainer(startupCtx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		tb.Fatalf("start redis-stack: %v", err)
	}

	host, err := container.Host(startupCtx)
	if err != nil {
		_ = container.Terminate(context.Background())
		tb.Fatalf("redis-stack host: %v", err)
	}
	port, err := container.MappedPort(startupCtx, "6379/tcp")
	if err != nil {
		_ = container.Terminate(context.Background())
		tb.Fatalf("redis-stack port: %v", err)
	}

	client := redis.NewClient(&redis.Options{
		Addr:     host + ":" + port.Port(),
		Protocol: 2,
	})
	pingCtx, cancelPing := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelPing()
	if err := client.Ping(pingCtx).Err(); err != nil {
		_ = client.Close()
		_ = container.Terminate(context.Background())
		tb.Fatalf("ping redis-stack: %v", err)
	}

	tb.Cleanup(func() {
		if err := client.Close(); err != nil {
			tb.Logf("close redis-stack client: %v", err)
		}
		cleanupCtx, cancelCleanup := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancelCleanup()
		if err := container.Terminate(cleanupCtx); err != nil {
			tb.Logf("terminate redis-stack: %v", err)
		}
	})

	return client
}
