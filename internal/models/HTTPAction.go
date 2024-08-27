package models

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"longboy/internal/config"
	"longboy/internal/utils"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

type HTTPAction struct {
	BaseAction
	URL     string            `json:"url"`
	Method  string            `json:"method"`
	Headers map[string]string `json:"headers"`
	Body    string            `json:"body"`
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

	// Replace placeholders in the body with actual secret values and context values
	body := h.Body
	secretRe := regexp.MustCompile(`{{(.+?)}}`)
	contextRe := regexp.MustCompile(`\[\[(.+?)\]\]`)

	body = secretRe.ReplaceAllStringFunc(body, func(match string) string {
		key := strings.Trim(match, "{}")
		return config.GetConfig().GetSecret(key)
	})

	body = contextRe.ReplaceAllStringFunc(body, func(match string) string {
		expr := strings.Trim(match, "[]")
		parts := strings.Split(expr, ".")
		var value interface{}
		var ok bool

		// Get the initial value
		value, ok = ctx.Results[parts[0]]
		if !ok {
			return match // Return original if not found in context
		}

		// Navigate through the parts
		for i, part := range parts[1:] {
			switch v := value.(type) {
			case map[string]interface{}:
				value, ok = v[part]
				if !ok {
					return match
				}
			case []interface{}:
				// Check if the part is a number for array indexing
				index, err := strconv.Atoi(part)
				if err != nil || index < 0 || index >= len(v) {
					return match
				}
				value = v[index]
			default:
				// If we can't navigate further but there are more parts, return original
				if i < len(parts)-2 {
					return match
				}
			}
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

	req, err := http.NewRequest(h.Method, h.URL, bytes.NewBufferString(body))
	if err != nil {
		return err
	}

	// Replace placeholders in headers with actual secret values
	for key, value := range h.Headers {
		headerValue := secretRe.ReplaceAllStringFunc(value, func(match string) string {
			secretKey := strings.Trim(match, "{}")
			return config.GetConfig().GetSecret(secretKey)
		})
		req.Header.Set(key, headerValue)
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

	// Check for BOM and remove it if present
	data = bytes.TrimPrefix(data, []byte("\xef\xbb\xbf"))

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
			description, ok := operationMap["description"].(string)
			if !ok {
				log.Printf("Warning: No description found for %s %s", method, path)
				description = ""
			}

			// Extract headers
			headers := map[string]string{
				"Content-Type": "application/json",
			}
			if parameters, ok := operationMap["parameters"].([]interface{}); ok {
				for _, param := range parameters {
					paramMap, ok := param.(map[string]interface{})
					if !ok {
						continue
					}
					if paramMap["in"] == "header" {
						if name, ok := paramMap["name"].(string); ok {
							headers[name] = ""
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
								headers[name] = ""
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
							resolvedSchema := resolveRef(schema, apiSpec)
							requestBodyMap := extractProperties(resolvedSchema)
							requestBodyBytes, err := json.MarshalIndent(requestBodyMap, "", "  ")
							if err != nil {
								log.Fatalf("Failed to marshal request body: %v", err)
							}
							requestBody = string(requestBodyBytes)
							//requestBody = strings.ReplaceAll(string(requestBodyBytes), `"`, `"{{`)
							//requestBody = strings.ReplaceAll(requestBody, `"`, `}}"`)
							// Replace escaped newline characters with actual newlines
							requestBody = strings.ReplaceAll(requestBody, "\\n", "\n")
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

			list = append(list, HTTPAction{
				BaseAction: BaseAction{
					ID:                id,
					Type:              "http",
					Description:       description,
					ResultID:          id,
					FollowingActionID: "",
				},
				URL:     fullURL,
				Method:  strings.ToUpper(method),
				Headers: headersMap,
				Body:    requestBody,
			})
		}
	}
	return list, nil
}

// resolveRef resolves a $ref field in the schema and replaces it with the actual definition
func resolveRef(schema map[string]interface{}, apiSpec map[string]interface{}) map[string]interface{} {
	if ref, ok := schema["$ref"].(string); ok {
		// Remove the initial '#/' from the reference
		ref = strings.TrimPrefix(ref, "#/")
		// Split the reference by '/'
		parts := strings.Split(ref, "/")
		// Traverse the apiSpec to find the referenced schema
		var result interface{} = apiSpec
		for _, part := range parts {
			if m, ok := result.(map[string]interface{}); ok {
				result = m[part]
			} else {
				return map[string]interface{}{} // Return empty object if the reference is invalid
			}
		}
		// Recursively resolve any nested $ref fields
		if resolvedSchema, ok := result.(map[string]interface{}); ok {
			return resolveRef(resolvedSchema, apiSpec)
		}
	}
	// Recursively resolve any nested $ref fields in the original schema
	for key, value := range schema {
		if subSchema, ok := value.(map[string]interface{}); ok {
			schema[key] = resolveRef(subSchema, apiSpec)
		}
	}
	return schema
}

// extractProperties extracts the properties from the schema and creates a map with empty values
func extractProperties(schema map[string]interface{}) map[string]interface{} {
	properties := make(map[string]interface{})
	if props, ok := schema["properties"].(map[string]interface{}); ok {
		for key := range props {
			properties[key] = ""
		}
	}
	return properties
}

func LoadAPITemplates(apiDirectory string) (map[string][]HTTPAction, error) {
	templates := make(map[string][]HTTPAction)

	files, err := os.ReadDir(apiDirectory)
	if err != nil {
		return nil, fmt.Errorf("failed to read API directory: %v", err)
	}

	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".json") {
			filePath := filepath.Join(apiDirectory, file.Name())
			actions, err := OpenAPIToHTTPActions(filePath)
			if err != nil {
				log.Printf("Error processing %s: %v", file.Name(), err)
				continue
			}

			templateName := strings.TrimSuffix(file.Name(), ".json")
			templates[templateName] = actions
		}
	}

	return templates, nil
}

func SaveAPITemplates(templates map[string][]HTTPAction, templateDir string) error {
	if err := os.MkdirAll(templateDir, 0755); err != nil {
		return fmt.Errorf("failed to create template directory: %v", err)
	}

	for name, actions := range templates {
		filePath := filepath.Join(templateDir, name+".json")
		file, err := os.Create(filePath)
		if err != nil {
			return fmt.Errorf("failed to create file %s: %v", filePath, err)
		}
		defer file.Close()

		encoder := json.NewEncoder(file)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(actions); err != nil {
			return fmt.Errorf("failed to encode template %s: %v", name, err)
		}
	}

	return nil
}
