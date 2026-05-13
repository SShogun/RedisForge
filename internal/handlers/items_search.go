package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/SShogun/redisforge/internal/domain"
	"github.com/SShogun/redisforge/internal/redisx"
)

func HandleSearchItems(search *redisx.SearchStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		query := strings.TrimSpace(r.URL.Query().Get("q"))
		if query == "" {
			writeError(w, r, domain.ErrInvalidInput)
			return
		}

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

		results, total, err := search.Search(r.Context(), query, offset, limit)
		if err != nil {
			writeError(w, r, err)
			return
		}

		items := make([]domain.Item, 0, len(results))
		for _, result := range results {
			var item domain.Item
			if err := json.Unmarshal([]byte(result.RawJSON), &item); err != nil {
				var fallback []domain.Item
				if err := json.Unmarshal([]byte(result.RawJSON), &fallback); err != nil || len(fallback) == 0 {
					writeError(w, r, fmt.Errorf("search result %s: decode failed", result.ID))
					return
				}
				item = fallback[0]
			}
			if item.ID == "" {
				item.ID = strings.TrimSpace(result.ID)
			}
			items = append(items, item)
		}

		writeJSON(w, http.StatusOK, envelope{
			"query":  query,
			"offset": offset,
			"limit":  limit,
			"total":  total,
			"items":  items,
		})
	}
}
