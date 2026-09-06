package handlers

import (
	"featureflags/internal/store"
)

// Service holds the handler dependencies for the individual route handlers.
type Service struct {
	store *store.Store
}

// New returns a Service backed by the given store.
func New(s *store.Store) *Service {
	return &Service{store: s}
}
