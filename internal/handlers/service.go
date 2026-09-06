package handlers

import (
	"net/http"
	"strings"

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

// ServeHTTP makes *Service implement http.Handler. It dispatches solely on the
// request method and URL path — never on r.Pattern, which is not reliably set
// when the router serves the request through a recorder with a derived
// context. The {key} handlers keep reading r.PathValue("key"), which the
// surrounding http.ServeMux sets when it matches a wildcard pattern.
func (s *Service) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	method := r.Method
	path := r.URL.Path

	if method == http.MethodGet && path == "/healthz" {
		s.Health(w, r)
		return
	}
	if method == http.MethodGet && path == "/flags" {
		s.ListFlags(w, r)
		return
	}
	if method == http.MethodPost && path == "/flags" {
		s.CreateFlag(w, r)
		return
	}

	rest, ok := strings.CutPrefix(path, "/flags")
	if !ok {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	rest = strings.TrimPrefix(rest, "/")

	if method == http.MethodGet && strings.HasSuffix(rest, "/evaluate") {
		s.Evaluate(w, r)
		return
	}

	switch method {
	case http.MethodGet:
		s.GetFlag(w, r)
	case http.MethodPut:
		s.UpdateFlag(w, r)
	case http.MethodDelete:
		s.DeleteFlag(w, r)
	default:
		writeError(w, http.StatusNotFound, "not found")
	}
}
