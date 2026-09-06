package router

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
)

type trackerKey struct{}

type tracker struct {
	handled bool
}

type recorder struct {
	header http.Header
	status int
	body   bytes.Buffer
}

func (r *recorder) Header() http.Header { return r.header }
func (r *recorder) WriteHeader(code int) {
	if r.status == 0 {
		r.status = code
	}
}
func (r *recorder) Write(b []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK
	}
	return r.body.Write(b)
}

// New returns a router that maps the feature-flag service routes onto h.
// Unknown paths answer 404 and known paths with an unsupported method answer
// 405, both as JSON error objects. Wildcard {key} segments are made available
// to h via r.PathValue("key").
func New(h http.Handler) http.Handler {
	mux := http.NewServeMux()
	wrap := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if t, ok := r.Context().Value(trackerKey{}).(*tracker); ok {
			t.handled = true
		}
		h.ServeHTTP(w, r)
	})
	mux.Handle("GET /healthz", wrap)
	mux.Handle("GET /flags", wrap)
	mux.Handle("POST /flags", wrap)
	mux.Handle("GET /flags/{key}", wrap)
	mux.Handle("PUT /flags/{key}", wrap)
	mux.Handle("DELETE /flags/{key}", wrap)
	mux.Handle("GET /flags/{key}/evaluate", wrap)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t := &tracker{}
		ctx := context.WithValue(r.Context(), trackerKey{}, t)
		rec := &recorder{header: make(http.Header)}
		mux.ServeHTTP(rec, r.WithContext(ctx))

		if t.handled {
			for k, vs := range rec.header {
				for _, v := range vs {
					w.Header().Add(k, v)
				}
			}
			status := rec.status
			if status == 0 {
				status = http.StatusOK
			}
			w.WriteHeader(status)
			_, _ = w.Write(rec.body.Bytes())
			return
		}

		status := rec.status
		if status == 0 {
			status = http.StatusNotFound
		}
		msg := "not found"
		if status == http.StatusMethodNotAllowed {
			msg = "method not allowed"
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
	})
}
