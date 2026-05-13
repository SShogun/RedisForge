package handlers

import (
	"net/http"

	"github.com/SShogun/redisforge/internal/domain"
	"github.com/SShogun/redisforge/internal/repo"
)

func HandleListItems(items repo.ItemRepo) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		offset, err := parseQueryInt(r, "offset", 0)
		if err != nil || offset < 0 {
			writeError(w, r, domain.ErrInvalidInput)
			return
		}

		limit, err := parseQueryInt(r, "limit", 20)
		if err != nil || limit < 1 {
			writeError(w, r, domain.ErrInvalidInput)
			return
		}
		if limit > 100 {
			limit = 100
		}

		itemsList, err := items.List(r.Context(), offset, limit)
		if err != nil {
			writeError(w, r, err)
			return
		}

		writeJSON(w, http.StatusOK, envelope{
			"items":  itemsList,
			"offset": offset,
			"limit":  limit,
			"count":  len(itemsList),
		})
	}
}
