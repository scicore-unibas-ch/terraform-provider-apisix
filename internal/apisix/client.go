package apisix

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client is the HTTP client for APISIX Admin API
type Client struct {
	BaseURL    string
	AdminKey   string
	HTTPClient *http.Client
}

// NewClient creates a new APISIX client
func NewClient(baseURL, adminKey string, timeout int) *Client {
	return &Client{
		BaseURL:  baseURL,
		AdminKey: adminKey,
		HTTPClient: &http.Client{
			Timeout: time.Duration(timeout) * time.Second,
		},
	}
}

// Create creates a new resource in APISIX
func (c *Client) Create(ctx context.Context, resourceType, id string, body interface{}) error {
	url := c.buildURL(resourceType, id)
	return c.doRequest(ctx, http.MethodPut, url, body, nil)
}

// Read retrieves a resource from APISIX
func (c *Client) Read(ctx context.Context, resourceType, id string) ([]byte, error) {
	url := c.buildURL(resourceType, id)

	var respBody bytes.Buffer
	err := c.doRequest(ctx, http.MethodGet, url, nil, &respBody)
	if err != nil {
		return nil, err
	}

	return respBody.Bytes(), nil
}

// Update updates an existing resource in APISIX
func (c *Client) Update(ctx context.Context, resourceType, id string, body interface{}) error {
	url := c.buildURL(resourceType, id)
	return c.doRequest(ctx, http.MethodPatch, url, body, nil)
}

// Delete removes a resource from APISIX
func (c *Client) Delete(ctx context.Context, resourceType, id string, force bool) error {
	url := c.buildURL(resourceType, id)
	if force {
		url += "?force=true"
	}
	return c.doRequest(ctx, http.MethodDelete, url, nil, nil)
}

// List retrieves all resources of a given type
func (c *Client) List(ctx context.Context, resourceType string) ([]byte, error) {
	url := c.buildURL(resourceType, "")

	var respBody bytes.Buffer
	err := c.doRequest(ctx, http.MethodGet, url, nil, &respBody)
	if err != nil {
		return nil, err
	}

	return respBody.Bytes(), nil
}

// buildURL constructs the full URL for a resource
func (c *Client) buildURL(resourceType, id string) string {
	if id != "" {
		return fmt.Sprintf("%s/%s/%s", c.BaseURL, resourceType, id)
	}
	return fmt.Sprintf("%s/%s", c.BaseURL, resourceType)
}

// doRequest performs an HTTP request to the APISIX Admin API
func (c *Client) doRequest(ctx context.Context, method, url string, body interface{}, respBody *bytes.Buffer) error {
	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonData)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-KEY", c.AdminKey)

	// Execute request
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	// Check for errors
	if resp.StatusCode >= 400 {
		return c.handleErrorResponse(resp.StatusCode, respData)
	}

	// Copy response body if needed
	if respBody != nil {
		respBody.Write(respData)
	}

	return nil
}

// handleErrorResponse parses and returns APISIX error messages
func (c *Client) handleErrorResponse(statusCode int, data []byte) error {
	var apisixErr APISIXError
	if err := json.Unmarshal(data, &apisixErr); err != nil {
		return fmt.Errorf("HTTP %d: %s", statusCode, string(data))
	}

	// Include status code in error message
	return fmt.Errorf("HTTP %d: %s", statusCode, apisixErr.Error())
}
