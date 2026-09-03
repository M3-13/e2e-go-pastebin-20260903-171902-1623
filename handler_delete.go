package main

import (
	"net/http"
	"time"
)

func (a *API) handleDelete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	a.store.mu.Lock()
	paste, ok := a.store.pastes[id]
	delete(a.store.pastes, id)
	a.store.mu.Unlock()

	if !ok || isExpired(paste, time.Now().UTC()) {
		writeError(w, http.StatusNotFound, "paste not found")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
