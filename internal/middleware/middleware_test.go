package middleware

import (
	"bytes"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLoggingLogsMethodPathStatusDuration(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(nil)

	handler := Logging(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))

	req := httptest.NewRequest(http.MethodPost, "/flags/abc", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	out := buf.String()
	if !strings.Contains(out, http.MethodPost) {
		t.Fatalf("log output %q missing method", out)
	}
	if !strings.Contains(out, "/flags/abc") {
		t.Fatalf("log output %q missing path", out)
	}
	if !strings.Contains(out, "201") {
		t.Fatalf("log output %q missing status", out)
	}
	// Duration is the final field; the line ends with a time.Duration value.
	fields := strings.Fields(out)
	if len(fields) < 4 {
		t.Fatalf("log output %q missing duration field", out)
	}
	if !strings.Contains(fields[len(fields)-1], "s") && !strings.Contains(fields[len(fields)-1], "ms") && !strings.Contains(fields[len(fields)-1], "µs") && !strings.Contains(fields[len(fields)-1], "ns") {
		t.Fatalf("log output %q duration field %q does not look like a duration", out, fields[len(fields)-1])
	}
}

func TestLoggingOmitsQueryString(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(nil)

	handler := Logging(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	req := httptest.NewRequest(http.MethodGet, "/flags/abc/evaluate?user=secret", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	out := buf.String()
	if strings.Contains(out, "secret") {
		t.Fatalf("log output %q leaks user parameter", out)
	}
	if strings.Contains(out, "user=") {
		t.Fatalf("log output %q contains query string", out)
	}
}

func TestLoggingSanitizesControlCharacters(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(nil)

	handler := Logging(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	req := httptest.NewRequest(http.MethodGet, "/flags/a%0Ab%0Dc%09d", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	out := buf.String()
	// The logger appends its own trailing newline; a sanitized path must not
	// produce any interior line breaks, so the output stays a single line.
	if strings.Count(out, "\n") > 1 {
		t.Fatalf("log output %q has more than one line", out)
	}
	if strings.Contains(out, "\r") {
		t.Fatalf("log output %q contains carriage return", out)
	}
	if strings.Contains(out, "\t") {
		t.Fatalf("log output %q contains tab", out)
	}
	if !strings.Contains(out, "/flags/a b c d") {
		t.Fatalf("log output %q missing sanitized path", out)
	}
}
