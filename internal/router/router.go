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

// maxResponseBytes bounds the response body a single request may buffer before
// it is sent to the client. Responses beyond this limit are aborted with 413
// instead of being buffered without bound.
const maxResponseBytes = 4 << 20 // 4 MiB

type recorder struct {
	header   http.Header
	status   int
	body     bytes.Buffer
	tooLarge bool
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
	if r.tooLarge {
		return len(b), nil
	}
	if r.body.Len()+len(b) > maxResponseBytes {
		r.tooLarge = true
		return len(b), nil
	}
	return r.body.Write(b)
}

// New returns a router that maps the feature-flag service routes onto h.
// Unknown paths answer 404 and known paths with an unsupported method answer
// 405, both as JSON error objects. Wildcard {key} segments are made available
// to the handlers via r.PathValue("key"). Every pattern is wired to a common
// wrap handler that marks the request as handled and delegates to h.ServeHTTP,
// so routing works with *handlers.Service (which dispatches on method+path) as
// well as with any other http.Handler, with no type assertion.
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

		if rec.tooLarge {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(http.StatusRequestEntityTooLarge)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "response too large"})
			return
		}

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
