package audit

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/SShogun/redisforge/internal/domain"
)

type fakeAppender struct {
	stream string
	fields map[string]interface{}
	err    error
}

func (f *fakeAppender) Append(_ context.Context, stream string, fields map[string]interface{}) (string, error) {
	f.stream = stream
	f.fields = fields
	if f.err != nil {
		return "", f.err
	}
	return "1-0", nil
}

func TestEmitterEmit(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	t.Run("appends serialized event to audit stream", func(t *testing.T) {
		appender := &fakeAppender{}
		emitter := NewEmitter(appender, logger)
		event := domain.AuditEvent{EventID: "event-1", ItemID: "item-1", Action: "created"}

		if err := emitter.Emit(context.Background(), event); err != nil {
			t.Fatalf("Emit: %v", err)
		}
		if appender.stream != StreamName {
			t.Fatalf("expected stream %q, got %q", StreamName, appender.stream)
		}
		raw, ok := appender.fields["event"].(string)
		if !ok || raw == "" {
			t.Fatalf("expected serialized event field, got %#v", appender.fields["event"])
		}
	})

	t.Run("propagates append failure", func(t *testing.T) {
		want := errors.New("redis unavailable")
		emitter := NewEmitter(&fakeAppender{err: want}, logger)

		err := emitter.Emit(context.Background(), domain.AuditEvent{Action: "updated"})
		if !errors.Is(err, want) {
			t.Fatalf("expected wrapped append error, got %v", err)
		}
	})

	t.Run("rejects nil stream", func(t *testing.T) {
		emitter := NewEmitter(nil, logger)
		if err := emitter.Emit(context.Background(), domain.AuditEvent{Action: "deleted"}); err == nil {
			t.Fatal("expected nil stream error")
		}
	})
}
