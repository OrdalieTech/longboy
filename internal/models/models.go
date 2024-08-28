package models

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"strings"
)

type Context struct {
	Results map[string]interface{}
}

type Description struct {
	Description            string `json:"description"`
	Author                 string `json:"author"`
	CreationDate           string `json:"creation_date"`
	LastUpdate             string `json:"last_update"`
	Version                string `json:"version"`
	Inputs                 string `json:"inputs"`
	Outputs                string `json:"outputs"`
	Dependencies           string `json:"dependencies"`
	UsageExamples          string `json:"usage_examples"`
	ErrorHandling          string `json:"error_handling"`
	RelatedActions         string `json:"related_actions"`
	SecurityConsiderations string `json:"security_considerations"`
	Licensing              string `json:"licensing"`
}

// Print method to display the Description details neatly
func (d Description) Print() {
	fmt.Println("Description:", d.Description)
	fmt.Println("Author:", d.Author)
	fmt.Println("Creation Date:", d.CreationDate)
	fmt.Println("Last Update:", d.LastUpdate)
	fmt.Println("Version:", d.Version)
	fmt.Println("Inputs:", d.Inputs)
	fmt.Println("Outputs:", d.Outputs)
	fmt.Println("Dependencies:", d.Dependencies)
	fmt.Println("Usage Examples:", d.UsageExamples)
	fmt.Println("Error Handling:", d.ErrorHandling)
	fmt.Println("Related Actions:", d.RelatedActions)
	fmt.Println("Security Considerations:", d.SecurityConsiderations)
	fmt.Println("Licensing:", d.Licensing)
}

type ActionChain struct {
	ID          string       `json:"id"`
	Trigger     *Trigger     `json:"trigger"`
	Context     *Context     `json:"context"`
	Description *Description `json:"description"`
}

type Trigger struct {
	ID                string            `json:"id"`
	Type              string            `json:"type"`
	URL               string            `json:"url"`
	Method            string            `json:"method"`
	Headers           map[string]string `json:"headers"`
	Body              string            `json:"body"`
	ResultID          string            `json:"result_id,omitempty"`
	FollowingActionID string            `json:"following_action_id,omitempty"`
	Description       *Description      `json:"description"`
}

func getActionByID(db *sql.DB, id string) (Action, error) {
	var data []byte
	err := db.QueryRow("SELECT data FROM actions WHERE id = ?", id).Scan(&data)
	if err != nil {
		return nil, err
	}

	action, err := UnmarshalAction(data)
	return action, err
}

func (t *Trigger) Exec(ctx *Context, db *sql.DB) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		parsedURL, _ := url.Parse(t.URL)
		if r.URL.Path != parsedURL.Path {
			http.NotFound(w, r)
			return
		}
		if r.Method != t.Method {
			http.Error(w, fmt.Sprintf("Invalid request method, expected %s", t.Method), http.StatusMethodNotAllowed)
			return
		}
		fmt.Println("Webhook received successfully")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Webhook received successfully"))

		// Read the request body
		body, err := io.ReadAll(r.Body)
		if err != nil {
			log.Printf("Error reading request body: %v", err)
			http.Error(w, "Error reading request body", http.StatusInternalServerError)
			return
		}
		defer r.Body.Close()

		// Parse the JSON
		var jsonData interface{}
		err = json.Unmarshal(body, &jsonData)
		if err != nil {
			log.Printf("Error parsing JSON: %v", err)
			http.Error(w, "Error parsing JSON", http.StatusBadRequest)
			return
		}

		// Store the parsed JSON in ctx.Results
		ctx.Results[t.ResultID] = jsonData

		// fmt.Printf("Stored in ctx.Results[%s]: %+v\n", t.ResultID, jsonData)

		// Trigger the following action
		if t.FollowingActionID != "" {
			// Execute following actions
			nextActionID := t.FollowingActionID
			for nextActionID != "" {
				nextAction, err := getActionByID(db, nextActionID) // Assume this function retrieves the next action by ID
				if err != nil {
					log.Printf("failed to get next action: %v", err)
					break
				}
				err = nextAction.Exec(ctx)
				if err != nil {
					log.Printf("failed to execute next action: %v", err)
					break
				}
				nextActionID = nextAction.GetFollowingActionID()
			}
		}
	})
	fmt.Printf("Listening for webhooks on %s...\n", t.URL)
	// Parse the URL to get the host and port
	parsedURL, err := url.Parse(t.URL)
	if err != nil {
		return fmt.Errorf("failed to parse URL: %v", err)
	}

	// Use the host and port from the parsed URL, or default to ":3000" if not specified
	addr := parsedURL.Host

	server := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	return server.ListenAndServe()
}

