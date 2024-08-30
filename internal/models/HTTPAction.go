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
	"strings"
)

type HTTPActionData struct {
	URL     string            `json:"url"`
	Method  string            `json:"method"`
	Headers map[string]string `json:"headers"`
	Body    string            `json:"body"`
}

func GetHTTPActionData(a *Action) (*HTTPActionData, error) {
	data := &HTTPActionData{}
	if a.Metadata["url"] != nil {
		data.URL = a.Metadata["url"].(string)
	}
	if a.Metadata["method"] != nil {
		data.Method = a.Metadata["method"].(string)
	}
	if a.Metadata["headers"] != nil {
		// Convert map[string]interface{} to map[string]string
		headersInterface, ok := a.Metadata["headers"].(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("headers is not a map[string]interface{}")
		}
		data.Headers = make(map[string]string)
		for k, v := range headersInterface {
			strValue, ok := v.(string)
			if !ok {
				return nil, fmt.Errorf("header value for key %s is not a string", k)
			}
			data.Headers[k] = strValue
		}
	}
	if a.Metadata["body"] != nil {
		data.Body = a.Metadata["body"].(string)
	}
	return data, nil
}

func HTTPActionDataToMetadata(data *HTTPActionData) map[string]interface{} {
	return map[string]interface{}{
		"url":     data.URL,
		"method":  data.Method,
		"headers": data.Headers,
		"body":    data.Body,
	}
}

func (a *Action) ExecHTTP(ctx *ActionChainContext) error {
	client := &http.Client{}

	h, err := GetHTTPActionData(a)
	if err != nil {
		return err
	}

	// Replace placeholders in the body with actual secret values and context values
	body, err := a.ProcessBody(ctx, h.Body)
	if err != nil {
		return err
	}

	var bodyReader io.Reader
	if h.Headers["Content-Type"] == "application/json" {
		// Parse the body as JSON and re-encode it to ensure it's valid
		var jsonBody map[string]interface{}
		if err := json.Unmarshal([]byte(body), &jsonBody); err != nil {
			// If unmarshaling fails, try to clean up the string
			cleanBody := strings.Replace(body, "\n", "\\n", -1)
			cleanBody = strings.Replace(cleanBody, "\r", "\\r", -1)
			if err := json.Unmarshal([]byte(cleanBody), &jsonBody); err != nil {
				return fmt.Errorf("invalid JSON body: %v", err)
			}
		}
		encodedBody, err := json.Marshal(jsonBody)
		if err != nil {
			return fmt.Errorf("failed to encode JSON body: %v", err)
		}
		bodyReader = bytes.NewBuffer(encodedBody)
	} else {
		bodyReader = bytes.NewBufferString(body)
	}

	req, err := http.NewRequest(h.Method, h.URL, bodyReader)
	if err != nil {
		return err
	}

	secretRe := regexp.MustCompile(`{{(.+?)}}`)

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

	if a.ResultID != "" {
		// Parse the JSON response
		var jsonData interface{}
		err = json.Unmarshal(respBody, &jsonData)
		if err != nil {
			log.Printf("Error parsing JSON response: %v", err)
			// Store the raw response if JSON parsing fails
			ctx.Results[a.ResultID] = string(respBody)
		} else {
			// Store the parsed JSON in ctx.Results
			ctx.Results[a.ResultID] = jsonData
		}

		// fmt.Printf("Stored in ctx.Results[%s]: %+v\n", a.ResultID, ctx.Results[a.ResultID])
	}

	return nil
}

func OpenAPIToHTTPActions(filename string) ([]Action, error) {
	list := []Action{}

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
			headers := make(map[string]string)
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

			// Extract Content-Type from requestBody
			if requestBodyMap, ok := operationMap["requestBody"].(map[string]interface{}); ok {
				if content, ok := requestBodyMap["content"].(map[string]interface{}); ok {
					for contentType := range content {
						headers["Content-Type"] = contentType
						break // Use the first content type found
					}
				}
			}

			// If no Content-Type was found in requestBody, default to application/json
			if _, ok := headers["Content-Type"]; !ok {
				headers["Content-Type"] = "application/json"
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

			id := fmt.Sprintf("%d", utils.GetNextActionID())

			list = append(list, Action{
				ID:                id,
				Type:              "http",
				Description:       description,
				ResultID:          id,
				FollowingActionID: "",
				Metadata: map[string]interface{}{
					"URL":     fullURL,
					"Method":  strings.ToUpper(method),
					"Headers": headers,
					"Body":    requestBody,
				},
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

func LoadAPITemplates(apiDirectory string) (map[string][]Action, error) {
	templates := make(map[string][]Action)

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

func SaveAPITemplates(templates map[string][]Action, templateDir string) error {
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
