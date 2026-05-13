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
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

func HandleUpdateItem(items repo.ItemRepo, stream *redisx.StreamClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimSpace(chi.URLParam(r, "id"))
		if id == "" {
			writeError(w, r, domain.ErrInvalidInput)
			return
		}

		var input struct {
			Name     *string   `json:"name"`
			Category *string   `json:"category"`
			Score    *float64  `json:"score"`
			Tags     *[]string `json:"tags"`
		}
		if err := readJSON(r, &input); err != nil {
			writeError(w, r, domain.ErrInvalidInput)
			return
		}

		existing, err := items.GetByID(r.Context(), id)
		if err != nil {
			writeError(w, r, err)
			return
		}

		updated := existing
		changed := false

		if input.Name != nil {
			updated.Name = strings.TrimSpace(*input.Name)
			changed = true
		}
		if input.Category != nil {
			updated.Category = strings.TrimSpace(*input.Category)
			changed = true
		}
		if input.Score != nil {
			updated.Score = *input.Score
			changed = true
		}
		if input.Tags != nil {
			updated.Tags = append([]string(nil), (*input.Tags)...)
			changed = true
		}

		if !changed {
			writeError(w, r, domain.ErrInvalidInput)
			return
		}
		if strings.TrimSpace(updated.Name) == "" || strings.TrimSpace(updated.Category) == "" {
			writeError(w, r, domain.ErrInvalidInput)
			return
		}

		updated.ID = existing.ID
		updated.Version = existing.Version
		updated.CreatedAt = existing.CreatedAt
		updated.UpdatedAt = existing.UpdatedAt

		updatedItem, err := items.Update(r.Context(), updated)
		if err != nil {
			writeError(w, r, err)
			return
		}

		go func(item domain.Item) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			event := domain.AuditEvent{
				EventID:   uuid.New().String(),
				ItemID:    item.ID,
				Action:    "updated",
				Timestamp: time.Now().UTC(),
			}
			eventJSON, _ := json.Marshal(event)
			_, _ = stream.Append(ctx, "audit-events", map[string]interface{}{
				"event": string(eventJSON),
			})
		}(updatedItem)

		writeJSON(w, http.StatusOK, envelope{"item": updatedItem})
	}
}