type BaseAction struct {
	ID                string `json:"id"`
	Type              string `json:"type"`
	Description       string `json:"description"`
	ResultID          string `json:"result_id,omitempty"`
	FollowingActionID string `json:"following_action_id,omitempty"`
}

type Action interface {
	GetID() string
	SetID(id string)
	GetType() string
	GetDescription() string
	Exec(ctx *Context) error
	GetResultID() string
	GetFollowingActionID() string
}

func UnmarshalAction(data []byte) (Action, error) {
	var baseAction struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(data, &baseAction); err != nil {
		return nil, err
	}

	var action Action
	switch baseAction.Type {
	case "http":
		var httpAction HTTPAction
		if err := json.Unmarshal(data, &httpAction); err != nil {
			return nil, err
		}
		action = &httpAction
	case "llm":
		var llmAction LLMAction
		if err := json.Unmarshal(data, &llmAction); err != nil {
			return nil, err
		}
		action = &llmAction
	case "code":
		var codeAction CodeAction
		if err := json.Unmarshal(data, &codeAction); err != nil {
			return nil, err
		}
		action = &codeAction
	case "branch":
		var branchAction BranchAction
		if err := json.Unmarshal(data, &branchAction); err != nil {
			return nil, err
		}
		action = &branchAction
	case "loop":
		var loopAction LoopAction
		if err := json.Unmarshal(data, &loopAction); err != nil {
			return nil, err
		}
		action = &loopAction
	default:
		return nil, fmt.Errorf("unknown action type: %s", baseAction.Type)
	}

	return action, nil
}

func MarshalAction(action Action) ([]byte, error) {
	switch action.GetType() {
	case "http":
		return json.Marshal(action)
	case "llm":
		return json.Marshal(action)
	case "code":
		return json.Marshal(action)
	case "branch":
		return json.Marshal(action)
	case "loop":
		return json.Marshal(action)
	default:
		return nil, fmt.Errorf("unknown action type: %s", action.GetType())
	}
}

func evaluateCondition(condition string, ctx *Context) (bool, error) {
	// Split the condition into parts
	parts := strings.Split(condition, " ")
	if len(parts) != 3 {
		return false, fmt.Errorf("invalid condition format: %s", condition)
	}

	leftOperand := parts[0]
	operator := parts[1]
	rightOperand := parts[2]

	// Get the left operand value from the context
	leftValue, ok := ctx.Results[leftOperand]
	if !ok {
		return false, fmt.Errorf("left operand '%s' not found in context", leftOperand)
	}

	// Convert right operand to appropriate type
	var rightValue interface{}
	switch leftValue.(type) {
	case int:
		rightValue, _ = strconv.Atoi(rightOperand)
	case float64:
		rightValue, _ = strconv.ParseFloat(rightOperand, 64)
	case string:
		rightValue = rightOperand
	case bool:
		rightValue, _ = strconv.ParseBool(rightOperand)
	default:
		return false, fmt.Errorf("unsupported type for left operand: %T", leftValue)
	}

	// Evaluate the condition
	switch operator {
	case "==":
		return reflect.DeepEqual(leftValue, rightValue), nil
	case "!=":
		return !reflect.DeepEqual(leftValue, rightValue), nil
	case ">":
		return compareValues(leftValue, rightValue, operator)
	case "<":
		return compareValues(leftValue, rightValue, operator)
	case ">=":
		return compareValues(leftValue, rightValue, operator)
	case "<=":
		return compareValues(leftValue, rightValue, operator)
	default:
		return false, fmt.Errorf("unsupported operator: %s", operator)
	}
}

func compareValues(left, right interface{}, operator string) (bool, error) {
	switch l := left.(type) {
	case int:
		r, ok := right.(int)
		if !ok {
			return false, fmt.Errorf("type mismatch: %T and %T", left, right)
		}
		switch operator {
		case ">":
			return l > r, nil
		case "<":
			return l < r, nil
		case ">=":
			return l >= r, nil
		case "<=":
			return l <= r, nil
		}
	case float64:
		r, ok := right.(float64)
		if !ok {
			return false, fmt.Errorf("type mismatch: %T and %T", left, right)
		}
		switch operator {
		case ">":
			return l > r, nil
		case "<":
			return l < r, nil
		case ">=":
			return l >= r, nil
		case "<=":
			return l <= r, nil
		}
	case string:
		r, ok := right.(string)
		if !ok {
			return false, fmt.Errorf("type mismatch: %T and %T", left, right)
		}
		switch operator {
		case ">":
			return l > r, nil
		case "<":
			return l < r, nil
		case ">=":
			return l >= r, nil
		case "<=":
			return l <= r, nil
		}
	default:
		return false, fmt.Errorf("unsupported type for comparison: %T", left)
	}
	return false, fmt.Errorf("invalid operator for type: %s", operator)
}
