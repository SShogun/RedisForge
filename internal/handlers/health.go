package handlers

import "net/http"

func HandleHealth() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, envelope{"status": "ok"})
	}
}
