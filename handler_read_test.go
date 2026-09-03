package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func putPaste(t *testing.T, api *API, p Paste) {
	t.Helper()
	api.store.mu.Lock()
	api.store.pastes[p.ID] = p
	api.store.mu.Unlock()
}

func TestGetExistingPasteReturns200WithAllFields(t *testing.T) {
	api := newTestAPI()
	created := time.Now().UTC()
	putPaste(t, api, Paste{
		ID:        "abc123",
		Content:   "hello world",
		Language:  "text",
		CreatedAt: created,
		ExpiresAt: nil,
	})

	req := httptest.NewRequest(http.MethodGet, "/pastes/abc123", nil)
	rec := httptest.NewRecorder()
	api.routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var got Paste
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}
	if got.ID != "abc123" {
		t.Fatalf("expected id abc123, got %q", got.ID)
	}
	if got.Content != "hello world" {
		t.Fatalf("expected content hello world, got %q", got.Content)
	}
	if got.Language != "text" {
		t.Fatalf("expected language text, got %q", got.Language)
	}
	if !got.CreatedAt.Equal(created) {
		t.Fatalf("expected created_at %v, got %v", created, got.CreatedAt)
	}
	if got.ExpiresAt != nil {
		t.Fatalf("expected expires_at null, got %v", got.ExpiresAt)
	}
}

func TestGetUnknownIDReturns404(t *testing.T) {
	api := newTestAPI()

	req := httptest.NewRequest(http.MethodGet, "/pastes/nonexistent", nil)
	rec := httptest.NewRecorder()
	api.routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
	if cc := rec.Header().Get("Cache-Control"); cc != "no-store" {
		t.Fatalf("expected Cache-Control no-store on 404, got %q", cc)
	}
	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}
	if _, ok := body["error"]; !ok {
		t.Fatalf("expected error field in body, got %v", body)
	}
}

func TestGetExpiredPasteReturns404(t *testing.T) {
	api := newTestAPI()
	exp := time.Now().UTC().Add(-time.Hour)
	putPaste(t, api, Paste{
		ID:        "expired",
		Content:   "old",
		Language:  "text",
		CreatedAt: time.Now().UTC(),
		ExpiresAt: &exp,
	})

	req := httptest.NewRequest(http.MethodGet, "/pastes/expired", nil)
	rec := httptest.NewRecorder()
	api.routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
	if cc := rec.Header().Get("Cache-Control"); cc != "no-store" {
		t.Fatalf("expected Cache-Control no-store on 404, got %q", cc)
	}
}

func TestGetSetsCacheControlNoStore(t *testing.T) {
	api := newTestAPI()
	putPaste(t, api, Paste{
		ID:        "abc",
		Content:   "x",
		Language:  "text",
		CreatedAt: time.Now().UTC(),
	})

	req := httptest.NewRequest(http.MethodGet, "/pastes/abc", nil)
	rec := httptest.NewRecorder()
	api.routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if cc := rec.Header().Get("Cache-Control"); cc != "no-store" {
		t.Fatalf("expected Cache-Control no-store, got %q", cc)
	}
}

func TestGetExpiredPasteRemovesFromMap(t *testing.T) {
	api := newTestAPI()
	exp := time.Now().UTC().Add(-time.Hour)
	putPaste(t, api, Paste{
		ID:        "expired",
		Content:   "old",
		Language:  "text",
		CreatedAt: time.Now().UTC(),
		ExpiresAt: &exp,
	})

	req := httptest.NewRequest(http.MethodGet, "/pastes/expired", nil)
	rec := httptest.NewRecorder()
	api.routes().ServeHTTP(rec, req)

	api.store.mu.RLock()
	_, ok := api.store.pastes["expired"]
	api.store.mu.RUnlock()
	if ok {
		t.Fatalf("expected expired paste to be removed from map")
	}
}
