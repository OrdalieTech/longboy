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
}

type Trigger struct {
	Type            string            `json:"type"`
	URL             string            `json:"url"`
	Method          string            `json:"method"`
	Headers         map[string]string `json:"headers"`
	Body            string            `json:"body"`
	ResultID        string            `json:"result_id,omitempty"`
	FollowingAction *Action           `json:"followingAction"`
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

type Action struct {
	Type            string                 `json:"type"`
	Main            map[string]interface{} `json:"main"`
	ResultID        string                 `json:"result_id,omitempty"`
	FollowingAction *Action                `json:"followingAction"`
}

func (a *Action) Exec(ctx *Context) error {
	var resultKey string
	if a.ResultID != "" {
		resultKey = a.ResultID
	} else {
		resultKey = a.Type + "_result"
	}

	switch a.Type {
	case "http":
		url, ok := a.Main["URL"].(string)
		if !ok {
			return fmt.Errorf("invalid URL in main data")
		}
		method, ok := a.Main["Method"].(string)
		if !ok {
			return fmt.Errorf("invalid Method in main data")
		}
		headers, ok := a.Main["Headers"].(map[string]interface{})
		if !ok {
			return fmt.Errorf("invalid Headers in main data")
		}
		body, ok := a.Main["Body"].(string)
		if !ok {
			return fmt.Errorf("invalid Body in main data")
		}
		client := &http.Client{}
		req, err := http.NewRequest(method, url, bytes.NewBufferString(body))
		if err != nil {
			return err
		}
		for key, value := range headers {
			strValue, ok := value.(string)
			if !ok {
				return fmt.Errorf("invalid header value for key %s", key)
			}
			req.Header.Set(key, strValue)
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
		ctx.Results[resultKey] = string(respBody)
		return nil

	case "llm":
		endpoint, ok := a.Main["Endpoint"].(string)
		if !ok {
			return fmt.Errorf("invalid Endpoint in main data")
		}
		model, ok := a.Main["Model"].(string)
		if !ok {
			return fmt.Errorf("invalid Model in main data")
		}
		prompt, ok := a.Main["Prompt"].(string)
		if !ok {
			return fmt.Errorf("invalid Prompt in main data")
		}
		parameters, ok := a.Main["Parameters"].(map[string]interface{})
		if !ok {
			return fmt.Errorf("invalid Parameters in main data")
		}
		payload := map[string]interface{}{
			"model":      model,
			"prompt":     prompt,
			"parameters": parameters,
		}
		payloadBytes, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		client := &http.Client{}
		req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(payloadBytes))
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
		ctx.Results[resultKey] = string(respBody)
		return nil

	case "code":
		language, ok := a.Main["Language"].(string)
		if !ok {
			return fmt.Errorf("invalid Language in main data")
		}
		sourceCode, ok := a.Main["SourceCode"].(string)
		if !ok {
			return fmt.Errorf("invalid SourceCode in main data")
		}
		var cmd *exec.Cmd
		switch language {
		case "python":
			cmd = exec.Command("python", "-c", sourceCode)
		case "bash":
			cmd = exec.Command("bash", "-c", sourceCode)
		default:
			return fmt.Errorf("unsupported language: %s", language)
		}
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("execution failed: %s, output: %s", err, output)
		}
		ctx.Results[resultKey] = string(output)
		return nil

	default:
		return fmt.Errorf("unsupported action type: %s", a.Type)
	}
}
