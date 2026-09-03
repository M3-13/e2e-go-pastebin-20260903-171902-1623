package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"
)

const (
	maxBodyBytes     = 1 << 20                     // 1 MiB
	maxExpirySeconds = int64(10 * 365 * 24 * 3600) // 10 Jahre in Sekunden
)

type createRequest struct {
	Content          string `json:"content"`
	Language         string `json:"language"`
	ExpiresInSeconds *int64 `json:"expires_in_seconds"`
}

func (a *API) handleCreate(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)

	var req createRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			writeError(w, http.StatusRequestEntityTooLarge, "request body too large")
			return
		}
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	if strings.TrimSpace(req.Content) == "" {
		writeError(w, http.StatusBadRequest, "content must not be empty")
		return
	}

	if req.Language == "" {
		req.Language = "text"
	}

	// Leseart der Ablaufzeit (bindend, Entscheidung des Tech Leads):
	// expires_in_seconds == 0 oder fehlend bedeutet „läuft nie ab" und wird mit
	// 201 akzeptiert, expires_at bleibt nil. Zurückgewiesen mit 400 werden NUR
	// negative Werte sowie Werte größer als 10 Jahre. AC-13 („positive Ganzzahl")
	// ist als Ausschluss negativer Werte plus Maximalbegrenzung zu verstehen,
	// NICHT als Ablehnung von 0.
	var expiresAt *time.Time
	if req.ExpiresInSeconds != nil {
		secs := *req.ExpiresInSeconds
		if secs < 0 {
			writeError(w, http.StatusBadRequest, "expires_in_seconds must not be negative")
			return
		}
		if secs > maxExpirySeconds {
			writeError(w, http.StatusBadRequest, "expires_in_seconds exceeds maximum allowed")
			return
		}
		if secs > 0 {
			exp := time.Now().UTC().Add(time.Duration(secs) * time.Second)
			expiresAt = &exp
		}
	}

	id, err := newID()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not generate id")
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

// newID erzeugt eine zufällige Paste-ID: 8 Bytes aus crypto/rand = 64 Bit
// Entropie, hex-kodiert zu 16 Zeichen.
func newID() (string, error) {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
