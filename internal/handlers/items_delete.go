package handlers

import (
	"net/http"
	"strings"
	"time"

	"github.com/SShogun/redisforge/internal/audit"
	"github.com/SShogun/redisforge/internal/domain"
	"github.com/SShogun/redisforge/internal/repo"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

func HandleDeleteItem(items repo.ItemRepo, auditEmitter *audit.Emitter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimSpace(chi.URLParam(r, "id"))
		if id == "" {
			writeError(w, r, domain.ErrInvalidInput)
			return
		}

		if err := items.Delete(r.Context(), id); err != nil {
			writeError(w, r, err)
			return
		}

		auditEmitter.EmitAsync(domain.AuditEvent{
			EventID:   uuid.New().String(),
			ItemID:    id,
			Action:    "deleted",
			Timestamp: time.Now().UTC(),
		})

		w.WriteHeader(http.StatusNoContent)
	}
}
