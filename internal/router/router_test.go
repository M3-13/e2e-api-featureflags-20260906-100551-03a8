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

func TestRouterPreservesErrorStatusWithBody(t *testing.T) {
	// The router must pass through a handler's own error status, body and
	// Content-Type rather than substituting its own 404/405. GET on a known
	// route with an unknown key exercises the handler's own 404 "flag not found".
	h := newTestHandler()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/flags/nope", nil)
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json; charset=utf-8" {
		t.Fatalf("Content-Type = %q, want application/json; charset=utf-8", ct)
	}
	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}
	if body["error"] != "flag not found" {
		t.Fatalf("body = %v, want error=flag not found", body)
	}
}

func TestKeyRoutesAreWired(t *testing.T) {
	h := newTestHandler()

	create := httptest.NewRequest(http.MethodPost, "/flags",
		strings.NewReader(`{"key":"myflag","enabled":false,"description":"d","rollout_percent":100}`))
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
