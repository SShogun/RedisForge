package handlers

import (
	"net/http"
	"strings"

	"github.com/SShogun/redisforge/internal/domain"
	"github.com/SShogun/redisforge/internal/repo"
	"github.com/go-chi/chi/v5"
)

func HandleGetItem(items repo.ItemRepo) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimSpace(chi.URLParam(r, "id"))
		if id == "" {
			writeError(w, r, domain.ErrInvalidInput)
			return
		}

		item, err := items.GetByID(r.Context(), id)
		if err != nil {
			writeError(w, r, err)
			return
		}

		writeJSON(w, http.StatusOK, envelope{"item": item})
	}
}
