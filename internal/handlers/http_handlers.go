package handlers

// http_helpers.go provides writeJSON, readJSON, and a standard error
// envelope. All handlers use these — never write JSON or read bodies directly.

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/SShogun/redisforge/internal/domain"
)

type envelope map[string]interface{}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func readJSON(r *http.Request, dst interface{}) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(dst)
}

func writeError(w http.ResponseWriter, r *http.Request, err error) {
	var status int
	var code string

	switch {
	case errors.Is(err, domain.ErrNotFound):
		status, code = http.StatusNotFound, "not_found"
	case errors.Is(err, domain.ErrConflict):
		status, code = http.StatusConflict, "conflict"
	case errors.Is(err, domain.ErrDuplicate):
		status, code = http.StatusConflict, "duplicate"
	case errors.Is(err, domain.ErrInvalidInput):
		status, code = http.StatusBadRequest, "invalid_input"
	default:
		status, code = http.StatusInternalServerError, "internal_error"
	}

	writeJSON(w, status, envelope{
		"error": envelope{"code": code, "message": err.Error()},
	})
}
