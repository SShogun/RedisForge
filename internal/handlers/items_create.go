package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/SShogun/redisforge/internal/domain"
	"github.com/SShogun/redisforge/internal/redisx"
	"github.com/SShogun/redisforge/internal/repo"
	"github.com/google/uuid"
)

// HandleCreateItem creates an Item and emits an audit event to the stream.
// Idempotency pre-check: BloomFilter → if "might exist", skip to duplicate check.
func HandleCreateItem(
	items repo.ItemRepo,
	stream *redisx.StreamClient,
	bloom *redisx.BloomFilter,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Name           string   `json:"name"`
			Category       string   `json:"category"`
			Score          float64  `json:"score"`
			Tags           []string `json:"tags"`
			IdempotencyKey string   `json:"idempotency_key"`
		}
		if err := readJSON(r, &input); err != nil {
			writeError(w, r, domain.ErrInvalidInput)
			return
		}
		if strings.TrimSpace(input.Name) == "" || strings.TrimSpace(input.Category) == "" {
			writeError(w, r, domain.ErrInvalidInput)
			return
		}

		// Idempotency pre-check with Bloom filter
		if input.IdempotencyKey != "" {
			exists, err := bloom.Exists(r.Context(), input.IdempotencyKey)
			if err == nil && exists {
				// Bloom says "might exist"
				// treat as duplicate to demonstrate the flow.
				writeError(w, r, domain.ErrDuplicate)
				return
			}
		}

		item := domain.Item{
			ID:       uuid.New().String(),
			Name:     input.Name,
			Category: input.Category,
			Score:    input.Score,
			Tags:     input.Tags,
		}

		created, err := items.Create(r.Context(), item)
		if err != nil {
			writeError(w, r, err)
			return
		}

		// Record Bloom key AFTER successful creation
		if input.IdempotencyKey != "" {
			_ = bloom.Add(r.Context(), input.IdempotencyKey)
		}

		// Emit audit event to stream (async, non-blocking path)
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			event := domain.AuditEvent{
				EventID:   uuid.New().String(),
				ItemID:    created.ID,
				Action:    "created",
				Timestamp: time.Now().UTC(),
			}
			eventJSON, _ := json.Marshal(event)
			_, _ = stream.Append(ctx, "audit-events", map[string]interface{}{
				"event": string(eventJSON),
			})
		}()

		writeJSON(w, http.StatusCreated, envelope{"item": created})
	}
}
