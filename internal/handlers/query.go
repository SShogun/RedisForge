package handlers

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/SShogun/redisforge/internal/domain"
)

func parseQueryInt(r *http.Request, key string, fallback int) (int, error) {
	value := r.URL.Query().Get(key)
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("%w: invalid %s", domain.ErrInvalidInput, key)
	}
	return parsed, nil
}
