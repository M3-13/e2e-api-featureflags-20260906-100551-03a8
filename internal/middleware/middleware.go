package middleware

import "net/http"

// Logging wraps next and passes every request straight through. The actual
// logging behaviour is implemented by the logging-middleware ticket.
func Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)
	})
}
