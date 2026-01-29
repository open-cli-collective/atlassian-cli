// Package api provides a client for the Confluence REST API.
package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	defaultTimeout = 30 * time.Second
)

// Client is the Confluence Cloud API client.
type Client struct {
	baseURL    string
	email      string
	apiToken   string
	httpClient *http.Client
}

// NewClient creates a new Confluence API client.
func NewClient(baseURL, email, apiToken string) *Client {
	return &Client{
		baseURL:  strings.TrimSuffix(baseURL, "/"),
		email:    email,
		apiToken: apiToken,
		httpClient: &http.Client{
			Timeout: defaultTimeout,
		},
	}
}

// do executes an HTTP request and returns the response body.
func (c *Client) do(ctx context.Context, method, path string, body interface{}) ([]byte, error) {
	// Ensure path starts with /
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	url := c.baseURL + path

	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.SetBasicAuth(c.email, c.apiToken)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Handle error responses
	if resp.StatusCode >= 400 {
		var errResp ErrorResponse
		if err := json.Unmarshal(respBody, &errResp); err != nil {
			return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(respBody))
		}
		errResp.StatusCode = resp.StatusCode
		return nil, &errResp
	}

	return respBody, nil
}

// Get performs a GET request.
func (c *Client) Get(ctx context.Context, path string) ([]byte, error) {
	return c.do(ctx, http.MethodGet, path, nil)
}

// Post performs a POST request.
func (c *Client) Post(ctx context.Context, path string, body interface{}) ([]byte, error) {
	return c.do(ctx, http.MethodPost, path, body)
}

// Put performs a PUT request.
func (c *Client) Put(ctx context.Context, path string, body interface{}) ([]byte, error) {
	return c.do(ctx, http.MethodPut, path, body)
}

// Delete performs a DELETE request.
func (c *Client) Delete(ctx context.Context, path string) ([]byte, error) {
	return c.do(ctx, http.MethodDelete, path, nil)
}
