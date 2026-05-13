package workers

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	"github.com/SShogun/redisforge/internal/domain"
	"github.com/SShogun/redisforge/internal/redisx"
)

const (
	auditStream   = "audit-events"
	auditGroup    = "audit-processors"
	batchSize     = 10
	blockDuration = 2 * time.Second
	claimInterval = 5 * time.Second
)

// AuditWorker processes audit events from the Redis Stream.
type AuditWorker struct {
	stream       *redisx.StreamClient
	logger       *slog.Logger
	consumerName string // unique per process
	wg           sync.WaitGroup
}

func NewAuditWorker(stream *redisx.StreamClient, logger *slog.Logger, consumerName string) *AuditWorker {
	return &AuditWorker{stream: stream, logger: logger, consumerName: consumerName}
}

// consume loop is
func (w *AuditWorker) Start(ctx context.Context) error {
	setupCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := w.stream.EnsureGroup(setupCtx, auditStream, auditGroup); err != nil {
		return err
	}

	w.wg.Add(2)
	go w.consumeLoop(ctx)
	go w.claimLoop(ctx)
	return nil
}

func (w *AuditWorker) consumeLoop(ctx context.Context) {
	defer w.wg.Done()
	for {
		select {
		case <-ctx.Done():
			w.logger.Info("audit_worker: context cancelled, stopping consume loop")
			return
		default:
		}

		msgs, err := w.stream.ReadGroup(ctx, auditStream, auditGroup, w.consumerName, batchSize, blockDuration)
		if err != nil {
			if ctx.Err() != nil {
				return // shutting down
			}
			w.logger.Error("audit_worker: ReadGroup error", "err", err)
			time.Sleep(1 * time.Second)
			continue
		}

		for _, msg := range msgs {
			w.process(ctx, msg)
		}
	}
}

func (w *AuditWorker) claimLoop(ctx context.Context) {
	defer w.wg.Done()
	ticker := time.NewTicker(claimInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			stale, _, err := w.stream.ClaimStale(ctx, auditStream, auditGroup,
				w.consumerName, batchSize)
			if err != nil {
				w.logger.Warn("audit_worker: ClaimStale error", "err", err)
				continue
			}
			for _, msg := range stale {
				w.logger.Info("audit_worker: reclaimed stale message", "id", msg.ID())
				w.process(ctx, msg)
			}
		}
	}
}

func (w *AuditWorker) process(ctx context.Context, msg redisx.Message) {
	raw, ok := msg.Values()["event"].(string)
	if !ok {
		w.logger.Warn("audit_worker: missing event field", "id", msg.ID())
		// need to ACK otherwise process will requeue forever
		_ = msg.Ack(ctx)
		return
	}

	var event domain.AuditEvent
	if err := json.Unmarshal([]byte(raw), &event); err != nil {
		w.logger.Error("audit_worker: unmarshal failed", "id", msg.ID(), "err", err)
		_ = msg.Ack(ctx) // dead-letter
		return
	}

	// just log
	w.logger.Info("audit_worker: processed event",
		"event_id", event.EventID,
		"item_id", event.ItemID,
		"action", event.Action,
	)

	if err := msg.Ack(ctx); err != nil {
		w.logger.Error("audit_worker: ACK failed", "id", msg.ID(), "err", err)
	}
}

func (w *AuditWorker) Stop() {
	w.wg.Wait()
}
