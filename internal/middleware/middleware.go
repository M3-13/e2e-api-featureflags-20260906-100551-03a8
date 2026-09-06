package middleware

import (
	"crypto/subtle"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
	"unicode"
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

// sanitize removes control characters (newline, carriage return, tab, other
// C0 control bytes, and C1 controls such as U+0085) from a client-controlled
// value so it cannot forge additional log lines.
func sanitize(s string) string {
	if s == "" {
		return s
	}
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if unicode.IsControl(r) {
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

// flagsPath is the base of every protected route, exactly as the shared
// contract declares it. Protected routes are /flags itself and every path
// beneath it (/{key} and /{key}/evaluate), addressed as flagsPath plus a
// trailing segment separator — never a standalone trailing-slash route.
const flagsPath = "/flags"

// RequireAPIKey protects every path under the /flags prefix by requiring a
// Bearer token that matches the FEATUREFLAGS_API_KEY environment variable,
// compared with crypto/subtle.ConstantTimeCompare. /healthz and all other
// paths pass through untouched.
//
// Fail-closed: if FEATUREFLAGS_API_KEY is not set, the service does not start
// unguarded and does not crash at boot — protected routes answer
// 503 {"error":"service locked"} instead. This keeps the service bootable and
// locked even without a configured key.
func RequireAPIKey(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if path != flagsPath && !strings.HasPrefix(path, flagsPath+"/") {
			next.ServeHTTP(w, r)
			return
		}

		key := os.Getenv("FEATUREFLAGS_API_KEY")
		if key == "" {
			writeError(w, http.StatusServiceUnavailable, "service locked")
			return
		}

		auth := r.Header.Get("Authorization")
		const prefix = "Bearer "
		if !strings.HasPrefix(auth, prefix) {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		token := strings.TrimPrefix(auth, prefix)

		if subtle.ConstantTimeCompare([]byte(token), []byte(key)) != 1 {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		next.ServeHTTP(w, r)
	})
}

// writeError writes a JSON error body {"error": msg} with the given status
// and the application/json; charset=utf-8 content type.
func writeError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
