package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"featureflags/internal/store"
)

func newTestService() *Service {
	return New(store.New())
}

func doCreate(t *testing.T, svc *Service, body string) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/flags", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	svc.CreateFlag(w, r)
	return w
}

func doList(t *testing.T, svc *Service) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/flags", nil)
	svc.ListFlags(w, r)
	return w
}

func doGet(t *testing.T, svc *Service, key string) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/flags/"+key, nil)
	r.SetPathValue("key", key)
	svc.GetFlag(w, r)
	return w
}

func doUpdate(t *testing.T, svc *Service, key, body string) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPut, "/flags/"+key, strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r.SetPathValue("key", key)
	svc.UpdateFlag(w, r)
	return w
}

func doDelete(t *testing.T, svc *Service, key string) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodDelete, "/flags/"+key, nil)
	r.SetPathValue("key", key)
	svc.DeleteFlag(w, r)
	return w
}

func decodeFlag(t *testing.T, rec *httptest.ResponseRecorder) store.Flag {
	t.Helper()
	var f store.Flag
	if err := json.Unmarshal(rec.Body.Bytes(), &f); err != nil {
		t.Fatalf("response is not a valid flag JSON: %v (body %q)", err, rec.Body.String())
	}
	return f
}

func decodeFlags(t *testing.T, rec *httptest.ResponseRecorder) []store.Flag {
	t.Helper()
	var fs []store.Flag
	if err := json.Unmarshal(rec.Body.Bytes(), &fs); err != nil {
		t.Fatalf("response is not a valid flag array JSON: %v (body %q)", err, rec.Body.String())
	}
	return fs
}

func decodeError(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	var m map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &m); err != nil {
		t.Fatalf("response is not a valid JSON error object: %v (body %q)", err, rec.Body.String())
	}
	if m["error"] == "" {
		t.Fatalf("response JSON has no error field: %q", rec.Body.String())
	}
	return m["error"]
}

func TestCreateFlagValid(t *testing.T) {
	svc := newTestService()
	rec := doCreate(t, svc, `{"key":"my_flag","enabled":true,"description":"hi","rollout_percent":50}`)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusCreated)
	}
	f := decodeFlag(t, rec)
	if f.Key != "my_flag" || !f.Enabled || f.Description != "hi" || f.RolloutPercent != 50 {
		t.Fatalf("flag = %+v, want populated values", f)
	}
}

func TestCreateFlagDefaults(t *testing.T) {
	svc := newTestService()
	rec := doCreate(t, svc, `{"key":"plain"}`)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusCreated)
	}
	f := decodeFlag(t, rec)
	if f.Key != "plain" {
		t.Fatalf("key = %q, want plain", f.Key)
	}
	if f.Enabled {
		t.Fatalf("enabled = true, want default false")
	}
	if f.Description != "" {
		t.Fatalf("description = %q, want default empty", f.Description)
	}
	if f.RolloutPercent != 100 {
		t.Fatalf("rollout_percent = %d, want default 100", f.RolloutPercent)
	}
}

func TestCreateFlagInvalidInput(t *testing.T) {
	cases := []struct {
		name string
		body string
	}{
		{"empty key", `{"key":""}`},
		{"missing key", `{"enabled":true}`},
		{"invalid json", `{"key":`},
		{"wrong field type", `{"key":123}`},
		{"rollout above range", `{"key":"a","rollout_percent":101}`},
		{"rollout below range", `{"key":"a","rollout_percent":-1}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := newTestService()
			rec := doCreate(t, svc, tc.body)
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
			}
			decodeError(t, rec)
		})
	}
}

func TestCreateFlagDuplicate(t *testing.T) {
	svc := newTestService()
	if rec := doCreate(t, svc, `{"key":"dup"}`); rec.Code != http.StatusCreated {
		t.Fatalf("first create status = %d, want %d", rec.Code, http.StatusCreated)
	}
	rec := doCreate(t, svc, `{"key":"dup"}`)
	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusConflict)
	}
	decodeError(t, rec)
}

func TestListFlagsEmpty(t *testing.T) {
	svc := newTestService()
	rec := doList(t, svc)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	fs := decodeFlags(t, rec)
	if fs == nil || len(fs) != 0 {
		t.Fatalf("flags = %#v, want empty non-nil array", fs)
	}
}

func TestListFlagsPopulated(t *testing.T) {
	svc := newTestService()
	doCreate(t, svc, `{"key":"a"}`)
	doCreate(t, svc, `{"key":"b"}`)
	rec := doList(t, svc)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	fs := decodeFlags(t, rec)
	if len(fs) != 2 {
		t.Fatalf("len(flags) = %d, want 2", len(fs))
	}
}

func TestGetFlag(t *testing.T) {
	svc := newTestService()
	doCreate(t, svc, `{"key":"present","enabled":true}`)

	rec := doGet(t, svc, "present")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if f := decodeFlag(t, rec); f.Key != "present" || !f.Enabled {
		t.Fatalf("flag = %+v, want key present and enabled", f)
	}
}

func TestGetFlagUnknown(t *testing.T) {
	svc := newTestService()
	rec := doGet(t, svc, "missing")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
	decodeError(t, rec)
}

func TestUpdateFlag(t *testing.T) {
	svc := newTestService()
	doCreate(t, svc, `{"key":"f","enabled":false,"description":"old","rollout_percent":10}`)

	rec := doUpdate(t, svc, "f", `{"enabled":true,"description":"new","rollout_percent":80}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	f := decodeFlag(t, rec)
	if !f.Enabled || f.Description != "new" || f.RolloutPercent != 80 {
		t.Fatalf("flag = %+v, want updated values", f)
	}
}

func TestUpdateFlagPartial(t *testing.T) {
	svc := newTestService()
	doCreate(t, svc, `{"key":"f","enabled":false,"description":"old","rollout_percent":10}`)

	rec := doUpdate(t, svc, "f", `{"enabled":true}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	f := decodeFlag(t, rec)
	if !f.Enabled {
		t.Fatalf("enabled = false, want true")
	}
	if f.Description != "old" {
		t.Fatalf("description = %q, want unchanged 'old'", f.Description)
	}
	if f.RolloutPercent != 10 {
		t.Fatalf("rollout_percent = %d, want unchanged 10", f.RolloutPercent)
	}
}

func TestUpdateFlagUnknown(t *testing.T) {
	svc := newTestService()
	rec := doUpdate(t, svc, "missing", `{"enabled":true}`)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
	decodeError(t, rec)
}

func TestUpdateFlagInvalidRollout(t *testing.T) {
	svc := newTestService()
	doCreate(t, svc, `{"key":"f"}`)
	rec := doUpdate(t, svc, "f", `{"rollout_percent":150}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
	decodeError(t, rec)
}

func TestDeleteFlag(t *testing.T) {
	svc := newTestService()
	doCreate(t, svc, `{"key":"f"}`)

	rec := doDelete(t, svc, "f")
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}

	rec = doDelete(t, svc, "f")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("second delete status = %d, want %d", rec.Code, http.StatusNotFound)
	}
	decodeError(t, rec)
}

func TestKeyCharset(t *testing.T) {
	valid := []string{"a", "ABC", "a1", "a_b", "a-b", "my_flag_2", "xYz_9-8"}
	invalid := []string{"../flag", "a/b", "a b", "a.b", "föö", "a?b", ".", "..", "a=b"}

	svc := newTestService()
	for _, k := range valid {
		if rec := doCreate(t, svc, `{"key":"`+k+`"}`); rec.Code != http.StatusCreated {
			t.Fatalf("key %q: status = %d, want %d", k, rec.Code, http.StatusCreated)
		}
	}
	for _, k := range invalid {
		svc2 := newTestService()
		rec := doCreate(t, svc2, `{"key":"`+k+`"}`)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("key %q: status = %d, want %d", k, rec.Code, http.StatusBadRequest)
		}
		decodeError(t, rec)
	}
}

