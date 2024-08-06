package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"longboy/backend"
	"net/http"
	"strings"
)

func setupRoutes(db *sql.DB) {
	http.HandleFunc("/api/action-chains", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Received %s request to /api/action-chains", r.Method)
		switch r.Method {
		case http.MethodGet:
			handleListActionChains(db, w)
		case http.MethodPost:
			handleCreateActionChain(db, w, r)
		default:
			log.Printf("Method %s not allowed for /api/action-chains", r.Method)
			http.Error(w, fmt.Sprintf("Method %s not allowed", r.Method), http.StatusMethodNotAllowed)
		}
	})

	http.HandleFunc("/api/action-chains/", func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimPrefix(r.URL.Path, "/api/action-chains/")
		switch r.Method {
		case http.MethodGet:
			handleGetActionChain(db, w, r, id)
		case http.MethodPut:
			handleUpdateActionChain(db, w, r, id)
		case http.MethodDelete:
			handleDeleteActionChain(db, w, r, id)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	http.HandleFunc("/api/action-chains/activate/", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Received %s request to %s", r.Method, r.URL.Path)
		if r.Method != http.MethodPost {
			log.Printf("Method %s not allowed for /api/action-chains/activate/", r.Method)
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		id := strings.TrimPrefix(r.URL.Path, "/api/action-chains/activate/")
		handleActivateActionChain(db, w, r, id)
	})
}

func handleListActionChains(db *sql.DB, w http.ResponseWriter) {
	chains, err := backend.ListActionChains(db)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(chains)
}

func handleCreateActionChain(db *sql.DB, w http.ResponseWriter, r *http.Request) {
	log.Println("Handling create action chain request")
	var chain backend.ActionChain
	err := json.NewDecoder(r.Body).Decode(&chain)
	if err != nil {
		log.Printf("Error decoding request body: %v", err)
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	err = backend.CreateActionChain(db, chain)
	if err != nil {
		log.Printf("Error creating action chain: %v", err)
		http.Error(w, fmt.Sprintf("Error creating action chain: %v", err), http.StatusInternalServerError)
		return
	}

	log.Println("Action chain created successfully")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(chain)
}

func handleGetActionChain(db *sql.DB, w http.ResponseWriter, r *http.Request, id string) {
	chain, err := backend.GetActionChain(db, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(chain)
}

func handleUpdateActionChain(db *sql.DB, w http.ResponseWriter, r *http.Request, id string) {
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

func handleDeleteActionChain(db *sql.DB, w http.ResponseWriter, r *http.Request, id string) {
	err := backend.DeleteActionChain(db, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func handleActivateActionChain(db *sql.DB, w http.ResponseWriter, r *http.Request, id string) {
	err := backend.ActivateActionChain(db, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
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
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	// Catch-all route for debugging
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Received %s request to %s", r.Method, r.URL.Path)
		http.Error(w, "Not found", http.StatusNotFound)
	})

	log.Println("Server starting on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}
