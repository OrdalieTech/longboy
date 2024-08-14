package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

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

func getActionByID(db *sql.DB, id string) (models.Action, error) {
	var data []byte
	err := db.QueryRow("SELECT data FROM actions WHERE id = ?", id).Scan(&data)
	if err != nil {
		return nil, err
	}

	action, err := models.UnmarshalAction(data)
	return action, err
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

	ctx := &models.Context{Results: make(map[string]interface{})}

	err = chain.Trigger.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to execute trigger: %v", err)
	}

	// Execute following actions
	nextActionID := chain.Trigger.FollowingActionID
	for nextActionID != "" {
		nextAction, err := getActionByID(db, nextActionID) // Assume this function retrieves the next action by ID
		if err != nil {
			return err
		}
		err = nextAction.Exec(ctx)
		if err != nil {
			return err
		}
		nextActionID = nextAction.GetFollowingActionID()
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
	actionID := GetNextActionID()
	action.SetID(fmt.Sprintf("action-%d", actionID)) // Assuming SetID method exists

	data, err := models.MarshalAction(action)
	if err != nil {
		return err
	}

	_, err = db.Exec("INSERT INTO actions (id, data) VALUES (?, ?)", action.GetID(), data)
	return err
}

// GetAction retrieves an action from the database by ID
func GetAction(db *sql.DB, id string) (models.Action, error) {
	var data []byte
	err := db.QueryRow("SELECT data FROM actions WHERE id = ?", id).Scan(&data)
	if err != nil {
		return nil, err
	}

	var action models.Action
	action, err = models.UnmarshalAction(data)
	return action, err
}

// UpdateAction updates an existing action in the database
func UpdateAction(db *sql.DB, action models.Action) error {
	data, err := models.MarshalAction(action)
	if err != nil {
		return err
	}

	_, err = db.Exec("UPDATE actions SET data = ? WHERE id = ?", data, action.GetID())
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
			if err := ActivateActionChain(db, id); err != nil {
				log.Printf("Error executing action chain %s: %v", id, err)
			}
		}
	}
}
