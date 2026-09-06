package middleware

import (
	"log"
	"net/http"
	"strings"
	"time"
)

// responseWriter wraps http.ResponseWriter and records the status code so the
// Logging middleware can report it after the handler has finished.
type responseWriter struct {
	http.ResponseWriter
	status int
}

func (w *responseWriter) WriteHeader(code int) {
	if w.status == 0 {
		w.status = code
	}
	w.ResponseWriter.WriteHeader(code)
}

func (w *responseWriter) Write(b []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	return w.ResponseWriter.Write(b)
}

// sanitize removes control characters (newline, carriage return, tab, and
// other C0 control bytes) from a client-controlled value so it cannot forge
// additional log lines.
func sanitize(s string) string {
	if s == "" {
		return s
	}
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if r < 0x20 || r == 0x7f {
			b.WriteByte(' ')
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

// Logging wraps next and logs a single line per request containing exactly the
// request method, the path without query string (r.URL.Path), the response
// status code and the request duration. Client-controlled values are sanitized
// so no control characters can forge extra log lines. It never logs query
// parameters, so the user parameter from evaluation requests never appears in
// the log, and it sets no CORS headers.
func Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &responseWriter{ResponseWriter: w}
		next.ServeHTTP(rw, r)
		status := rw.status
		if status == 0 {
			status = http.StatusOK
		}
		log.Printf("%s %s %d %s",
			r.Method,
			sanitize(r.URL.Path),
			status,
			time.Since(start),
		)
	})
}