func TestCreateBodyTooLarge(t *testing.T) {
	svc := newTestService()
	body := `{"key":"` + strings.Repeat("x", maxBodyBytes) + `"}`
	rec := doCreate(t, svc, body)
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusRequestEntityTooLarge)
	}
	decodeError(t, rec)
}

func TestUpdateBodyTooLarge(t *testing.T) {
	svc := newTestService()
	doCreate(t, svc, `{"key":"f"}`)
	body := `{"description":"` + strings.Repeat("x", maxBodyBytes) + `"}`
	rec := doUpdate(t, svc, "f", body)
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusRequestEntityTooLarge)
	}
	decodeError(t, rec)
}

func TestCreateDescriptionTooLong(t *testing.T) {
	svc := newTestService()
	body := `{"key":"f","description":"` + strings.Repeat("x", maxDescriptionLen+1) + `"}`
	rec := doCreate(t, svc, body)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
	if msg := decodeError(t, rec); msg != "description too long" {
		t.Fatalf("error = %q, want %q", msg, "description too long")
	}
}

func TestCreateDescriptionAtLimit(t *testing.T) {
	svc := newTestService()
	body := `{"key":"f","description":"` + strings.Repeat("x", maxDescriptionLen) + `"}`
	rec := doCreate(t, svc, body)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusCreated)
	}
}

func TestUpdateDescriptionTooLong(t *testing.T) {
	svc := newTestService()
	doCreate(t, svc, `{"key":"f"}`)
	body := `{"description":"` + strings.Repeat("x", maxDescriptionLen+1) + `"}`
	rec := doUpdate(t, svc, "f", body)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
	if msg := decodeError(t, rec); msg != "description too long" {
		t.Fatalf("error = %q, want %q", msg, "description too long")
	}
}

func TestCreateMissingContentType(t *testing.T) {
	svc := newTestService()
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/flags", strings.NewReader(`{"key":"f"}`))
	svc.CreateFlag(w, r)
	if w.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusUnsupportedMediaType)
	}
	if msg := decodeError(t, w); msg != "content type must be application/json" {
		t.Fatalf("error = %q, want %q", msg, "content type must be application/json")
	}
}

func TestCreateWrongContentType(t *testing.T) {
	svc := newTestService()
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/flags", strings.NewReader(`{"key":"f"}`))
	r.Header.Set("Content-Type", "text/plain")
	svc.CreateFlag(w, r)
	if w.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusUnsupportedMediaType)
	}
	decodeError(t, w)
}

func TestUpdateMissingContentType(t *testing.T) {
	svc := newTestService()
	doCreate(t, svc, `{"key":"f"}`)
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPut, "/flags/f", strings.NewReader(`{"enabled":true}`))
	r.SetPathValue("key", "f")
	svc.UpdateFlag(w, r)
	if w.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusUnsupportedMediaType)
	}
	if msg := decodeError(t, w); msg != "content type must be application/json" {
		t.Fatalf("error = %q, want %q", msg, "content type must be application/json")
	}
}

func TestCreateContentTypeWithCharset(t *testing.T) {
	svc := newTestService()
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/flags", strings.NewReader(`{"key":"f"}`))
	r.Header.Set("Content-Type", "application/json; charset=utf-8")
	svc.CreateFlag(w, r)
	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusCreated)
	}
}
