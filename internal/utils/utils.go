package utils

import (
	"sync"
)

var (
	actionIDCounter int
	counterMutex    sync.Mutex
)

func GetNextActionID() int {
	// Used to generate unique action IDs
	// Whenever the server is restarted, the ID counter is reset
	// TODO: Implement a more sophisticated ID generation strategy (perhaps a random string)
	// Not implemented for now, user enters their own ID
	counterMutex.Lock()
	defer counterMutex.Unlock()
	actionIDCounter++
	return actionIDCounter
}
