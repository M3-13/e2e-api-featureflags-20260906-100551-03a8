package router

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"featureflags/internal/handlers"
	"featureflags/internal/store"
)

func newTestHandler() http.Handler {
	return New(handlers.New(store.New()))
}

func TestHealth(t *testing.T) {
	h := newTestHandler()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json; charset=utf-8" {
		t.Fatalf("Content-Type = %q, want application/json; charset=utf-8", ct)
	}
	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}
	if body["status"] != "ok" {
		t.Fatalf("body = %v, want status ok", body)
	}
}

func TestUnknownPathReturns404JSON(t *testing.T) {
	h := newTestHandler()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/no-such-path", nil)
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
	assertJSONError(t, rec)
}

func TestWrongMethodReturns405JSON(t *testing.T) {
	h := newTestHandler()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/healthz", nil)
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
	assertJSONError(t, rec)
}

func TestKeyRoutesAreWired(t *testing.T) {
	h := newTestHandler()

	create := httptest.NewRequest(http.MethodPost, "/flags",
		strings.NewReader(`{"key":"myflag","enabled":false,"description":"d","rollout_percent":100}`))
	create.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, create)
	if rec.Code != http.StatusCreated {
		t.Fatalf("POST /flags status = %d, want %d", rec.Code, http.StatusCreated)
	}

	get := httptest.NewRequest(http.MethodGet, "/flags/myflag", nil)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, get)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /flags/{key} status = %d, want %d", rec.Code, http.StatusOK)
	}
	var flag map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &flag); err != nil {
		t.Fatalf("GET /flags/{key} body not valid JSON: %v", err)
	}
	if flag["key"] != "myflag" {
		t.Fatalf("GET /flags/{key} key = %v, want myflag", flag["key"])
	}

	put := httptest.NewRequest(http.MethodPut, "/flags/myflag",
		strings.NewReader(`{"enabled":true}`))
	put.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, put)
	if rec.Code != http.StatusOK {
		t.Fatalf("PUT /flags/{key} status = %d, want %d", rec.Code, http.StatusOK)
	}

	eval := httptest.NewRequest(http.MethodGet, "/flags/myflag/evaluate?user=alice", nil)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, eval)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /flags/{key}/evaluate status = %d, want %d", rec.Code, http.StatusOK)
	}

	del := httptest.NewRequest(http.MethodDelete, "/flags/myflag", nil)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, del)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("DELETE /flags/{key} status = %d, want %d", rec.Code, http.StatusNoContent)
	}

	unknown := httptest.NewRequest(http.MethodGet, "/flags/does-not-exist", nil)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, unknown)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("GET /flags/{unknown} status = %d, want %d", rec.Code, http.StatusNotFound)
	}
	var errBody map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &errBody); err != nil {
		t.Fatalf("GET /flags/{unknown} body not valid JSON: %v", err)
	}
	if errBody["error"] != "flag not found" {
		t.Fatalf("GET /flags/{unknown} error = %q, want handler text %q", errBody["error"], "flag not found")
	}
}

func TestGenericHandlerIsWiredWithoutPanic(t *testing.T) {
	h := New(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestLargeResponseReturns413(t *testing.T) {
	h := New(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		chunk := make([]byte, 1024*1024)
		for i := 0; i < 8; i++ {
			_, _ = w.Write(chunk)
		}
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusRequestEntityTooLarge)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json; charset=utf-8" {
		t.Fatalf("Content-Type = %q, want application/json; charset=utf-8", ct)
	}
	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}
	if body["error"] != "response too large" {
		t.Fatalf("error = %q, want %q", body["error"], "response too large")
	}
}

func TestNormalResponsePassesThrough(t *testing.T) {
	h := New(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Custom", "yes")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("hello"))
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if got := rec.Header().Get("X-Custom"); got != "yes" {
		t.Fatalf("X-Custom = %q, want yes", got)
	}
	if got := rec.Body.String(); got != "hello" {
		t.Fatalf("body = %q, want hello", got)
	}
}

func assertJSONError(t *testing.T, rec *httptest.ResponseRecorder) {
	t.Helper()
	if ct := rec.Header().Get("Content-Type"); ct != "application/json; charset=utf-8" {
		t.Fatalf("Content-Type = %q, want application/json; charset=utf-8", ct)
	}
	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}
	if _, ok := body["error"]; !ok {
		t.Fatalf("body = %v, want an error field", body)
	}
}
