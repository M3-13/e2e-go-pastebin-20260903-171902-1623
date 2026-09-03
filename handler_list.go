package main

import (
	"net/http"
	"time"
)

func (a *API) handleList(w http.ResponseWriter, r *http.Request) {
	now := time.Now().UTC()

	a.store.mu.Lock()
	metas := make([]PasteMeta, 0, len(a.store.pastes))
	for id, p := range a.store.pastes {
		if isExpired(p, now) {
			delete(a.store.pastes, id)
			continue
		}
		metas = append(metas, PasteMeta{
			ID:        p.ID,
			Language:  p.Language,
			CreatedAt: p.CreatedAt,
			ExpiresAt: p.ExpiresAt,
		})
	}
	a.store.mu.Unlock()

	writeJSON(w, http.StatusOK, metas)
}
