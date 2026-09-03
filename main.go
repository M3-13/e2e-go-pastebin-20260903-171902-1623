package main

import (
	"log"
	"net/http"
)

type API struct {
	store *Store
}

func main() {
	api := &API{store: NewStore()}
	server := &http.Server{
		Addr:    ":8080",
		Handler: api.routes(),
	}
	log.Fatal(server.ListenAndServe())
}

func (a *API) routes() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", a.handleHealth)
	mux.HandleFunc("/pastes", a.dispatchPastes)
	mux.HandleFunc("/pastes/{id}", a.dispatchPasteByID)
	return mux
}

func (a *API) dispatchPastes(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		a.handleList(w, r)
	case http.MethodPost:
		a.handleCreate(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (a *API) dispatchPasteByID(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		a.handleGet(w, r)
	case http.MethodDelete:
		a.handleDelete(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (a *API) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
