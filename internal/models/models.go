package models

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"longboy/internal/utils"
	"net/http"
	"os"
	"os/exec"
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

func (t Trigger) Exec(ctx *Context) error {
	client := &http.Client{}
	req, err := http.NewRequest(t.Method, t.URL, bytes.NewBufferString(t.Body))
	if err != nil {
		return err
	}
	for key, value := range t.Headers {
		req.Header.Set(key, value)
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	// Update context with the response body if ResultID is provided
	if t.ResultID != "" {
		ctx.Results[t.ResultID] = string(respBody)
	}

	return nil
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

type HTTPAction struct {
	ID                string            `json:"id"`
	Type              string            `json:"type"`
	Description       string            `json:"description"`
	URL               string            `json:"url"`
	Method            string            `json:"method"`
	Headers           map[string]string `json:"headers"`
	Body              string            `json:"body"`
	ResultID          string            `json:"result_id,omitempty"`
	FollowingActionID string            `json:"following_action_id,omitempty"`
}

func (h *HTTPAction) GetID() string {
	return h.ID
}

func (h *HTTPAction) SetID(id string) {
	h.ID = id
	h.ResultID = id
}

func (h *HTTPAction) GetDescription() string {
	return h.Description
}

func (h *HTTPAction) GetType() string {
	return "http"
}

func (h *HTTPAction) Exec(ctx *Context) error {
	client := &http.Client{}
	req, err := http.NewRequest(h.Method, h.URL, bytes.NewBufferString(h.Body))
	if err != nil {
		return err
	}
	for key, value := range h.Headers {
		req.Header.Set(key, value)
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if h.ResultID != "" {
		ctx.Results[h.ResultID] = string(respBody)
	}

	return nil
}

func (h *HTTPAction) GetResultID() string {
	return h.ResultID
}

func (h *HTTPAction) GetFollowingActionID() string {
	return h.FollowingActionID
}

func OpenAPIToHTTPActions(filename string) ([]HTTPAction, error) {
	list := []HTTPAction{}

	// Open the JSON file
	file, err := os.Open(filename)
	if err != nil {
		log.Fatalf("Failed to open file: %v", err)
	}
	defer file.Close()

	// Read the file's content
	data, err := io.ReadAll(file)
	if err != nil {
		log.Fatalf("Failed to read file: %v", err)
	}

	// Parse the JSON data
	var apiSpec map[string]interface{}
	if err := json.Unmarshal(data, &apiSpec); err != nil {
		log.Fatalf("Failed to parse JSON: %v", err)
	}

	// Extract servers
	var baseURL string
	if servers, ok := apiSpec["servers"].([]interface{}); ok && len(servers) > 0 {
		if server, ok := servers[0].(map[string]interface{}); ok {
			baseURL, _ = server["url"].(string)
		}
	}

	// Extract paths
	paths, ok := apiSpec["paths"].(map[string]interface{})
	if !ok {
		log.Fatalf("Failed to extract paths")
	}

	// Iterate over each path
	for path, pathItem := range paths {
		pathItemMap, ok := pathItem.(map[string]interface{})
		if !ok {
			continue
		}

		// Iterate over each method
		for method, operation := range pathItemMap {
			operationMap, ok := operation.(map[string]interface{})
			if !ok {
				continue
			}

			// Extract description
			description, _ := operationMap["description"].(string)

			// Extract headers
			headers := []string{}
			if parameters, ok := operationMap["parameters"].([]interface{}); ok {
				for _, param := range parameters {
					paramMap, ok := param.(map[string]interface{})
					if !ok {
						continue
					}
					if paramMap["in"] == "header" {
						if name, ok := paramMap["name"].(string); ok {
							headers = append(headers, name)
						}
					}
				}
			}

			// Check for security schemes in components
			if components, ok := apiSpec["components"].(map[string]interface{}); ok {
				if securitySchemes, ok := components["securitySchemes"].(map[string]interface{}); ok {
					for _, scheme := range securitySchemes {
						schemeMap, ok := scheme.(map[string]interface{})
						if !ok {
							continue
						}
						if schemeMap["type"] == "apiKey" && schemeMap["in"] == "header" {
							if name, ok := schemeMap["name"].(string); ok {
								headers = append(headers, name)
							}
						}
					}
				}
			}

			// Extract request body
			var requestBody string
			if requestBodyMap, ok := operationMap["requestBody"].(map[string]interface{}); ok {
				if content, ok := requestBodyMap["content"].(map[string]interface{}); ok {
					for _, mediaType := range content {
						mediaTypeMap, ok := mediaType.(map[string]interface{})
						if !ok {
							continue
						}
						if schema, ok := mediaTypeMap["schema"].(map[string]interface{}); ok {
							requestBodyBytes, err := json.Marshal(schema)
							if err == nil {
								requestBody = string(requestBodyBytes)
							}
						}
					}
				}
			}

			fullURL := baseURL + path

			headersMap := make(map[string]string)
			for _, header := range headers {
				headersMap[header] = ""
			}

			id := fmt.Sprintf("%d", utils.GetNextActionID())

			//ID and ResultID fields should be filled with a random non used ID,
			//description could be filled by the user, this could be handled in the UI
			list = append(list, HTTPAction{
				ID:                id,
				Type:              "http",
				Description:       description,
				URL:               fullURL,
				Method:            strings.ToUpper(method),
				Headers:           headersMap,
				Body:              requestBody,
				ResultID:          id,
				FollowingActionID: "",
			})
		}
	}
	return list, nil
}

type LLMAction struct {
	ID                string                 `json:"id"`
	Type              string                 `json:"type"`
	Description       string                 `json:"description"`
	Endpoint          string                 `json:"endpoint"`
	Model             string                 `json:"model"`
	Prompt            string                 `json:"prompt"`
	Parameters        map[string]interface{} `json:"parameters"`
	ResultID          string                 `json:"result_id,omitempty"`
	FollowingActionID string                 `json:"following_action_id,omitempty"`
}

func (l *LLMAction) GetID() string {
	return l.ID
}

func (l *LLMAction) SetID(id string) {
	l.ID = id
	l.ResultID = id
}

func (l *LLMAction) GetDescription() string {
	return l.Description
}

func (l *LLMAction) GetType() string {
	return "llm"
}

func (l *LLMAction) Exec(ctx *Context) error {
	payload := map[string]interface{}{
		"model":      l.Model,
		"prompt":     l.Prompt,
		"parameters": l.Parameters,
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	client := &http.Client{}
	req, err := http.NewRequest("POST", l.Endpoint, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if l.ResultID != "" {
		ctx.Results[l.ResultID] = string(respBody)
	}

	return nil
}

func (l *LLMAction) GetResultID() string {
	return l.ResultID
}

func (l *LLMAction) GetFollowingActionID() string {
	return l.FollowingActionID
}

type CodeAction struct {
	ID                string `json:"id"`
	Type              string `json:"type"`
	Description       string `json:"description"`
	Language          string `json:"language"`
	SourceCode        string `json:"source_code"`
	ResultID          string `json:"result_id,omitempty"`
	FollowingActionID string `json:"following_action_id,omitempty"`
}

func (c *CodeAction) GetID() string {
	return c.ID
}

func (c *CodeAction) SetID(id string) {
	c.ID = id
	c.ResultID = id
}

func (c *CodeAction) GetDescription() string {
	return c.Description
}

func (c *CodeAction) GetType() string {
	return "code"
}

func (c *CodeAction) Exec(ctx *Context) error {
	var cmd *exec.Cmd
	switch c.Language {
	case "python":
		cmd = exec.Command("python", "-c", c.SourceCode)
	case "bash":
		cmd = exec.Command("bash", "-c", c.SourceCode)
	default:
		return fmt.Errorf("unsupported language: %s", c.Language)
	}
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("execution failed: %s, output: %s", err, output)
	}

	if c.ResultID != "" {
		ctx.Results[c.ResultID] = string(output)
	}

	return nil
}

func (c *CodeAction) GetResultID() string {
	return c.ResultID
}

func (c *CodeAction) GetFollowingActionID() string {
	return c.FollowingActionID
}

type BranchAction struct {
	ID                string `json:"id"`
	Type              string `json:"type"`
	Description       string `json:"description"`
	Condition         string `json:"condition"`
	TrueActionID      string `json:"true_action_id"`
	FalseActionID     string `json:"false_action_id"`
	FollowingActionID string `json:"following_action_id"`
}

func (b *BranchAction) GetID() string {
	return b.ID
}

func (b *BranchAction) SetID(id string) {
	b.ID = id
}

func (b *BranchAction) GetDescription() string {
	return b.Description
}

func (b *BranchAction) GetType() string {
	return "branch"
}

func (b *BranchAction) Exec(ctx *Context) error {
	// Evaluate the condition
	result, err := evaluateCondition(b.Condition, ctx)
	if err != nil {
		return fmt.Errorf("error evaluating condition: %v", err)
	}

	// Set the FollowingActionID based on the condition result
	if result {
		b.FollowingActionID = b.TrueActionID
	} else {
		b.FollowingActionID = b.FalseActionID
	}

	return nil
}

func (b *BranchAction) GetResultID() string {
	return ""
}

func (b *BranchAction) GetFollowingActionID() string {
	return b.FollowingActionID
}

type LoopAction struct {
	ID                string `json:"id"`
	Type              string `json:"type"`
	Description       string `json:"description"`
	Action            Action `json:"action"`
	Condition         string `json:"condition"`
	FollowingActionID string `json:"following_action_id"`
}

func (l *LoopAction) GetID() string {
	return l.ID
}

func (l *LoopAction) SetID(id string) {
	l.ID = id
}

func (l *LoopAction) GetDescription() string {
	return l.Description
}

func (l *LoopAction) GetType() string {
	return "loop"
}

func (l *LoopAction) Exec(ctx *Context) error {
	for {
		// Execute the action
		if err := l.Action.Exec(ctx); err != nil {
			return fmt.Errorf("error executing action: %v", err)
		}

		// Evaluate the condition
		conditionMet, err := evaluateCondition(l.Condition, ctx)
		if err != nil {
			return fmt.Errorf("error evaluating condition: %v", err)
		}
		if conditionMet {
			break
		}
	}

	return nil
}

func (l *LoopAction) GetResultID() string {
	return ""
}

func (l *LoopAction) GetFollowingActionID() string {
	return l.FollowingActionID
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
