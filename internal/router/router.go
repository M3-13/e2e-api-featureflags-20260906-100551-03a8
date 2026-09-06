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

// New returns a router that maps the feature-flag service routes onto s.
// Unknown paths answer 404 and known paths with an unsupported method answer
// 405, both as JSON error objects. Each route is wired directly to its handler
// method on s, so the ServeMux calls the handler directly and r.PathValue("key")
// resolves the wildcard {key} segment reliably.
func New(s *handlers.Service) http.Handler {
	mux := http.NewServeMux()
	handle := func(fn http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if t, ok := r.Context().Value(trackerKey{}).(*tracker); ok {
				t.handled = true
			}
			fn(w, r)
		}
	}
	mux.Handle("GET /healthz", handle(s.Health))
	mux.Handle("GET /flags", handle(s.ListFlags))
	mux.Handle("POST /flags", handle(s.CreateFlag))
	mux.Handle("GET /flags/{key}", handle(s.GetFlag))
	mux.Handle("PUT /flags/{key}", handle(s.UpdateFlag))
	mux.Handle("DELETE /flags/{key}", handle(s.DeleteFlag))
	mux.Handle("GET /flags/{key}/evaluate", handle(s.Evaluate))

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
