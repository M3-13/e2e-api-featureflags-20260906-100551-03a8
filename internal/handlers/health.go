package handlers

import "net/http"

// Health responds with 200 {"status":"ok"}.
func (s *Service) Health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
