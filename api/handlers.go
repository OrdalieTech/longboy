package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"longboy/internal/database"
	"longboy/internal/models"
)

func SetupRoutes(db *sql.DB) {
	http.HandleFunc("/actionchains", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			handleCreateActionChain(db, w, r)
		case http.MethodGet:
			handleListActionChains(db, w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	http.HandleFunc("/actionchains/", func(w http.ResponseWriter, r *http.Request) {
		id := r.URL.Path[len("/actionchains/"):]
		if id == "" {
			http.Error(w, "ID is required", http.StatusBadRequest)
			return
		}
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
}

func handleCreateActionChain(db *sql.DB, w http.ResponseWriter, r *http.Request) {
	var chain models.ActionChain
	err := json.NewDecoder(r.Body).Decode(&chain)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = database.CreateActionChain(db, chain)
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

	chain, err := database.GetActionChain(db, id)
	if err != nil {
		log.Printf("Error getting action chain: %v", err)
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(chain)
}

func handleUpdateActionChain(db *sql.DB, w http.ResponseWriter, r *http.Request, id string) {
	var chain models.ActionChain
	err := json.NewDecoder(r.Body).Decode(&chain)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = database.UpdateActionChain(db, chain)
	if err != nil {
		log.Printf("Error updating action chain: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func handleDeleteActionChain(db *sql.DB, w http.ResponseWriter, r *http.Request, id string) {
	err := database.DeleteActionChain(db, id)
	if err != nil {
		log.Printf("Error deleting action chain: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func handleListActionChains(db *sql.DB, w http.ResponseWriter, r *http.Request) {
	chains, err := database.ListActionChains(db)
	if err != nil {
		log.Printf("Error listing action chains: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(chains)
}
