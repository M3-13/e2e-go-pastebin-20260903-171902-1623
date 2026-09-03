package main

import "net/http"

func (a *API) handleDelete(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusNotImplemented, "not implemented")
}
