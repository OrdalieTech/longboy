package models

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
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
	Type            string            `json:"type"`
	URL             string            `json:"url"`
	Method          string            `json:"method"`
	Headers         map[string]string `json:"headers"`
	Body            string            `json:"body"`
	ResultID        string            `json:"result_id,omitempty"`
	FollowingAction *Action           `json:"following_action,omitempty"`
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
	GetType() string
	Exec(ctx *Context) error
	GetResultID() string
	GetFollowingAction() *Action
}

type HTTPAction struct {
	URL             string            `json:"url"`
	Method          string            `json:"method"`
	Headers         map[string]string `json:"headers"`
	Body            string            `json:"body"`
	ResultID        string            `json:"result_id,omitempty"`
	FollowingAction *Action           `json:"following_action,omitempty"`
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

func (h *HTTPAction) GetFollowingAction() *Action {
	return h.FollowingAction
}

type LLMAction struct {
	ID              string                 `json:"id"`
	Endpoint        string                 `json:"endpoint"`
	Model           string                 `json:"model"`
	Prompt          string                 `json:"prompt"`
	Parameters      map[string]interface{} `json:"parameters"`
	ResultID        string                 `json:"result_id,omitempty"`
	FollowingAction *Action                `json:"following_action,omitempty"`
}

func (l *LLMAction) GetID() string {
	return l.ID
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

func (l *LLMAction) GetFollowingAction() *Action {
	return l.FollowingAction
}

type CodeAction struct {
	ID              string  `json:"id"`
	Language        string  `json:"language"`
	SourceCode      string  `json:"source_code"`
	ResultID        string  `json:"result_id,omitempty"`
	FollowingAction *Action `json:"following_action,omitempty"`
}

func (c *CodeAction) GetID() string {
	return c.ID
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

func (c *CodeAction) GetFollowingAction() *Action {
	return c.FollowingAction
}
