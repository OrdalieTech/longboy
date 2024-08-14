package models

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"longboy/internal/utils"
)

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
