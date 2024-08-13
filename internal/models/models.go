package models

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"reflect"
	"strconv"
	"strings"
)

type Context struct {
	Results map[string]interface{}
}

type ActionChain struct {
	ID      string   `json:"id"`
	Trigger *Trigger `json:"trigger"`
	Context *Context `json:"context"`
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
	GetType() string
	GetDescription() string
	Exec(ctx *Context) error
	GetResultID() string
	GetFollowingActionID() string
}

type HTTPAction struct {
	ID                string            `json:"id"`
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

type LLMAction struct {
	ID                string                 `json:"id"`
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
	Description       string `json:"description"`
	Language          string `json:"language"`
	SourceCode        string `json:"source_code"`
	ResultID          string `json:"result_id,omitempty"`
	FollowingActionID string `json:"following_action_id,omitempty"`
}

func (c *CodeAction) GetID() string {
	return c.ID
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
	Description       string `json:"description"`
	Condition         string `json:"condition"`
	TrueActionID      string `json:"true_action_id"`
	FalseActionID     string `json:"false_action_id"`
	FollowingActionID string `json:"following_action_id"`
}

func (b *BranchAction) GetID() string {
	return b.ID
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
