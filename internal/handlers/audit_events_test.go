package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/SShogun/redisforge/internal/audit"
	"github.com/SShogun/redisforge/internal/domain"
	"github.com/SShogun/redisforge/internal/handlers"
	"github.com/SShogun/redisforge/internal/repo"
	"github.com/go-chi/chi/v5"
)

type recordingAuditAppender struct {
	events chan domain.AuditEvent
}

func (r *recordingAuditAppender) Append(_ context.Context, stream string, fields map[string]interface{}) (string, error) {
	if stream != audit.StreamName {
		return "", nil
	}
	raw, _ := fields["event"].(string)
	var event domain.AuditEvent
	if err := json.Unmarshal([]byte(raw), &event); err == nil {
		r.events <- event
	}
	return "1-0", nil
}

func TestItemWriteHandlersEmitAuditEvents(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	t.Run("create", func(t *testing.T) {
		items := repo.NewMemoryItemRepo()
		appender := &recordingAuditAppender{events: make(chan domain.AuditEvent, 1)}
		emitter := audit.NewEmitter(appender, logger)
		handler := handlers.HandleCreateItem(items, emitter, nil)

		req := httptest.NewRequest(http.MethodPost, "/v1/items", bytes.NewBufferString(`{"name":"created","category":"tests"}`))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusCreated {
			t.Fatalf("expected %d, got %d", http.StatusCreated, rr.Code)
		}

		event := waitForAuditEvent(t, appender.events)
		if event.Action != "created" || event.ItemID == "" {
			t.Fatalf("unexpected create audit event: %#v", event)
		}
	})

	t.Run("update", func(t *testing.T) {
		items := repo.NewMemoryItemRepo()
		created, err := items.Create(context.Background(), domain.Item{ID: "item-update", Name: "before", Category: "tests"})
		if err != nil {
			t.Fatalf("seed item: %v", err)
		}
		if created.ID == "" {
			t.Fatal("seed item missing id")
		}

		appender := &recordingAuditAppender{events: make(chan domain.AuditEvent, 1)}
		emitter := audit.NewEmitter(appender, logger)
		router := chi.NewRouter()
		router.Put("/v1/items/{id}", handlers.HandleUpdateItem(items, emitter))

		req := httptest.NewRequest(http.MethodPut, "/v1/items/item-update", bytes.NewBufferString(`{"name":"after"}`))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected %d, got %d", http.StatusOK, rr.Code)
		}

		event := waitForAuditEvent(t, appender.events)
		if event.Action != "updated" || event.ItemID != "item-update" {
			t.Fatalf("unexpected update audit event: %#v", event)
		}
	})

	t.Run("delete", func(t *testing.T) {
		items := repo.NewMemoryItemRepo()
		if _, err := items.Create(context.Background(), domain.Item{ID: "item-delete", Name: "delete", Category: "tests"}); err != nil {
			t.Fatalf("seed item: %v", err)
		}

		appender := &recordingAuditAppender{events: make(chan domain.AuditEvent, 1)}
		emitter := audit.NewEmitter(appender, logger)
		router := chi.NewRouter()
		router.Delete("/v1/items/{id}", handlers.HandleDeleteItem(items, emitter))

		req := httptest.NewRequest(http.MethodDelete, "/v1/items/item-delete", nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		if rr.Code != http.StatusNoContent {
			t.Fatalf("expected %d, got %d", http.StatusNoContent, rr.Code)
		}

		event := waitForAuditEvent(t, appender.events)
		if event.Action != "deleted" || event.ItemID != "item-delete" {
			t.Fatalf("unexpected delete audit event: %#v", event)
		}
	})
}

func waitForAuditEvent(t *testing.T, events <-chan domain.AuditEvent) domain.AuditEvent {
	t.Helper()
	select {
	case event := <-events:
		return event
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for audit event")
		return domain.AuditEvent{}
	}
}
