package handlers

import (
	"net/http"
	"strings"

	"github.com/SShogun/redisforge/internal/domain"
	"github.com/SShogun/redisforge/internal/repo"
	"github.com/go-chi/chi/v5"
)

func HandleDeleteItem(items repo.ItemRepo) http.HandlerFunc {
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

		w.WriteHeader(http.StatusNoContent)
	}
}
