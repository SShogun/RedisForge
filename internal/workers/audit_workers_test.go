package workers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/SShogun/redisforge/internal/domain"
	"github.com/SShogun/redisforge/internal/redisx"
	"github.com/redis/go-redis/v9"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

func TestAuditWorkerStreamRecovery(t *testing.T) {
	addr := startRedisForWorkerTest(t)
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	t.Run("replays entries that existed before group creation", func(t *testing.T) {
		client := redis.NewClient(&redis.Options{Addr: addr})
		defer client.Close()
		if err := client.FlushDB(ctx).Err(); err != nil {
			t.Fatalf("flush redis: %v", err)
		}

		stream := redisx.NewStreamClient(client)
		if _, err := stream.Append(ctx, auditStream, auditFields(t, "before-group")); err != nil {
			t.Fatalf("append existing entry: %v", err)
		}
		if err := stream.EnsureGroup(ctx, auditStream, auditGroup); err != nil {
			t.Fatalf("ensure group: %v", err)
		}

		msgs, err := stream.ReadGroup(ctx, auditStream, auditGroup, "bootstrap-consumer", 10, 100*time.Millisecond)
		if err != nil {
			t.Fatalf("read group: %v", err)
		}
		if len(msgs) != 1 {
			t.Fatalf("expected pre-existing entry to be replayed, got %d messages", len(msgs))
		}
	})

	t.Run("paginates past fresh pending entries to recover stale tail", func(t *testing.T) {
		client := redis.NewClient(&redis.Options{Addr: addr})
		defer client.Close()
		if err := client.FlushDB(ctx).Err(); err != nil {
			t.Fatalf("flush redis: %v", err)
		}

		stream := redisx.NewStreamClient(client)
		if err := stream.EnsureGroup(ctx, auditStream, auditGroup); err != nil {
			t.Fatalf("ensure group: %v", err)
		}

		const totalPending = 65
		for i := 0; i < totalPending; i++ {
			if _, err := stream.Append(ctx, auditStream, auditFields(t, fmt.Sprintf("event-%02d", i))); err != nil {
				t.Fatalf("append event %d: %v", i, err)
			}
		}

		pending, err := stream.ReadGroup(ctx, auditStream, auditGroup, "dead-consumer", totalPending, 100*time.Millisecond)
		if err != nil {
			t.Fatalf("seed pending entries: %v", err)
		}
		if len(pending) != totalPending {
			t.Fatalf("expected %d pending entries, got %d", totalPending, len(pending))
		}

		// XAUTOCLAIM scans at most COUNT*10 PEL entries per call. Keep the first
		// 60 entries fresh and make only the final five stale. With batchSize=5,
		// recovery must advance the returned cursor to ever reach the stale tail.
		for _, msg := range pending[60:] {
			if err := client.Do(
				ctx,
				"XCLAIM",
				auditStream,
				auditGroup,
				"dead-consumer",
				0,
				msg.ID(),
				"IDLE",
				60_000,
			).Err(); err != nil {
				t.Fatalf("mark %s stale: %v", msg.ID(), err)
			}
		}

		worker := NewAuditWorker(stream, logger, "recovery-consumer")
		worker.batchSize = 5

		reclaimed, err := worker.reclaimPending(ctx)
		if err != nil {
			t.Fatalf("reclaim pending: %v", err)
		}
		if reclaimed != 5 {
			t.Fatalf("expected 5 stale entries to be reclaimed, got %d", reclaimed)
		}

		remaining, err := stream.GetPendingCount(ctx, auditStream, auditGroup)
		if err != nil {
			t.Fatalf("pending count: %v", err)
		}
		if remaining != 60 {
			t.Fatalf("expected 60 fresh pending entries to remain, got %d", remaining)
		}
	})

	t.Run("returns ACK failures instead of reporting success", func(t *testing.T) {
		client := redis.NewClient(&redis.Options{Addr: addr})
		if err := client.FlushDB(ctx).Err(); err != nil {
			t.Fatalf("flush redis: %v", err)
		}

		stream := redisx.NewStreamClient(client)
		if err := stream.EnsureGroup(ctx, auditStream, auditGroup); err != nil {
			t.Fatalf("ensure group: %v", err)
		}
		if _, err := stream.Append(ctx, auditStream, auditFields(t, "ack-failure")); err != nil {
			t.Fatalf("append event: %v", err)
		}
		msgs, err := stream.ReadGroup(ctx, auditStream, auditGroup, "ack-consumer", 1, 100*time.Millisecond)
		if err != nil {
			t.Fatalf("read event: %v", err)
		}
		if len(msgs) != 1 {
			t.Fatalf("expected one message, got %d", len(msgs))
		}

		if err := client.Close(); err != nil {
			t.Fatalf("close redis client: %v", err)
		}

		worker := NewAuditWorker(stream, logger, "ack-consumer")
		if err := worker.process(ctx, msgs[0]); err == nil {
			t.Fatal("expected ACK failure to be returned")
		}
	})
}

func auditFields(t *testing.T, id string) map[string]interface{} {
	t.Helper()
	eventJSON, err := json.Marshal(domain.AuditEvent{
		EventID:   id,
		ItemID:    "item-1",
		Action:    "updated",
		Timestamp: time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("marshal audit event: %v", err)
	}
	return map[string]interface{}{"event": string(eventJSON)}
}

func startRedisForWorkerTest(t *testing.T) string {
	t.Helper()
	ctx := context.Background()
	req := testcontainers.ContainerRequest{
		Image:        "redis:7.4.11-alpine",
		ExposedPorts: []string{"6379/tcp"},
		WaitingFor:   wait.ForLog("Ready to accept connections").WithStartupTimeout(30 * time.Second),
	}
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Fatalf("start redis container: %v", err)
	}
	t.Cleanup(func() {
		if err := container.Terminate(context.Background()); err != nil {
			t.Logf("terminate redis container: %v", err)
		}
	})

	host, err := container.Host(ctx)
	if err != nil {
		t.Fatalf("redis container host: %v", err)
	}
	port, err := container.MappedPort(ctx, "6379/tcp")
	if err != nil {
		t.Fatalf("redis container port: %v", err)
	}
	return host + ":" + port.Port()
}
