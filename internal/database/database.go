package database

import (
	"database/sql"
	"encoding/json"
	"fmt"

	models "longboy/internal/models"
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

	var action models.Action
	err = json.Unmarshal(data, &action)
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
	chain, err := GetActionChain(db, id)
	if err != nil {
		return fmt.Errorf("failed to retrieve action chain: %v", err)
	}

	ctx := &models.Context{Results: make(map[string]interface{})}

	err = chain.Trigger.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to execute trigger: %v", err)
	}

	if chain.Trigger.FollowingActionID != "" {
		action, err := getActionByID(db, chain.Trigger.FollowingActionID)
		if err != nil {
			return fmt.Errorf("failed to execute following action: %v", err)
		}
		err = action.Exec(ctx)
		if err != nil {
			return fmt.Errorf("failed to execute following action: %v", err)
		}
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

// CreateAction creates a new action in the database
func CreateAction(db *sql.DB, action models.Action) error {
	data, err := json.Marshal(action)
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
	err = json.Unmarshal(data, &action)
	return action, err
}

// UpdateAction updates an existing action in the database
func UpdateAction(db *sql.DB, action models.Action) error {
	data, err := json.Marshal(action)
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
		var action models.Action

		err := rows.Scan(&data)
		if err != nil {
			return nil, err
		}

		err = json.Unmarshal(data, &action)
		if err != nil {
			return nil, err
		}

		actions = append(actions, action)
	}

	return actions, nil
}
