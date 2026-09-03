package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func postPaste(api *API, body string) *httptest.ResponseRecorder {
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
	if !ok || id == "" {
		t.Fatalf("expected non-empty id, got %v", body)
	}
	return id
}

func TestCreatePasteReturns201WithID(t *testing.T) {
	api := newTestAPI()
	rec := postPaste(api, `{"content":"hello world"}`)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("expected Content-Type application/json, got %q", ct)
	}
	id := decodeID(t, rec)
	if len(id) != 16 {
		t.Fatalf("expected 16 hex character id, got %q", id)
	}
}

func TestCreatePasteSecondCallDifferentID(t *testing.T) {
	api := newTestAPI()
	rec1 := postPaste(api, `{"content":"first"}`)
	rec2 := postPaste(api, `{"content":"second"}`)

	if rec1.Code != http.StatusCreated || rec2.Code != http.StatusCreated {
		t.Fatalf("expected 201 on both calls, got %d and %d", rec1.Code, rec2.Code)
	}
	id1 := decodeID(t, rec1)
	id2 := decodeID(t, rec2)
	if id1 == id2 {
		t.Fatalf("expected different ids, both are %q", id1)
	}
}

func TestCreatePasteEmptyContent400(t *testing.T) {
	api := newTestAPI()
	rec := postPaste(api, `{"content":""}`)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("expected Content-Type application/json, got %q", ct)
	}
}

func TestCreatePasteMissingContent400(t *testing.T) {
	api := newTestAPI()
	rec := postPaste(api, `{}`)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestCreatePasteInvalidJSON400(t *testing.T) {
	api := newTestAPI()
	rec := postPaste(api, `{"content": `)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestCreatePasteNegativeExpires400(t *testing.T) {
	api := newTestAPI()
	rec := postPaste(api, `{"content":"x","expires_in_seconds":-1}`)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestCreatePasteTooLargeExpires400(t *testing.T) {
	api := newTestAPI()
	rec := postPaste(api, `{"content":"x","expires_in_seconds":315360001}`)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestCreatePasteBodyTooLarge413(t *testing.T) {
	api := newTestAPI()
	big := strings.Repeat("a", 2<<20)
	rec := postPaste(api, `{"content":"`+big+`"}`)

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected 413, got %d", rec.Code)
	}
}

func TestCreatePasteStoresDefaults(t *testing.T) {
	api := newTestAPI()
	rec := postPaste(api, `{"content":"hello"}`)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", rec.Code)
	}
	id := decodeID(t, rec)

	api.store.mu.RLock()
	paste, ok := api.store.pastes[id]
	api.store.mu.RUnlock()

	if !ok {
		t.Fatalf("paste %q not stored", id)
	}
	if paste.Content != "hello" {
		t.Fatalf("expected content hello, got %q", paste.Content)
	}
	if paste.Language != "text" {
		t.Fatalf("expected default language text, got %q", paste.Language)
	}
	if paste.ExpiresAt != nil {
		t.Fatalf("expected nil expires_at, got %v", paste.ExpiresAt)
	}
}

func TestCreatePastePositiveExpiresSetsExpiresAt(t *testing.T) {
	api := newTestAPI()
	rec := postPaste(api, `{"content":"hello","expires_in_seconds":60}`)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", rec.Code)
	}
	id := decodeID(t, rec)

	api.store.mu.RLock()
	paste, ok := api.store.pastes[id]
	api.store.mu.RUnlock()

	if !ok {
		t.Fatalf("paste %q not stored", id)
	}
	if paste.ExpiresAt == nil {
		t.Fatalf("expected expires_at to be set")
	}
}
