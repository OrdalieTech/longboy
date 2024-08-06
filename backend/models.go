package backend

import (
	"bytes"
	"io"
	"net/http"
)

type ActionChain struct {
	ID       string    `json:"id"`
	Triggers []Trigger `json:"triggers"`
}

type Trigger struct {
	Type            string            `json:"type"`
	URL             string            `json:"url"`
	Method          string            `json:"method"`
	Headers         map[string]string `json:"headers"`
	Body            string            `json:"body"`
	FollowingAction *Action           `json:"followingAction"`
	// Add other trigger types as needed
}

func (t Trigger) Exec() error {
	// Create a new HTTP client
	client := &http.Client{}

	// Create a new request
	req, err := http.NewRequest(t.Method, t.URL, bytes.NewBufferString(t.Body))
	if err != nil {
		return err
	}

	// Set headers
	for key, value := range t.Headers {
		req.Header.Set(key, value)
	}

	// Send the request
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Read the response body (optional, depending on your needs)
	_, err = io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	// You can add additional logic here to handle the response if needed

	return nil
}

type HTTPAction struct {
	URL             string            `json:"url"`
	Method          string            `json:"method"`
	Headers         map[string]string `json:"headers"`
	Body            string            `json:"body"`
	FollowingAction Action            `json:"followingAction"`
}

func (a HTTPAction) GetType() string {
	return "http"
}

func (a HTTPAction) Exec() error {
	// Create a new HTTP client
	client := &http.Client{}

	// Create a new request
	req, err := http.NewRequest(a.Method, a.URL, bytes.NewBufferString(a.Body))
	if err != nil {
		return err
	}

	// Set headers
	for key, value := range a.Headers {
		req.Header.Set(key, value)
	}

	// Send the request
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Read the response body (optional, depending on your needs)
	_, err = io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	// You can add additional logic here to handle the response if needed

	return nil
}

type Action struct {
	Type    string            `json:"type"`
	URL     string            `json:"url"`
	Method  string            `json:"method"`
	Headers map[string]string `json:"headers"`
	Body    string            `json:"body"`
}
