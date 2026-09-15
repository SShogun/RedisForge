package workers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/SShogun/redisforge/internal/domain"
	"github.com/SShogun/redisforge/internal/observability"
	"github.com/SShogun/redisforge/internal/redisx"
)

const (
	auditStream          = "audit-events"
	auditGroup           = "audit-processors"
	defaultBatchSize     = 10
	defaultBlockDuration = 2 * time.Second
	defaultClaimInterval = 5 * time.Second
)

// AuditWorker processes audit events from the Redis Stream.
type AuditWorker struct {
	stream        *redisx.StreamClient
	logger        *slog.Logger
	consumerName  string
	batchSize     int64
	blockDuration time.Duration
	claimInterval time.Duration
	wg            sync.WaitGroup
}

func NewAuditWorker(stream *redisx.StreamClient, logger *slog.Logger, consumerName string) *AuditWorker {
	return &AuditWorker{
		stream:        stream,
		logger:        logger,
		consumerName:  consumerName,
		batchSize:     defaultBatchSize,
		blockDuration: defaultBlockDuration,
		claimInterval: defaultClaimInterval,
	}
}

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

		msgs, err := w.stream.ReadGroup(ctx, auditStream, auditGroup, w.consumerName, w.batchSize, w.blockDuration)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			w.logger.Error("audit_worker: ReadGroup error", "err", err)
			if !waitForRetry(ctx, time.Second) {
				return
			}
			continue
		}

		for _, msg := range msgs {
			_ = w.process(ctx, msg)
		}
	}
}

func (w *AuditWorker) claimLoop(ctx context.Context) {
	defer w.wg.Done()
	ticker := time.NewTicker(w.claimInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			reclaimed, err := w.reclaimPending(ctx)
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				w.logger.Warn("audit_worker: reclaim pending failed", "err", err)
			} else if reclaimed > 0 {
				w.logger.Info("audit_worker: reclaimed stale messages", "count", reclaimed)
			}

			pending, err := w.stream.GetPendingCount(ctx, auditStream, auditGroup)
			if err == nil {
				observability.SetStreamPendingCount(auditStream, auditGroup, float64(pending))
			} else if ctx.Err() == nil {
				w.logger.Warn("audit_worker: failed to get pending count", "err", err)
			}
		}
	}
}

// reclaimPending walks the entire XAUTOCLAIM cursor so stale messages cannot
// be stranded behind newer pending entries that are not eligible for claiming.
func (w *AuditWorker) reclaimPending(ctx context.Context) (int, error) {
	cursor := "0-0"
	reclaimed := 0

	for {
		msgs, nextCursor, err := w.stream.ClaimStale(
			ctx,
			auditStream,
			auditGroup,
			w.consumerName,
			cursor,
			w.batchSize,
		)
		if err != nil {
			return reclaimed, err
		}

		for _, msg := range msgs {
			w.logger.Info("audit_worker: reclaimed stale message", "id", msg.ID())
			if err := w.process(ctx, msg); err != nil && ctx.Err() != nil {
				return reclaimed, ctx.Err()
			}
			reclaimed++
		}

		if nextCursor == "0-0" {
			return reclaimed, nil
		}
		if nextCursor == cursor {
			return reclaimed, fmt.Errorf("audit_worker: XAUTOCLAIM cursor did not advance from %s", cursor)
		}
		cursor = nextCursor
	}
}

func (w *AuditWorker) process(ctx context.Context, msg redisx.Message) (resultErr error) {
	start := time.Now()
	action := "unknown"
	var metricErr error

	defer func() {
		if !errors.Is(metricErr, context.Canceled) {
			observability.RecordStreamProcessing(start, metricErr, action)
		}
	}()

	if err := ctx.Err(); err != nil {
		w.logger.Info("audit_worker: context cancelled, skipping process", "id", msg.ID())
		metricErr = err
		return err
	}

	raw, ok := msg.Values()["event"].(string)
	if !ok {
		metricErr = fmt.Errorf("audit_worker: missing event field")
		w.logger.Warn("audit_worker: missing event field", "id", msg.ID())
		if ackErr := msg.Ack(ctx); ackErr != nil {
			w.logger.Error("audit_worker: ACK failed for malformed message", "id", msg.ID(), "err", ackErr)
			metricErr = errors.Join(metricErr, ackErr)
			return ackErr
		}
		return nil
	}

	var event domain.AuditEvent
	if err := json.Unmarshal([]byte(raw), &event); err != nil {
		metricErr = fmt.Errorf("audit_worker: unmarshal event: %w", err)
		w.logger.Error("audit_worker: unmarshal failed", "id", msg.ID(), "err", err)
		if ackErr := msg.Ack(ctx); ackErr != nil {
			w.logger.Error("audit_worker: ACK failed for malformed message", "id", msg.ID(), "err", ackErr)
			metricErr = errors.Join(metricErr, ackErr)
			return ackErr
		}
		return nil
	}
	action = event.Action

	w.logger.Info("audit_worker: processed event",
		"event_id", event.EventID,
		"item_id", event.ItemID,
		"action", event.Action,
	)

	if err := msg.Ack(ctx); err != nil {
		resultErr = fmt.Errorf("audit_worker: ACK processed message %s: %w", msg.ID(), err)
		metricErr = resultErr
		w.logger.Error("audit_worker: ACK failed", "id", msg.ID(), "err", err)
		return resultErr
	}

	return nil
}

func (w *AuditWorker) Stop() {
	w.wg.Wait()
}

func waitForRetry(ctx context.Context, delay time.Duration) bool {
	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}
