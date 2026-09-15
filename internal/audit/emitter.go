package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/SShogun/redisforge/internal/domain"
	"github.com/SShogun/redisforge/internal/observability"
)

const (
	StreamName  = "audit-events"
	emitTimeout = 5 * time.Second
)

type StreamAppender interface {
	Append(context.Context, string, map[string]interface{}) (string, error)
}

// Emitter owns producer-side audit serialization, timeout, metrics, and failure logging.
// HTTP handlers intentionally use EmitAsync so item writes are not rolled back when Redis
// is unavailable. Once an event is appended, the audit worker provides at-least-once recovery.
type Emitter struct {
	stream  StreamAppender
	logger  *slog.Logger
	timeout time.Duration
}

func NewEmitter(stream StreamAppender, logger *slog.Logger) *Emitter {
	if logger == nil {
		logger = slog.Default()
	}
	return &Emitter{stream: stream, logger: logger, timeout: emitTimeout}
}

// Emit appends one audit event synchronously and returns any serialization or Redis error.
func (e *Emitter) Emit(ctx context.Context, event domain.AuditEvent) (err error) {
	start := time.Now()
	defer func() {
		observability.RecordAuditEmit(start, event.Action, err)
	}()

	if e == nil || e.stream == nil {
		return fmt.Errorf("audit.Emitter.Emit: stream is nil")
	}

	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("audit.Emitter.Emit: marshal: %w", err)
	}

	if _, err := e.stream.Append(ctx, StreamName, map[string]interface{}{
		"event": string(payload),
	}); err != nil {
		return fmt.Errorf("audit.Emitter.Emit: append: %w", err)
	}
	return nil
}

// EmitAsync preserves the current best-effort producer contract: the HTTP response is not
// blocked on Redis. Failures are still visible through structured logs and Prometheus metrics.
func (e *Emitter) EmitAsync(event domain.AuditEvent) {
	if e == nil {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), e.timeout)
		defer cancel()

		if err := e.Emit(ctx, event); err != nil {
			e.logger.Error(
				"audit event emission failed",
				"event_id", event.EventID,
				"item_id", event.ItemID,
				"action", event.Action,
				"err", err,
			)
		}
	}()
}
