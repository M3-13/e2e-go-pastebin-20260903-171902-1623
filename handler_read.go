package main

import "net/http"

func (a *API) handleGet(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusNotImplemented, "not implemented")
}
