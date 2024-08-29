package models

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"longboy/internal/config"
)

type ConvMessage struct {
	Role          string `json:"role"`
	Content       string `json:"content"`
	LinkedContent []struct {
		ID   string `json:"id"`
		Type string `json:"type"`
	} `json:"linkedContent,omitempty"`
	Intent string `json:"intent,omitempty"`
}

type StandardLLMResponse struct {
	Content string `json:"content"`
	Intent  string `json:"intent"`
}

type LLMClient struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
	appName    string
	appURL     string
}

type ClientConfig struct {
	APIKey         string
	BaseURL        string
	HTTPClient     *http.Client
	AppName        string
	AppURL         string
	Provider       string
	ResourceName   string
	DeploymentName string
}

type ChatCompletionRequest struct {
	Models      []string      `json:"models"`
	Messages    []ConvMessage `json:"messages"`
	Stream      bool          `json:"stream"`
	Temperature float64       `json:"temperature"`
	MaxTokens   int           `json:"max_tokens"`
}

type LLMActionData struct {
	LLMClient
	ChatCompletionRequest
	Provider       string `json:"provider"`
	DeploymentName string `json:"deployment_name"`
}

func GetLLMActionData(a *Action) (*LLMActionData, error) {
	data := &LLMActionData{}
	if a.Metadata["apiKey"] != nil {
		data.apiKey = a.Metadata["apiKey"].(string)
	}
	if a.Metadata["baseURL"] != nil {
		data.baseURL = a.Metadata["baseURL"].(string)
	}
	if a.Metadata["httpClient"] != nil {
		data.httpClient = a.Metadata["httpClient"].(*http.Client)
	}
	if a.Metadata["appName"] != nil {
		data.appName = a.Metadata["appName"].(string)
	}
	if a.Metadata["appURL"] != nil {
		data.appURL = a.Metadata["appURL"].(string)
	}
	if a.Metadata["models"] != nil {
		models, ok := a.Metadata["models"].([]interface{})
		if !ok {
			return data, fmt.Errorf("invalid models format in metadata")
		}
		for _, model := range models {
			if modelStr, ok := model.(string); ok {
				data.Models = append(data.Models, modelStr)
			}
		}
	}
	if a.Metadata["messages"] != nil {
		messages, ok := a.Metadata["messages"].([]interface{})
		if !ok {
			return data, fmt.Errorf("invalid messages format in metadata")
		}
		for _, msg := range messages {
			if msgMap, ok := msg.(map[string]interface{}); ok {
				role, _ := msgMap["role"].(string)
				content, _ := msgMap["content"].(string)
				data.Messages = append(data.Messages, ConvMessage{Role: role, Content: content})
			}
		}
	}
	if a.Metadata["stream"] != nil {
		data.Stream = a.Metadata["stream"].(bool)
	}
	if a.Metadata["temperature"] != nil {
		data.Temperature = a.Metadata["temperature"].(float64)
	}
	if a.Metadata["max_tokens"] != nil {
		switch v := a.Metadata["max_tokens"].(type) {
		case int:
			data.MaxTokens = v
		case float64:
			data.MaxTokens = int(v)
		default:
			return data, fmt.Errorf("invalid max_tokens format in metadata")
		}
	}
	if a.Metadata["provider"] != nil {
		data.Provider = a.Metadata["provider"].(string)
	}
	if a.Metadata["deployment_name"] != nil {
		data.DeploymentName = a.Metadata["deployment_name"].(string)
	}
	return data, nil
}

func LLMActionDataToMetadata(data *LLMActionData) map[string]interface{} {
	return map[string]interface{}{
		"apiKey":          data.apiKey,
		"baseURL":         data.baseURL,
		"httpClient":      data.httpClient,
		"appName":         data.appName,
		"appURL":          data.appURL,
		"models":          data.Models,
		"messages":        data.Messages,
		"stream":          data.Stream,
		"temperature":     data.Temperature,
		"max_tokens":      data.MaxTokens,
		"provider":        data.Provider,
		"deployment_name": data.DeploymentName,
	}
}

func NewLLMClient(clientConfig ClientConfig) *LLMClient {
	cfg := config.GetConfig()

	switch clientConfig.Provider {
	case "azure":
		clientConfig.BaseURL = fmt.Sprintf("https://%s/openai/deployments/%s/chat/completions?api-version=2023-12-01-preview", os.Getenv("AZURE_OAI_DOMAIN"), clientConfig.DeploymentName)
		clientConfig.APIKey = cfg.GetSecret("AZURE_API_KEY")
	case "openai":
		clientConfig.BaseURL = "https://api.openai.com/v1/chat/completions"
		clientConfig.APIKey = cfg.GetSecret("OPENAI_API_KEY")
	case "openrouter":
		clientConfig.BaseURL = "https://openrouter.ai/api/v1/chat/completions"
		clientConfig.APIKey = cfg.GetSecret("OPENROUTER_API_KEY")
	default:
		clientConfig.BaseURL = "https://openrouter.ai/api/v1/chat/completions"
		clientConfig.APIKey = cfg.GetSecret("OPENROUTER_API_KEY")
	}

	if clientConfig.HTTPClient == nil {
		clientConfig.HTTPClient = &http.Client{Timeout: 120 * time.Second}
	}

	return &LLMClient{
		apiKey:     clientConfig.APIKey,
		baseURL:    clientConfig.BaseURL,
		httpClient: clientConfig.HTTPClient,
		appName:    clientConfig.AppName,
		appURL:     clientConfig.AppURL,
	}
}

