package main

import "net/http"

func (a *API) handleCreate(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusNotImplemented, "not implemented")
}
