package handlers

import (
	"net/http"

	"featureflags/internal/hash"
)

// Evaluate handles GET /flags/{key}/evaluate. It resolves the flag by the key
// from the path (404 if unknown), requires a non-empty "user" query parameter
// (400 otherwise), and answers {"active": bool} deterministically for that
// key/user pair.
func (s *Service) Evaluate(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")

	flag, ok := s.store.Get(key)
	if !ok {
		writeError(w, http.StatusNotFound, "flag not found")
		return
	}

	user := r.URL.Query().Get("user")
	if user == "" {
		writeError(w, http.StatusBadRequest, "user is required")
		return
	}

	active := hash.Evaluate(key, user, flag.RolloutPercent, flag.Enabled)
	writeJSON(w, http.StatusOK, map[string]bool{"active": active})
}
