package models

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"longboy/internal/config"
	"net/http"
	"net/url"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"gorm.io/gorm"
)

type ActionChainContext struct {
	Results map[string]interface{} `json:"results" gorm:"serializer:json"`
}

type Description struct {
	Description            string `json:"description" gorm:"type:text"`
	Author                 string `json:"author" gorm:"type:varchar(100)"`
	CreationDate           string `json:"creation_date" gorm:"type:varchar(20)"`
	LastUpdate             string `json:"last_update" gorm:"type:varchar(20)"`
	Version                string `json:"version" gorm:"type:varchar(20)"`
	Inputs                 string `json:"inputs" gorm:"type:text"`
	Outputs                string `json:"outputs" gorm:"type:text"`
	Dependencies           string `json:"dependencies" gorm:"type:text"`
	UsageExamples          string `json:"usage_examples" gorm:"type:text"`
	ErrorHandling          string `json:"error_handling" gorm:"type:text"`
	RelatedActions         string `json:"related_actions" gorm:"type:text"`
	SecurityConsiderations string `json:"security_considerations" gorm:"type:text"`
	Licensing              string `json:"licensing" gorm:"type:text"`
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
	ID          string              `json:"id" gorm:"primaryKey"`
	Trigger     *Trigger            `json:"trigger" gorm:"serializer:json"`
	Context     *ActionChainContext `json:"context" gorm:"serializer:json"`
	Description *Description        `json:"description" gorm:"serializer:json"`
	Active      bool                `json:"active" gorm:"default:false"`
}

type Trigger struct {
	ID                string            `json:"id" gorm:"type:varchar(100)"`
	Type              string            `json:"type" gorm:"type:varchar(50)"`
	URL               string            `json:"url" gorm:"type:text"`
	Method            string            `json:"method" gorm:"type:varchar(10)"`
	Headers           map[string]string `json:"headers" gorm:"serializer:json"`
	Body              string            `json:"body" gorm:"type:text"`
	ResultID          string            `json:"result_id,omitempty" gorm:"type:varchar(100)"`
	FollowingActionID string            `json:"following_action_id,omitempty" gorm:"type:varchar(100)"`
	Description       *Description      `json:"description" gorm:"serializer:json"`
}

func getActionByID(db *gorm.DB, id string) (Action, error) {
	var action Action
	err := db.First(&action, "id = ?", id).Error
	return action, err
}

func (t *Trigger) Exec(ctx *ActionChainContext, db *gorm.DB) error {
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
				nextAction, err := getActionByID(db, nextActionID)
				if err != nil {
					log.Printf("failed to get next action: %v", err)
					break
				}
				err = nextAction.Exec(ctx)
				if err != nil {
					log.Printf("failed to execute next action: %v", err)
					break
				}
				nextActionID = nextAction.FollowingActionID
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

	go server.ListenAndServe()

	return nil
}

type Placeholder struct {
	Name string       `json:"name" gorm:"type:varchar(100)"`
	Next *Placeholder `json:"next,omitempty" gorm:"serializer:json"`
}

type Action struct {
	ID                string                  `json:"id" gorm:"primaryKey"`
	Type              string                  `json:"type" gorm:"type:varchar(50)"`
	Description       string                  `json:"description" gorm:"type:text"`
	ResultID          string                  `json:"result_id,omitempty" gorm:"type:varchar(100)"`
	FollowingActionID string                  `json:"following_action_id,omitempty" gorm:"type:varchar(100)"`
	Placeholders      map[string]*Placeholder `json:"placeholders" gorm:"serializer:json"`
	Metadata          map[string]interface{}  `json:"metadata" gorm:"serializer:json"`
}

func evaluateCondition(condition string) (bool, error) {
	// Clean up the condition string
	condition = strings.TrimSpace(condition)
	condition = strings.ReplaceAll(condition, "\n", " ")
	condition = strings.Join(strings.Fields(condition), " ")

	// Split the condition into parts
	parts := strings.Split(condition, " ")
	if len(parts) != 3 {
		return false, fmt.Errorf("invalid condition format: %s", condition)
	}

	leftOperand := parts[0]
	operator := parts[1]
	rightOperand := parts[2]

	// Get the left operand value from the context
	var leftValue interface{}
	_, err := strconv.Atoi(leftOperand)
	if err != nil {
		_, err = strconv.ParseFloat(leftOperand, 64)
		if err != nil {
			_, err = strconv.ParseBool(leftOperand)
			if err != nil {
				leftValue = leftOperand
			} else {
				leftValue, _ = strconv.ParseBool(leftOperand)
			}
		} else {
			leftValue, _ = strconv.ParseFloat(leftOperand, 64)
		}
	} else {
		leftValue, _ = strconv.Atoi(leftOperand)
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

func (a *Action) Exec(ctx *ActionChainContext) error {
	switch a.Type {
	case "http":
		return a.ExecHTTP(ctx)
	case "llm":
		return a.ExecLLM(ctx)
	case "code":
		return a.ExecCode(ctx)
	case "if_then":
		return a.ExecIfThen(ctx)
	case "loop":
		return a.ExecLoop(ctx)
	case "branch":
		return a.ExecBranch(ctx)
	default:
		return fmt.Errorf("unknown action type: %s", a.Type)
	}
}

func (a *Action) ProcessBody(ctx *ActionChainContext, body string) (string, error) {
	secretRe := regexp.MustCompile(`{{(.+?)}}`)
	contextRe := regexp.MustCompile(`\[\[(.+?)\]\]`)

	body = secretRe.ReplaceAllStringFunc(body, func(match string) string {
		key := strings.Trim(match, "{}")
		return config.GetConfig().GetSecret(key)
	})

	body = contextRe.ReplaceAllStringFunc(body, func(match string) string {
		expr := strings.Trim(match, "[]")
		placeholder, ok := a.Placeholders[expr]
		if !ok {
			return match // Return original if not found in placeholders
		}
		value := ctx.Results[placeholder.Name]
		current := placeholder.Next
		for current != nil && current.Name != "" {
			if mapValue, ok := value.(map[string]interface{}); ok {
				value, ok = mapValue[current.Name]
				if !ok {
					return match
				}
			} else if sliceValue, ok := value.([]interface{}); ok {
				index, err := strconv.Atoi(current.Name)
				if err != nil || index < 0 || index >= len(sliceValue) {
					return match
				}
				value = sliceValue[index]
			} else {
				return match
			}
			current = current.Next
		}

		// Convert the final value to string
		switch v := value.(type) {
		case string:
			return v
		case []byte:
			return string(v)
		default:
			jsonBytes, err := json.Marshal(v)
			if err != nil {
				log.Printf("Error marshaling context value for expression %s: %v", expr, err)
				return match
			}
			return string(jsonBytes)
		}
	})
	// fmt.Printf("Body before processing: %s\n", body)

	return body, nil
}
