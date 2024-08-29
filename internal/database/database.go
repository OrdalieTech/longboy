package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"sync"

	_ "github.com/mattn/go-sqlite3"

	models "longboy/internal/models"
)

var (
	actionIDCounter int
	counterMutex    sync.Mutex
)

func GetNextActionID() int {
	counterMutex.Lock()
	defer counterMutex.Unlock()
	actionIDCounter++
	return actionIDCounter
}

func InitDB(dbPath string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	// Create action_chains table if it doesn't exist
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS action_chains (
			id TEXT PRIMARY KEY,
			data JSON NOT NULL,
			active BOOLEAN NOT NULL DEFAULT 0
		)
	`)
	if err != nil {
		return nil, err
	}

	// Create actions table if it doesn't exist
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS actions (
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
	log.Printf("Activating action chain with ID: %s", id)

	// Set the action chain as active
	_, err := db.Exec("UPDATE action_chains SET active = 1 WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to activate action chain: %v", err)
	}

	chain, err := GetActionChain(db, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("action chain with ID %s not found", id)
		}
		return fmt.Errorf("failed to retrieve action chain: %v", err)
	}

	// Start the action chain in a new goroutine

	ctx := &models.Context{Results: make(map[string]interface{})}

	// Keep the trigger always active
	log.Printf("Executing trigger: %v", chain.Trigger)
	err = chain.Trigger.Exec(ctx, db)
	if err != nil {
		log.Printf("failed to execute trigger: %v", err)
		// Optionally, handle the error, e.g., retry, backoff, etc.
	}

	return nil
}

// DeactivateActionChain sets the action chain as inactive
func DeactivateActionChain(db *sql.DB, id string) error {
	_, err := db.Exec("UPDATE action_chains SET active = 0 WHERE id = ?", id)
	return err
}

// CreateAction creates a new action in the database
func CreateAction(db *sql.DB, action models.Action) error {
	// actionID := GetNextActionID()
	// action.ID = fmt.Sprintf("action-%d", actionID)

	data, err := models.MarshalAction(action)
	if err != nil {
		return err
	}

	_, err = db.Exec("INSERT INTO actions (id, data) VALUES (?, ?)", action.ID, data)
	return err
}

// GetAction retrieves an action from the database by ID
func GetAction(db *sql.DB, id string) (models.Action, error) {
	var data []byte
	err := db.QueryRow("SELECT data FROM actions WHERE id = ?", id).Scan(&data)
	if err != nil {
		return models.Action{}, err
	}

	var action models.Action
	action, err = models.UnmarshalAction(data)
	return action, err
}

// UpdateAction updates an existing action in the database
func UpdateAction(db *sql.DB, action models.Action, id string) error {
	data, err := models.MarshalAction(action)
	if err != nil {
		return err
	}

	_, err = db.Exec("UPDATE actions SET data = ? WHERE id = ?", data, id)
	return err
}

// DeleteAction removes an action from the database by ID
func DeleteAction(db *sql.DB, id string) error {
	_, err := db.Exec("DELETE FROM actions WHERE id = ?", id)
	return err
}

// ListActions retrieves all actions from the database
func ListActions(db *sql.DB) ([]models.Action, error) {
	rows, err := db.Query("SELECT data FROM actions")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var actions []models.Action
	for rows.Next() {
		var data []byte
		err := rows.Scan(&data)
		if err != nil {
			return nil, err
		}

		action, err := models.UnmarshalAction(data)
		if err != nil {
			return nil, err
		}

		actions = append(actions, action)
	}

	return actions, nil
}
