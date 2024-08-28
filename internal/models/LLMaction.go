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
	"regexp"
	"strconv"
	"strings"
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
	Temperature float32       `json:"temperature"`
	MaxTokens   int           `json:"max_tokens"`
}

type LLMAction struct {
	BaseAction
	LLMClient
	ChatCompletionRequest
	Provider       string `json:"provider"`
	DeploymentName string `json:"deployment_name"`
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

func (l *LLMAction) Exec(ctx *Context) error {
	fmt.Printf("LLMAction: %+v\n", l)
	l.LLMClient = *NewLLMClient(ClientConfig{
		Provider:       l.Provider,
		DeploymentName: l.DeploymentName,
	})
	fmt.Printf("LLMClient: %+v\n", l.LLMClient)

	for i := range l.ChatCompletionRequest.Messages {
		body := l.ChatCompletionRequest.Messages[i].Content
		secretRe := regexp.MustCompile(`{{(.+?)}}`)
		contextRe := regexp.MustCompile(`\[\[(.+?)\]\]`)

		body = secretRe.ReplaceAllStringFunc(body, func(match string) string {
			key := strings.Trim(match, "{}")
			return config.GetConfig().GetSecret(key)
		})

		body = contextRe.ReplaceAllStringFunc(body, func(match string) string {
			expr := strings.Trim(match, "[]")
			placeholder, ok := l.Placeholders[expr]
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
		l.ChatCompletionRequest.Messages[i].Content = body
		fmt.Printf("Message %d: %s\n", i, l.ChatCompletionRequest.Messages[i].Content)
	}
	fmt.Printf("ChatCompletionRequest: %+v\n", l.ChatCompletionRequest)
	respChan, errChan := l.Completion(context.Background(), l.ChatCompletionRequest)

	select {
	case responseTxt := <-respChan:
		ctx.Results[l.ResultID] = responseTxt
		return nil
	case err := <-errChan:
		log.Printf("Error in Completion: %v", err)
		return err
	}
}

func (l *LLMAction) GetResultID() string {
	return l.ResultID
}

func (l *LLMAction) GetFollowingActionID() string {
	return l.FollowingActionID
}
