package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"time"
)

const (
	maxBodySize      = 1 << 20                 // 1 MiB
	maxExpirySeconds = 10 * 365 * 24 * 60 * 60 // 10 Jahre
)

type createRequest struct {
	Content          string `json:"content"`
	Language         string `json:"language"`
	ExpiresInSeconds *int64 `json:"expires_in_seconds"`
}

func (a *API) handleCreate(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodySize)

	var req createRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			writeError(w, http.StatusRequestEntityTooLarge, "request body too large")
			return
		}
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Content == "" {
		writeError(w, http.StatusBadRequest, "content must not be empty")
		return
	}

	if req.Language == "" {
		req.Language = "text"
	}

	var expiresAt *time.Time
	if req.ExpiresInSeconds != nil {
		seconds := *req.ExpiresInSeconds
		if seconds < 0 {
			writeError(w, http.StatusBadRequest, "expires_in_seconds must be positive")
			return
		}
		if seconds > maxExpirySeconds {
			writeError(w, http.StatusBadRequest, "expires_in_seconds too large")
			return
		}
		if seconds > 0 {
			exp := time.Now().UTC().Add(time.Duration(seconds) * time.Second)
			expiresAt = &exp
		}
	}

	id, err := newID()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	paste := Paste{
		ID:        id,
		Content:   req.Content,
		Language:  req.Language,
		CreatedAt: time.Now().UTC(),
		ExpiresAt: expiresAt,
	}

	a.store.mu.Lock()
	a.store.pastes[id] = paste
	a.store.mu.Unlock()

	writeJSON(w, http.StatusCreated, map[string]string{"id": id})
}

func newID() (string, error) {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
