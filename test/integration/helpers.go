package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
)

// HTTPClient provides helper methods for making HTTP requests in tests
type HTTPClient struct {
	BaseURL string
	Client  *http.Client
}

// NewHTTPClient creates a new HTTP client for testing
func NewHTTPClient(baseURL string) *HTTPClient {
	return &HTTPClient{
		BaseURL: baseURL,
		Client:  &http.Client{},
	}
}

// Get performs a GET request
func (c *HTTPClient) Get(t *testing.T, path string) (*http.Response, error) {
	t.Helper()
	return c.Client.Get(c.BaseURL + path)
}

// Post performs a POST request with JSON body
func (c *HTTPClient) Post(t *testing.T, path string, body interface{}) (*http.Response, error) {
	t.Helper()

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal body: %w", err)
	}

	return c.Client.Post(
		c.BaseURL+path,
		"application/json",
		bytes.NewBuffer(jsonBody),
	)
}

// GetJSON performs a GET request and decodes the JSON response
func (c *HTTPClient) GetJSON(t *testing.T, path string, target interface{}) (*http.Response, error) {
	t.Helper()

	resp, err := c.Get(t, path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
		return resp, fmt.Errorf("failed to decode JSON: %w", err)
	}

	return resp, nil
}

// PostJSON performs a POST request and decodes the JSON response
func (c *HTTPClient) PostJSON(t *testing.T, path string, body interface{}, target interface{}) (*http.Response, error) {
	t.Helper()

	resp, err := c.Post(t, path, body)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if target != nil {
		if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
			return resp, fmt.Errorf("failed to decode JSON: %w", err)
		}
	}

	return resp, nil
}

// AssertStatus asserts that the response has the expected status code
func AssertStatus(t *testing.T, resp *http.Response, expectedStatus int) {
	t.Helper()

	if resp.StatusCode != expectedStatus {
		bodyBytes, _ := io.ReadAll(resp.Body)
		t.Errorf(
			"expected status %d, got %d\nResponse body: %s",
			expectedStatus,
			resp.StatusCode,
			string(bodyBytes),
		)
	}
}

// AssertJSONField asserts that a JSON response contains a specific field value
func AssertJSONField(t *testing.T, data map[string]interface{}, field string, expected interface{}) {
	t.Helper()

	actual, ok := data[field]
	if !ok {
		t.Errorf("field '%s' not found in response", field)
		return
	}

	if actual != expected {
		t.Errorf("field '%s': expected %v, got %v", field, expected, actual)
	}
}
