package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"
)

func postPaste(t *testing.T, api *API, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/pastes", strings.NewReader(body))
	rec := httptest.NewRecorder()
	api.routes().ServeHTTP(rec, req)
	return rec
}

func decodeID(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}
	id, ok := body["id"]
	if !ok {
		t.Fatalf("expected id field in body, got %v", body)
	}
	return id
}

func TestCreatePasteReturns201WithID(t *testing.T) {
	api := newTestAPI()
	rec := postPaste(t, api, `{"content":"hello world","language":"go"}`)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("expected Content-Type application/json, got %q", ct)
	}
	id := decodeID(t, rec)
	if len(id) != 16 {
		t.Fatalf("expected 16 hex char id, got %q", id)
	}

	api.store.mu.RLock()
	p, ok := api.store.pastes[id]
	api.store.mu.RUnlock()
	if !ok {
		t.Fatalf("expected paste %q in store", id)
	}
	if p.Content != "hello world" {
		t.Fatalf("expected content hello world, got %q", p.Content)
	}
	if p.Language != "go" {
		t.Fatalf("expected language go, got %q", p.Language)
	}
	if p.ExpiresAt != nil {
		t.Fatalf("expected expires_at nil, got %v", p.ExpiresAt)
	}
}

func TestCreatePasteTwiceYieldsDifferentIDs(t *testing.T) {
	api := newTestAPI()
	rec1 := postPaste(t, api, `{"content":"first"}`)
	rec2 := postPaste(t, api, `{"content":"second"}`)

	if rec1.Code != http.StatusCreated || rec2.Code != http.StatusCreated {
		t.Fatalf("expected 201/201, got %d/%d", rec1.Code, rec2.Code)
	}
	id1 := decodeID(t, rec1)
	id2 := decodeID(t, rec2)
	if id1 == id2 {
		t.Fatalf("expected different ids, got %q twice", id1)
	}
}

func TestCreatePasteEmptyContentReturns400(t *testing.T) {
	api := newTestAPI()
	rec := postPaste(t, api, `{"content":""}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestCreatePasteInvalidJSONReturns400(t *testing.T) {
	api := newTestAPI()
	rec := postPaste(t, api, `{"content":`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestCreatePasteNegativeExpiresReturns400(t *testing.T) {
	api := newTestAPI()
	rec := postPaste(t, api, `{"content":"x","expires_in_seconds":-1}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestCreatePasteExpiresOver10YearsReturns400(t *testing.T) {
	api := newTestAPI()
	body := `{"content":"x","expires_in_seconds":` + strconv.FormatInt(maxExpirySeconds+1, 10) + `}`
	rec := postPaste(t, api, body)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestCreatePasteBodyOver1MiBReturns413(t *testing.T) {
	api := newTestAPI()
	big := strings.Repeat("a", maxBodyBytes+1)
	rec := postPaste(t, api, `{"content":"`+big+`"}`)
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected 413, got %d", rec.Code)
	}
}

func TestCreatePasteZeroExpiresMeansNever(t *testing.T) {
	api := newTestAPI()
	rec := postPaste(t, api, `{"content":"x","expires_in_seconds":0}`)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	id := decodeID(t, rec)
	api.store.mu.RLock()
	p := api.store.pastes[id]
	api.store.mu.RUnlock()
	if p.ExpiresAt != nil {
		t.Fatalf("expected expires_at null, got %v", p.ExpiresAt)
	}
}

func TestCreatePastePositiveExpirySetsExpiresAt(t *testing.T) {
	api := newTestAPI()
	rec := postPaste(t, api, `{"content":"x","expires_in_seconds":3600}`)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", rec.Code)
	}
	id := decodeID(t, rec)
	api.store.mu.RLock()
	p := api.store.pastes[id]
	api.store.mu.RUnlock()
	if p.ExpiresAt == nil {
		t.Fatalf("expected non-nil expires_at")
	}
	want := p.CreatedAt.Add(time.Hour)
	if !p.ExpiresAt.Equal(want) {
		t.Fatalf("expected expires_at %v, got %v", want, p.ExpiresAt)
	}
}

func TestCreatePasteDefaultsLanguageToText(t *testing.T) {
	api := newTestAPI()
	rec := postPaste(t, api, `{"content":"x"}`)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", rec.Code)
	}
	id := decodeID(t, rec)
	api.store.mu.RLock()
	p := api.store.pastes[id]
	api.store.mu.RUnlock()
	if p.Language != "text" {
		t.Fatalf("expected default language text, got %q", p.Language)
	}
}
