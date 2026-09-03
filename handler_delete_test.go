package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestDeleteExistingPasteReturns204AndRemovesIt(t *testing.T) {
	api := newTestAPI()
	id := "abc123"
	api.store.pastes[id] = Paste{
		ID:        id,
		Content:   "hello",
		Language:  "text",
		CreatedAt: time.Now().UTC(),
	}

	req := httptest.NewRequest(http.MethodDelete, "/pastes/"+id, nil)
	rec := httptest.NewRecorder()
	api.routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rec.Code)
	}
	if rec.Body.Len() != 0 {
		t.Fatalf("expected empty body, got %q", rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); ct != "" {
		t.Fatalf("expected no Content-Type, got %q", ct)
	}

	api.store.mu.RLock()
	_, ok := api.store.pastes[id]
	api.store.mu.RUnlock()
	if ok {
		t.Fatalf("expected paste %q to be removed", id)
	}
}

func TestDeleteUnknownIDReturns404(t *testing.T) {
	api := newTestAPI()

	req := httptest.NewRequest(http.MethodDelete, "/pastes/does-not-exist", nil)
	rec := httptest.NewRecorder()
	api.routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestDeleteExpiredPasteReturns404AndRemovesIt(t *testing.T) {
	api := newTestAPI()
	id := "expired1"
	expires := time.Now().UTC().Add(-time.Minute)
	api.store.pastes[id] = Paste{
		ID:        id,
		Content:   "old",
		Language:  "text",
		CreatedAt: time.Now().UTC().Add(-time.Hour),
		ExpiresAt: &expires,
	}

	req := httptest.NewRequest(http.MethodDelete, "/pastes/"+id, nil)
	rec := httptest.NewRecorder()
	api.routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}

	api.store.mu.RLock()
	_, ok := api.store.pastes[id]
	api.store.mu.RUnlock()
	if ok {
		t.Fatalf("expected expired paste %q to be removed", id)
	}
}
