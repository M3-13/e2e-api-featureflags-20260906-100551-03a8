package handlers

import (
	"net/http"

	"featureflags/internal/store"
)

// Service holds the handler dependencies.
type Service struct {
	store *store.Store
}

// New returns a Service backed by the given store.
func New(s *store.Store) *Service {
	return &Service{store: s}
}

// ServeHTTP makes *Service implement http.Handler, as required by the router
// contract (router.New takes an http.Handler). The router registers the
// concrete handler methods directly, so request routing never depends on this
// method; it exists only so that *Service can be passed where an http.Handler
// is expected.
func (s *Service) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Pattern {
	case "GET /healthz":
		s.Health(w, r)
	case "GET /flags":
		s.ListFlags(w, r)
	case "POST /flags":
		s.CreateFlag(w, r)
	case "GET /flags/{key}":
		s.GetFlag(w, r)
	case "PUT /flags/{key}":
		s.UpdateFlag(w, r)
	case "DELETE /flags/{key}":
		s.DeleteFlag(w, r)
	case "GET /flags/{key}/evaluate":
		s.Evaluate(w, r)
	default:
		writeError(w, http.StatusNotFound, "not found")
	}
}
