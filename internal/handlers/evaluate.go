package handlers

import "net/http"

// Evaluate handles GET /flags/{key}/evaluate. Implemented by the
// evaluation-endpoint ticket.
func (s *Service) Evaluate(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusNotImplemented, "Evaluate not implemented")
}
