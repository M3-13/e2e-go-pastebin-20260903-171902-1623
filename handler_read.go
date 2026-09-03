package main

import (
	"net/http"
	"time"
)

func (a *API) handleGet(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	w.Header().Set("Cache-Control", "no-store")

	a.store.mu.RLock()
	p, ok := a.store.pastes[id]
	a.store.mu.RUnlock()

	if !ok {
		writeError(w, http.StatusNotFound, "paste not found")
		return
	}

	if isExpired(p, time.Now().UTC()) {
		a.store.mu.Lock()
		delete(a.store.pastes, id)
		a.store.mu.Unlock()
		writeError(w, http.StatusNotFound, "paste not found")
		return
	}

	writeJSON(w, http.StatusOK, p)
}
