package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"featureflags/internal/store"
)

func newEvaluateService() *Service {
	s := store.New()
	_ = s.Create(store.Flag{Key: "myflag", Enabled: true, RolloutPercent: 100})
	_ = s.Create(store.Flag{Key: "off", Enabled: false, RolloutPercent: 100})
	return New(s)
}

func evaluateRequest(key, query string) *http.Request {
	req := httptest.NewRequest(http.MethodGet, "/flags/"+key+"/evaluate"+query, nil)
	req.SetPathValue("key", key)
	return req
}

func TestEvaluateMissingUser(t *testing.T) {
	s := newEvaluateService()
	rec := httptest.NewRecorder()
	s.Evaluate(rec, evaluateRequest("myflag", ""))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestEvaluateEmptyUser(t *testing.T) {
	s := newEvaluateService()
	rec := httptest.NewRecorder()
	s.Evaluate(rec, evaluateRequest("myflag", "?user="))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestEvaluateUnknownKey(t *testing.T) {
	s := newEvaluateService()
	rec := httptest.NewRecorder()
	s.Evaluate(rec, evaluateRequest("nope", "?user=alice"))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestEvaluateValid(t *testing.T) {
	s := newEvaluateService()
	rec := httptest.NewRecorder()
	s.Evaluate(rec, evaluateRequest("myflag", "?user=alice"))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json; charset=utf-8" {
		t.Fatalf("Content-Type = %q, want application/json; charset=utf-8", ct)
	}
	var body map[string]bool
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}
	if !body["active"] {
		t.Fatalf("active = false, want true (enabled, rollout 100)")
	}
}

func TestEvaluateDisabledFlag(t *testing.T) {
	s := newEvaluateService()
	rec := httptest.NewRecorder()
	s.Evaluate(rec, evaluateRequest("off", "?user=alice"))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	var body map[string]bool
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}
	if body["active"] {
		t.Fatal("disabled flag evaluated active")
	}
}
