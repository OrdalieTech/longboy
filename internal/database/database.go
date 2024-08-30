package database

import (
	"fmt"
	"log"
	"sync"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

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

func InitDB(dbPath string) (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	// Auto Migrate the schema
	err = db.AutoMigrate(&models.ActionChain{}, &models.Action{})
	if err != nil {
		return nil, err
	}

	return db, nil
}

// CreateActionChain inserts a new action chain into the database
func CreateActionChain(db *gorm.DB, chain models.ActionChain) error {
	return db.Create(&chain).Error
}

// GetActionChain retrieves an action chain from the database by ID
func GetActionChain(db *gorm.DB, id string) (models.ActionChain, error) {
	var chain models.ActionChain
	err := db.First(&chain, "id = ?", id).Error
	return chain, err
}

// UpdateActionChain updates an existing action chain in the database
func UpdateActionChain(db *gorm.DB, chain models.ActionChain) error {
	return db.Save(&chain).Error
}

// DeleteActionChain removes an action chain from the database by ID
func DeleteActionChain(db *gorm.DB, id string) error {
	return db.Delete(&models.ActionChain{}, "id = ?", id).Error
}

// ListActionChains retrieves all action chains from the database
func ListActionChains(db *gorm.DB) ([]models.ActionChain, error) {
	var chains []models.ActionChain
	err := db.Find(&chains).Error
	return chains, err
}

func ActivateActionChain(db *gorm.DB, id string) error {
	log.Printf("Activating action chain with ID: %s", id)

	// Set the action chain as active
	err := db.Model(&models.ActionChain{}).Where("id = ?", id).Update("active", true).Error
	if err != nil {
		return fmt.Errorf("failed to activate action chain: %v", err)
	}

	var chain models.ActionChain
	err = db.First(&chain, "id = ?", id).Error
	if err != nil {
		return fmt.Errorf("failed to retrieve action chain: %v", err)
	}
	// Start the action chain in a new goroutine

	ctx := &models.ActionChainContext{Results: make(map[string]interface{})}

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
func DeactivateActionChain(db *gorm.DB, id string) error {
	return db.Model(&models.ActionChain{}).Where("id = ?", id).Update("active", false).Error
}

// CreateAction creates a new action in the database
func CreateAction(db *gorm.DB, action models.Action) error {
	return db.Create(&action).Error
}

// GetAction retrieves an action from the database by ID
func GetAction(db *gorm.DB, id string) (models.Action, error) {
	var action models.Action
	err := db.First(&action, "id = ?", id).Error
	return action, err
}

// UpdateAction updates an existing action in the database
func UpdateAction(db *gorm.DB, action models.Action) error {
	return db.Save(&action).Error
}

// DeleteAction removes an action from the database by ID
func DeleteAction(db *gorm.DB, id string) error {
	return db.Delete(&models.Action{}, "id = ?", id).Error
}

// ListActions retrieves all actions from the database
func ListActions(db *gorm.DB) ([]models.Action, error) {
	var actions []models.Action
	err := db.Find(&actions).Error
	return actions, err
}
