package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"regexp"

	"featureflags/internal/store"
)

// maxBodyBytes is the maximum accepted request body size (1 MiB).
const maxBodyBytes = 1 << 20

var keyPattern = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

// createFlagRequest is the JSON body accepted by POST /flags. Pointer fields
// distinguish an omitted field from an explicitly zero value, so the defaults
// only apply to fields that were actually absent.
type createFlagRequest struct {
	Key            string  `json:"key"`
	Enabled        *bool   `json:"enabled"`
	Description    *string `json:"description"`
	RolloutPercent *int    `json:"rollout_percent"`
}

// updateFlagRequest is the JSON body accepted by PUT /flags/{key}. Pointer
// fields keep fields that are absent from the body unchanged.
type updateFlagRequest struct {
	Enabled        *bool   `json:"enabled"`
	Description    *string `json:"description"`
	RolloutPercent *int    `json:"rollout_percent"`
}

// decodeJSON reads and decodes the request body, enforcing the 1 MiB limit
// before the body is fully read. The returned error is io.EOF for an empty
// body, *http.MaxBytesError when the limit is exceeded, or the JSON decoding
// error otherwise.
func decodeJSON(w http.ResponseWriter, r *http.Request, v interface{}) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	return json.NewDecoder(r.Body).Decode(v)
}

// writeDecodeError maps a decoding error from decodeJSON to the appropriate
// HTTP response: 413 for a body that exceeds the limit, 400 otherwise.
func writeDecodeError(w http.ResponseWriter, err error) {
	var maxErr *http.MaxBytesError
	if errors.As(err, &maxErr) {
		writeError(w, http.StatusRequestEntityTooLarge, "request body too large")
		return
	}
	writeError(w, http.StatusBadRequest, "invalid JSON body")
}

// CreateFlag handles POST /flags.
func (s *Service) CreateFlag(w http.ResponseWriter, r *http.Request) {
	var req createFlagRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeDecodeError(w, err)
		return
	}

	if req.Key == "" {
		writeError(w, http.StatusBadRequest, "key is required")
		return
	}
	if !keyPattern.MatchString(req.Key) {
		writeError(w, http.StatusBadRequest, "invalid key: allowed characters are [a-zA-Z0-9_-]")
		return
	}

	flag := store.Flag{
		Key:            req.Key,
		Enabled:        false,
		Description:    "",
		RolloutPercent: 100,
	}
	if req.Enabled != nil {
		flag.Enabled = *req.Enabled
	}
	if req.Description != nil {
		flag.Description = *req.Description
	}
	if req.RolloutPercent != nil {
		flag.RolloutPercent = *req.RolloutPercent
	}
	if flag.RolloutPercent < 0 || flag.RolloutPercent > 100 {
		writeError(w, http.StatusBadRequest, "rollout_percent must be between 0 and 100")
		return
	}

	if err := s.store.Create(flag); err != nil {
		if errors.Is(err, store.ErrExists) {
			writeError(w, http.StatusConflict, "flag already exists")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeJSON(w, http.StatusCreated, flag)
}

// ListFlags handles GET /flags.
func (s *Service) ListFlags(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.store.List())
}

// GetFlag handles GET /flags/{key}.
func (s *Service) GetFlag(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	flag, ok := s.store.Get(key)
	if !ok {
		writeError(w, http.StatusNotFound, "flag not found")
		return
	}
	writeJSON(w, http.StatusOK, flag)
}

// UpdateFlag handles PUT /flags/{key}.
func (s *Service) UpdateFlag(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	existing, ok := s.store.Get(key)
	if !ok {
		writeError(w, http.StatusNotFound, "flag not found")
		return
	}

	var req updateFlagRequest
	if err := decodeJSON(w, r, &req); err != nil {
		if err == io.EOF {
			writeJSON(w, http.StatusOK, existing)
			return
		}
		writeDecodeError(w, err)
		return
	}

	if req.Enabled != nil {
		existing.Enabled = *req.Enabled
	}
	if req.Description != nil {
		existing.Description = *req.Description
	}
	if req.RolloutPercent != nil {
		if *req.RolloutPercent < 0 || *req.RolloutPercent > 100 {
			writeError(w, http.StatusBadRequest, "rollout_percent must be between 0 and 100")
			return
		}
		existing.RolloutPercent = *req.RolloutPercent
	}

	updated, _ := s.store.Update(key, existing)
	writeJSON(w, http.StatusOK, updated)
}

// DeleteFlag handles DELETE /flags/{key}.
func (s *Service) DeleteFlag(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	if !s.store.Delete(key) {
		writeError(w, http.StatusNotFound, "flag not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
