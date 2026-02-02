// Package api provides a client for the Confluence REST API.
package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/open-cli-collective/atlassian-go/client"
)

// Client is the Confluence Cloud API client.
// HTTP methods (Get, Post, Put, Delete) are promoted from the embedded *client.Client.
type Client struct {
	*client.Client
}

// NewClient creates a new Confluence API client.
func NewClient(baseURL, email, apiToken string) *Client {
	return &Client{
		Client: client.New(baseURL, email, apiToken, nil),
	}
}

// GetHTTPClient returns the underlying HTTP client for custom requests.
func (c *Client) GetHTTPClient() *http.Client {
	return c.HTTPClient
}

// GetBaseURL returns the base URL.
func (c *Client) GetBaseURL() string {
	return c.BaseURL
}

// GetAuthHeader returns the authorization header value.
func (c *Client) GetAuthHeader() string {
	return c.AuthHeader
}

// GetCurrentUser returns the currently authenticated user.
// Uses the legacy REST API endpoint /rest/api/user/current.
func (c *Client) GetCurrentUser(ctx context.Context) (*User, error) {
	// The base URL includes /wiki suffix for Confluence Cloud
	// The legacy API endpoint is at /wiki/rest/api/user/current
	// Strip /wiki suffix to avoid duplication, then add it back with the endpoint
	baseURL := strings.TrimSuffix(c.BaseURL, "/wiki")
	url := baseURL + "/wiki/rest/api/user/current"

	body, err := c.Get(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("failed to get current user: %w", err)
	}

	var user User
	if err := json.Unmarshal(body, &user); err != nil {
		return nil, fmt.Errorf("failed to decode user response: %w", err)
	}

	return &user, nil
}
