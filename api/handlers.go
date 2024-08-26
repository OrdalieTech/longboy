package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"longboy/internal/config"
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
		case http.MethodPost:
			handleActivateActionChain(db, w, id)
		case http.MethodPatch:
			handleDeactivateActionChain(db, w, id)
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
			handleUpdateAction(db, w, r, id)
		case http.MethodDelete:
			handleDeleteAction(db, w, id)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// New route for adding secrets to .env file
	http.HandleFunc("/secrets", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		handleAddSecret(w, r)
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

func handleActivateActionChain(db *sql.DB, w http.ResponseWriter, id string) {
	err := database.ActivateActionChain(db, id)
	if err != nil {
		log.Printf("Error activating action chain: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Action chain activated successfully"))
}

func handleDeactivateActionChain(db *sql.DB, w http.ResponseWriter, id string) {
	err := database.DeactivateActionChain(db, id)
	if err != nil {
		log.Printf("Error deactivating action chain: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Action chain deactivated successfully"))
}

// Action Handlers
func handleCreateAction(db *sql.DB, w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	action, err := models.UnmarshalAction(body)
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

func handleUpdateAction(db *sql.DB, w http.ResponseWriter, r *http.Request, id string) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	action, err := models.UnmarshalAction(body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = database.UpdateAction(db, action, id)
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

// MonitorActiveTriggers continuously checks for active triggers and executes them
func MonitorActiveTriggers(db *sql.DB) {
	ticker := time.NewTicker(1 * time.Minute) // Adjust the interval as needed
	defer ticker.Stop()

	for range ticker.C { // Corrected loop
		// Retrieve all active action chains
		rows, err := db.Query("SELECT id FROM action_chains WHERE active = 1")
		if err != nil {
			log.Printf("Error retrieving active action chains: %v", err)
			continue
		}
		defer rows.Close()

		for rows.Next() {
			var id string
			if err := rows.Scan(&id); err != nil {
				log.Printf("Error scanning action chain ID: %v", err)
				continue
			}

			// Execute the action chain
			if err := database.ActivateActionChain(db, id); err != nil {
				log.Printf("Error executing action chain %s: %v", id, err)
			}
		}
	}
}

// New handler for adding secrets to .env file
func handleAddSecret(w http.ResponseWriter, r *http.Request) {
	var secret struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	}

	err := json.NewDecoder(r.Body).Decode(&secret)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if secret.Key == "" || secret.Value == "" {
		http.Error(w, "Both key and value are required", http.StatusBadRequest)
		return
	}

	err = addSecretToEnvFile(secret.Key, secret.Value)
	if err != nil {
		log.Printf("Error adding secret to .env file: %v", err)
		http.Error(w, "Failed to add secret", http.StatusInternalServerError)
		return
	}

	config.SetSecret(secret.Key, secret.Value)

	w.WriteHeader(http.StatusCreated)
	w.Write([]byte("Secret added successfully"))
}

func addSecretToEnvFile(key, value string) error {
	envFile := ".env"

	// Read existing contents
	content, err := os.ReadFile(envFile)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	lines := strings.Split(string(content), "\n")
	found := false

	// Check for existing key and update if found
	for i, line := range lines {
		if strings.HasPrefix(line, key+"=") {
			lines[i] = fmt.Sprintf("%s=%s", key, value)
			found = true
			break
		}
	}

	// If key not found, append new line
	if !found {
		lines = append(lines, fmt.Sprintf("%s=%s", key, value))
	}

	// Write updated content back to file
	return os.WriteFile(envFile, []byte(strings.Join(lines, "\n")), 0600)
}
