package handlers

import "net/http"

// CreateFlag handles POST /flags. Implemented by the CRUD-handler ticket.
func (s *Service) CreateFlag(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusNotImplemented, "CreateFlag not implemented")
}

// ListFlags handles GET /flags. Implemented by the CRUD-handler ticket.
func (s *Service) ListFlags(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusNotImplemented, "ListFlags not implemented")
}

// GetFlag handles GET /flags/{key}. Implemented by the CRUD-handler ticket.
func (s *Service) GetFlag(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusNotImplemented, "GetFlag not implemented")
}

// UpdateFlag handles PUT /flags/{key}. Implemented by the CRUD-handler ticket.
func (s *Service) UpdateFlag(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusNotImplemented, "UpdateFlag not implemented")
}

// DeleteFlag handles DELETE /flags/{key}. Implemented by the CRUD-handler ticket.
func (s *Service) DeleteFlag(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusNotImplemented, "DeleteFlag not implemented")
}
