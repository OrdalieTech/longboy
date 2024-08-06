package backend

type ActionChain struct {
	ID       string    `json:"id"`
	Triggers []Trigger `json:"triggers"`
}

type Trigger struct {
	Type            string            `json:"type"`
	URL             string            `json:"url"`
	Method          string            `json:"method"`
	Headers         map[string]string `json:"headers"`
	Body            string            `json:"body"`
	FollowingAction Action            `json:"followingAction"`
	// Add other trigger types as needed
}

type Action interface {
	GetType() string
	Exec() error
	GetFollowingAction() Action
}

type HTTPAction struct {
	URL             string            `json:"url"`
	Method          string            `json:"method"`
	Headers         map[string]string `json:"headers"`
	Body            string            `json:"body"`
	FollowingAction Action            `json:"followingAction"`
}

func (a HTTPAction) GetType() string {
	return "http"
}

func (a HTTPAction) Exec() error {
	// Implementation here
	return nil
}

func (a HTTPAction) GetFollowingAction() Action {
	return a.FollowingAction
}

type CoreAction struct {
	Type            string `json:"type"`
	FollowingAction Action `json:"followingAction"`
}

func (a CoreAction) GetType() string {
	return a.Type
}

func (a CoreAction) Exec() error {
	// Implementation here
	return nil
}

func (a CoreAction) GetFollowingAction() Action {
	return a.FollowingAction
}

// Add other action types as needed