func (c *LLMClient) Completion(ctx context.Context, request ChatCompletionRequest) (<-chan string, <-chan error) {
	responseChan := make(chan string)
	errChan := make(chan error)

	go func() {
		defer close(responseChan)
		defer close(errChan)

		log.Printf("Starting completion attempt with %d models", len(request.Models))
		log.Printf("Base URL: %s", c.baseURL)

		for _, model := range request.Models {
			select {
			case <-ctx.Done():
				errChan <- ctx.Err()
				return
			default:
				log.Printf("Attempting completion with model: %s", model)

				requestBody := map[string]interface{}{
					"model":       model,
					"messages":    request.Messages,
					"stream":      request.Stream,
					"temperature": request.Temperature,
					"max_tokens":  4000,
				}

				jsonBody, err := json.Marshal(requestBody)
				if err != nil {
					log.Printf("Error marshaling request for model %s: %v", model, err)
					errChan <- fmt.Errorf("error marshaling request for model %s: %v", model, err)
					continue
				}

				req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL, bytes.NewReader(jsonBody))
				if err != nil {
					log.Printf("Error creating request for model %s: %v", model, err)
					errChan <- fmt.Errorf("error creating request for model %s: %v", model, err)
					continue
				}

				req.Header.Set("Authorization", "Bearer "+c.apiKey)
				req.Header.Set("api-key", c.apiKey)
				req.Header.Set("Content-Type", "application/json")

				log.Printf("Sending request for model %s", model)
				resp, err := c.httpClient.Do(req)
				if err != nil {
					log.Printf("Error making request for model %s: %v", model, err)
					errChan <- fmt.Errorf("error making request for model %s: %v", model, err)
					continue
				}
				defer resp.Body.Close()

				log.Printf("Received response for model %s with status code: %d", model, resp.StatusCode)

				if resp.StatusCode != http.StatusOK {
					body, _ := io.ReadAll(resp.Body)
					log.Printf("Model %s failed with status code: %d\nResponse body: %s", model, resp.StatusCode, string(body))
					errChan <- fmt.Errorf("model %s failed with status code: %d\nResponse body: %s", model, resp.StatusCode, string(body))
					continue
				}

				log.Printf("Successfully received response for model %s", model)

				if request.Stream {
					c.handleStreamingResponse(resp.Body, responseChan, errChan)
				} else {
					c.handleNonStreamingResponse(resp.Body, responseChan, errChan)
				}
				return
			}
		}

		log.Printf("All models failed")
		errChan <- fmt.Errorf("all models failed")
	}()

	return responseChan, errChan
}

func (c *LLMClient) handleStreamingResponse(body io.ReadCloser, responseChan chan<- string, errChan chan<- error) {
	reader := bufio.NewReader(body)
	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			if err != io.EOF {
				errChan <- fmt.Errorf("error reading stream: %v", err)
			}
			return
		}

		line = bytes.TrimSpace(line)
		if !bytes.HasPrefix(line, []byte("data: ")) {
			continue
		}
		line = bytes.TrimPrefix(line, []byte("data: "))

		if string(line) == "[DONE]" {
			return
		}

		var streamResponse struct {
			Choices []struct {
				Delta struct {
					Content string `json:"content"`
				} `json:"delta"`
			} `json:"choices"`
		}

		if err := json.Unmarshal(line, &streamResponse); err != nil {
			errChan <- err
			return
		}

		if len(streamResponse.Choices) > 0 {
			responseChan <- streamResponse.Choices[0].Delta.Content
		}
	}
}

func (c *LLMClient) handleNonStreamingResponse(body io.ReadCloser, responseChan chan<- string, errChan chan<- error) {
	var response struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.NewDecoder(body).Decode(&response); err != nil {
		errChan <- fmt.Errorf("error decoding response: %v", err)
		return
	}

	if len(response.Choices) > 0 {
		responseChan <- response.Choices[0].Message.Content
	} else {
		errChan <- fmt.Errorf("no content in response")
	}
}

func (a *Action) ExecLLM(ctx *Context) error {
	l, err := GetLLMActionData(a)
	if err != nil {
		return err
	}
	fmt.Printf("LLMAction: %+v\n", l)
	l.LLMClient = *NewLLMClient(ClientConfig{
		Provider:       l.Provider,
		DeploymentName: l.DeploymentName,
	})
	fmt.Printf("LLMClient: %+v\n", l.LLMClient)

	for i := range l.ChatCompletionRequest.Messages {
		body := l.ChatCompletionRequest.Messages[i].Content
		body, err = a.ProcessBody(ctx, body)
		if err != nil {
			return err
		}
		l.ChatCompletionRequest.Messages[i].Content = body
		fmt.Printf("Message %d: %s\n", i, l.ChatCompletionRequest.Messages[i].Content)
	}
	fmt.Printf("ChatCompletionRequest: %+v\n", l.ChatCompletionRequest)
	respChan, errChan := l.Completion(context.Background(), l.ChatCompletionRequest)

	select {
	case responseTxt := <-respChan:
		ctx.Results[a.ResultID] = responseTxt
		return nil
	case err := <-errChan:
		log.Printf("Error in Completion: %v", err)
		return err
	}
}
