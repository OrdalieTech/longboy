package backend

type ActionChain struct {
	ID         string   `json:"id"`
	TriggerIDs []string `json:"trigger-ids"`
}

type Trigger struct {
	ID                string            `json:"id"`
	Type              string            `json:"type"`
	URL               string            `json:"url,omitempty"`
	Method            string            `json:"method,omitempty"`
	Headers           map[string]string `json:"headers,omitempty"`
	Body              string            `json:"body,omitempty"`
	FollowingActionID string            `json:"following-action-id"`
	// Add other trigger types as needed
}

type Action interface {
	GetID() string
	GetType() string
	Exec() error
	GetFollowingActionID() string
}

type HTTPAction struct {
	ID                string            `json:"id"`
	URL               string            `json:"url"`
	Method            string            `json:"method"`
	Headers           map[string]string `json:"headers"`
	Body              string            `json:"body,omitempty"`
	FollowingActionID string            `json:"following-action-id"`
}

func (a HTTPAction) GetID() string {
	return a.ID
}

func (a HTTPAction) GetType() string {
	return "http"
}

func (a HTTPAction) Exec() error {
	// Implementation here
	return nil
}

func (a HTTPAction) GetFollowingActionID() string {
	return a.FollowingActionID
}

type CoreAction struct {
	ID                string `json:"id"`
	Type              string `json:"type"` // loop, bifurcation, etc.
	FollowingActionID string `json:"following-action-id"`
}

func (a CoreAction) GetID() string {
	return a.ID
}

func (a CoreAction) GetType() string {
	return a.Type
}

func (a CoreAction) Exec() error {
	// Implementation here
	return nil
}

func (a CoreAction) GetFollowingActionID() string {
	return a.FollowingActionID
}
