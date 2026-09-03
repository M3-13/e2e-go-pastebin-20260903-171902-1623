package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestListEmptyReturnsEmptyArray(t *testing.T) {
	api := newTestAPI()
	req := httptest.NewRequest(http.MethodGet, "/pastes", nil)
	rec := httptest.NewRecorder()
	api.routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("expected Content-Type application/json, got %q", ct)
	}
	if trimmed := strings.TrimSpace(rec.Body.String()); trimmed != "[]" {
		t.Fatalf("expected empty JSON array [], got %q", trimmed)
	}
}

func TestListReturnsMetadataWithoutContent(t *testing.T) {
	api := newTestAPI()
	now := time.Now().UTC()
	api.store.mu.Lock()
	api.store.pastes["abc123"] = Paste{
		ID:        "abc123",
		Content:   "secret content",
		Language:  "go",
		CreatedAt: now,
		ExpiresAt: nil,
	}
	api.store.mu.Unlock()

	req := httptest.NewRequest(http.MethodGet, "/pastes", nil)
	rec := httptest.NewRecorder()
	api.routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var metas []map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &metas); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}
	if len(metas) != 1 {
		t.Fatalf("expected 1 paste, got %d", len(metas))
	}
	m := metas[0]
	if m["id"] != "abc123" {
		t.Fatalf("expected id abc123, got %v", m["id"])
	}
	if m["language"] != "go" {
		t.Fatalf("expected language go, got %v", m["language"])
	}
	if _, ok := m["created_at"]; !ok {
		t.Fatalf("expected created_at field, got %v", m)
	}
	if _, ok := m["expires_at"]; !ok {
		t.Fatalf("expected expires_at field, got %v", m)
	}
	if _, ok := m["content"]; ok {
		t.Fatalf("content field must not be present, got %v", m)
	}
}

func TestListRemovesExpiredPastes(t *testing.T) {
	api := newTestAPI()
	now := time.Now().UTC()
	past := now.Add(-time.Hour)
	future := now.Add(time.Hour)
	api.store.mu.Lock()
	api.store.pastes["expired"] = Paste{
		ID:        "expired",
		Content:   "old",
		Language:  "text",
		CreatedAt: past,
		ExpiresAt: &past,
	}
	api.store.pastes["alive"] = Paste{
		ID:        "alive",
		Content:   "new",
		Language:  "text",
		CreatedAt: now,
		ExpiresAt: &future,
	}
	api.store.mu.Unlock()

	req := httptest.NewRequest(http.MethodGet, "/pastes", nil)
	rec := httptest.NewRecorder()
	api.routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var metas []PasteMeta
	if err := json.Unmarshal(rec.Body.Bytes(), &metas); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}
	if len(metas) != 1 {
		t.Fatalf("expected 1 remaining paste, got %d", len(metas))
	}
	if metas[0].ID != "alive" {
		t.Fatalf("expected alive paste, got %q", metas[0].ID)
	}

	api.store.mu.RLock()
	_, expiredStillThere := api.store.pastes["expired"]
	_, aliveStillThere := api.store.pastes["alive"]
	api.store.mu.RUnlock()
	if expiredStillThere {
		t.Fatalf("expired paste should be removed from store")
	}
	if !aliveStillThere {
		t.Fatalf("alive paste should remain in store")
	}
}
