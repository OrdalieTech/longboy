package backend

type ActionChain struct {
	ID          string   `json:"id"`
	WebhooksIDs []string `json:"webhook-ids"`
	ActionIDs   []string `json:"action-ids"`
}

type HTTPDetails struct {
	ID      string            `json:"id"`
	URL     string            `json:"url"`
	Method  string            `json:"method"`
	Headers map[string]string `json:"headers"`
	Body    string            `json:"body,omitempty"`
}

type Trigger struct {
	ID   string       `json:"id"`
	Type string       `json:"type"`
	HTTP *HTTPDetails `json:"http,omitempty"`
	// Add other trigger types as needed
}

type Action interface {
	GetID() string
	GetType() string
	Execute() error
}

type HTTPAction struct {
	ID   string      `json:"id"`
	Type string      `json:"type"`
	HTTP HTTPDetails `json:"http"`
}

// Implement the Action interface
func (a HTTPAction) GetType() string {
	return a.Type
}

func (a HTTPAction) Execute() error {
	// Implementation here
	return nil
}

// Add other action types as needed
