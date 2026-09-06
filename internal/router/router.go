package router

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"

	"featureflags/internal/handlers"
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
// to the handlers via r.PathValue("key"). Every pattern is wired directly to
// its concrete handler method (s.Health, s.ListFlags, ...), so routing never
// depends on http.Request.Pattern.
func New(h http.Handler) http.Handler {
	mux := http.NewServeMux()

	route := func(fn func(w http.ResponseWriter, r *http.Request)) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if t, ok := r.Context().Value(trackerKey{}).(*tracker); ok {
				t.handled = true
			}
			fn(w, r)
		})
	}

	s := h.(*handlers.Service)

	mux.Handle("GET /healthz", route(s.Health))
	mux.Handle("GET /flags", route(s.ListFlags))
	mux.Handle("POST /flags", route(s.CreateFlag))
	mux.Handle("GET /flags/{key}", route(s.GetFlag))
	mux.Handle("PUT /flags/{key}", route(s.UpdateFlag))
	mux.Handle("DELETE /flags/{key}", route(s.DeleteFlag))
	mux.Handle("GET /flags/{key}/evaluate", route(s.Evaluate))

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
