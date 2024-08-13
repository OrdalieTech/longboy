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
			handleListActionChains(db, w)
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
			handleGetActionChain(db, w, id)
		case http.MethodPut:
			handleUpdateActionChain(db, w, r)
		case http.MethodDelete:
			handleDeleteActionChain(db, w, id)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// Action routes
	http.HandleFunc("/actions", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			handleCreateAction(db, w, r)
		case http.MethodGet:
			handleListActions(db, w)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	http.HandleFunc("/actions/", func(w http.ResponseWriter, r *http.Request) {
		id := r.URL.Path[len("/actions/"):]
		if id == "" {
			http.Error(w, "ID is required", http.StatusBadRequest)
			return
		}
		switch r.Method {
		case http.MethodGet:
			handleGetAction(db, w, id)
		case http.MethodPut:
			handleUpdateAction(db, w, r)
		case http.MethodDelete:
			handleDeleteAction(db, w, id)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// Trigger routes
	http.HandleFunc("/triggers", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			handleCreateTrigger(db, w, r)
		case http.MethodGet:
			handleListTriggers(db, w)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	http.HandleFunc("/triggers/", func(w http.ResponseWriter, r *http.Request) {
		id := r.URL.Path[len("/triggers/"):]
		if id == "" {
			http.Error(w, "ID is required", http.StatusBadRequest)
			return
		}
		switch r.Method {
		case http.MethodGet:
			handleGetTrigger(db, w, id)
		case http.MethodPut:
			handleUpdateTrigger(db, w, r)
		case http.MethodDelete:
			handleDeleteTrigger(db, w, id)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})
}

// ActionChain Handlers
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

func handleGetActionChain(db *sql.DB, w http.ResponseWriter, id string) {
	chain, err := database.GetActionChain(db, id)
	if err != nil {
		log.Printf("Error getting action chain: %v", err)
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(chain)
}

func handleUpdateActionChain(db *sql.DB, w http.ResponseWriter, r *http.Request) {
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

func handleDeleteActionChain(db *sql.DB, w http.ResponseWriter, id string) {
	err := database.DeleteActionChain(db, id)
	if err != nil {
		log.Printf("Error deleting action chain: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func handleListActionChains(db *sql.DB, w http.ResponseWriter) {
	chains, err := database.ListActionChains(db)
	if err != nil {
		log.Printf("Error listing action chains: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(chains)
}

// Action Handlers
func handleCreateAction(db *sql.DB, w http.ResponseWriter, r *http.Request) {
	var action models.Action
	err := json.NewDecoder(r.Body).Decode(&action)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = database.CreateAction(db, action)
	if err != nil {
		log.Printf("Error creating action: %v", err)
		http.Error(w, fmt.Sprintf("Error creating action: %v", err), http.StatusInternalServerError)
		return
	}

	log.Println("Action created successfully")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(action)
}

func handleGetAction(db *sql.DB, w http.ResponseWriter, id string) {
	action, err := database.GetAction(db, id)
	if err != nil {
		log.Printf("Error getting action: %v", err)
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(action)
}

func handleUpdateAction(db *sql.DB, w http.ResponseWriter, r *http.Request) {
	var action models.Action
	err := json.NewDecoder(r.Body).Decode(&action)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = database.UpdateAction(db, action)
	if err != nil {
		log.Printf("Error updating action: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func handleDeleteAction(db *sql.DB, w http.ResponseWriter, id string) {
	err := database.DeleteAction(db, id)
	if err != nil {
		log.Printf("Error deleting action: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func handleListActions(db *sql.DB, w http.ResponseWriter) {
	actions, err := database.ListActions(db)
	if err != nil {
		log.Printf("Error listing actions: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(actions)
}

// Trigger Handlers
func handleCreateTrigger(db *sql.DB, w http.ResponseWriter, r *http.Request) {
	var trigger models.Trigger
	err := json.NewDecoder(r.Body).Decode(&trigger)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = database.CreateTrigger(db, trigger)
	if err != nil {
		log.Printf("Error creating trigger: %v", err)
		http.Error(w, fmt.Sprintf("Error creating trigger: %v", err), http.StatusInternalServerError)
		return
	}

	log.Println("Trigger created successfully")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(trigger)
}

func handleGetTrigger(db *sql.DB, w http.ResponseWriter, id string) {
	trigger, err := database.GetTrigger(db, id)
	if err != nil {
		log.Printf("Error getting trigger: %v", err)
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(trigger)
}

func handleUpdateTrigger(db *sql.DB, w http.ResponseWriter, r *http.Request) {
	var trigger models.Trigger
	err := json.NewDecoder(r.Body).Decode(&trigger)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = database.UpdateTrigger(db, trigger)
	if err != nil {
		log.Printf("Error updating trigger: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func handleDeleteTrigger(db *sql.DB, w http.ResponseWriter, id string) {
	err := database.DeleteTrigger(db, id)
	if err != nil {
		log.Printf("Error deleting trigger: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func handleListTriggers(db *sql.DB, w http.ResponseWriter) {
	triggers, err := database.ListTriggers(db)
	if err != nil {
		log.Printf("Error listing triggers: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(triggers)
}
