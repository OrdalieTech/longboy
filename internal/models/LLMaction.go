package models

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
)

type LLMAction struct {
	BaseAction
	Endpoint   string                 `json:"endpoint"`
	Model      string                 `json:"model"`
	Prompt     string                 `json:"prompt"`
	Parameters map[string]interface{} `json:"parameters"`
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
