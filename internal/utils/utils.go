package utils

import (
	"sync"
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
