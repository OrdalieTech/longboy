package backend

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	models "longboy/internal/models"
	"net/http"

	_ "github.com/mattn/go-sqlite3"
)

func InitDB(dbPath string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	// Create action_chains table if it doesn't exist
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS action_chains (
			id TEXT PRIMARY KEY,
			data JSON NOT NULL
		)
	`)
	if err != nil {
		return nil, err
	}

	return db, nil
}

// CreateActionChain inserts a new action chain into the database
func CreateActionChain(db *sql.DB, chain models.ActionChain) error {
	data, err := json.Marshal(chain)
	if err != nil {
		return err
	}

	_, err = db.Exec("INSERT INTO action_chains (id, data) VALUES (?, ?)", chain.ID, data)
	return err
}

// GetActionChain retrieves an action chain from the database by ID
func GetActionChain(db *sql.DB, id string) (models.ActionChain, error) {
	var data []byte
	var chain models.ActionChain

	err := db.QueryRow("SELECT data FROM action_chains WHERE id = ?", id).Scan(&data)
	if err != nil {
		return chain, err
	}

	err = json.Unmarshal(data, &chain)
	return chain, err
}

// UpdateActionChain updates an existing action chain in the database
func UpdateActionChain(db *sql.DB, chain models.ActionChain) error {
	data, err := json.Marshal(chain)
	if err != nil {
		return err
	}

	_, err = db.Exec("UPDATE action_chains SET data = ? WHERE id = ?", data, chain.ID)
	return err
}

// DeleteActionChain removes an action chain from the database by ID
func DeleteActionChain(db *sql.DB, id string) error {
	_, err := db.Exec("DELETE FROM action_chains WHERE id = ?", id)
	return err
}

// ListActionChains retrieves all action chains from the database
func ListActionChains(db *sql.DB) ([]models.ActionChain, error) {
	rows, err := db.Query("SELECT data FROM action_chains")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var chains []models.ActionChain
	for rows.Next() {
		var data []byte
		var chain models.ActionChain

		err := rows.Scan(&data)
		if err != nil {
			return nil, err
		}

		err = json.Unmarshal(data, &chain)
		if err != nil {
			return nil, err
		}

		chains = append(chains, chain)
	}

	return chains, nil
}

func ActivateActionChain(db *sql.DB, id string) error {
	// Retrieve the action chain
	chain, err := GetActionChain(db, id)
	if err != nil {
		return fmt.Errorf("failed to retrieve action chain: %v", err)
	}

	// Execute each trigger in the chain
	for _, trigger := range chain.Triggers {
		err := trigger.Exec()
		if err != nil {
			return fmt.Errorf("failed to execute trigger: %v", err)
		}

		// If there's a following action, execute it
		if trigger.FollowingAction != nil {
			err := executeAction(*trigger.FollowingAction)
			if err != nil {
				return fmt.Errorf("failed to execute following action: %v", err)
			}
		}
	}

	return nil
}

func executeAction(action models.Action) error {
	// Create a new HTTP client
	client := &http.Client{}

	// Create a new request
	req, err := http.NewRequest(action.Method, action.URL, bytes.NewBufferString(action.Body))
	if err != nil {
		return err
	}

	// Set headers
	for key, value := range action.Headers {
		req.Header.Set(key, value)
	}

	// Send the request
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// You can add additional logic here to handle the response if needed

	return nil
}
