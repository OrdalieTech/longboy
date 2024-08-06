package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"longboy/backend"
	"net/http"
	"strings"
)

func setupRoutes(db *sql.DB) {
	http.HandleFunc("/api/action-chains", func(w http.ResponseWriter, r *http.Request) {
		// Check if the path has an ID
		id := strings.TrimPrefix(r.URL.Path, "/api/action-chains/")

		switch {
		case id == "" && r.Method == http.MethodGet:
			handleListActionChains(db, w, r)
		case id == "" && r.Method == http.MethodPost:
			handleCreateActionChain(db, w, r)
		case id != "" && r.Method == http.MethodGet:
			handleGetActionChain(db, w, r)
		case id != "" && r.Method == http.MethodPut:
			handleUpdateActionChain(db, w, r)
		case id != "" && r.Method == http.MethodDelete:
			handleDeleteActionChain(db, w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})
}

func handleListActionChains(db *sql.DB, w http.ResponseWriter, r *http.Request) {
	chains, err := backend.ListActionChains(db)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(chains)
}

func handleCreateActionChain(db *sql.DB, w http.ResponseWriter, r *http.Request) {
	var chain backend.ActionChain
	err := json.NewDecoder(r.Body).Decode(&chain)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = backend.CreateActionChain(db, chain)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func handleGetActionChain(db *sql.DB, w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/action-chains/")
	chain, err := backend.GetActionChain(db, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(chain)
}

func handleUpdateActionChain(db *sql.DB, w http.ResponseWriter, r *http.Request) {
	var chain backend.ActionChain
	err := json.NewDecoder(r.Body).Decode(&chain)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = backend.UpdateActionChain(db, chain)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func handleDeleteActionChain(db *sql.DB, w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/action-chains/")
	err := backend.DeleteActionChain(db, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func main() {
	// Initialize database
	db, err := backend.InitDB("./db/longboy.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Set up API routes
	setupRoutes(db)

	// Serve static files from the src directory
	fs := http.FileServer(http.Dir("./src"))
	http.Handle("/", fs)

	log.Println("Server starting on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}
