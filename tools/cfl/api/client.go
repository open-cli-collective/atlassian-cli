// Package api provides a client for the Confluence REST API.
package api //nolint:revive // package name is intentional

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/open-cli-collective/atlassian-go/auth"
	"github.com/open-cli-collective/atlassian-go/client"
)

// Validation errors for bearer auth.
var (
	ErrAPITokenRequired = errors.New("API token is required")
	ErrCloudIDRequired  = errors.New("cloud ID is required for bearer auth")
)

// Client is the Confluence Cloud API client.
// HTTP methods (Get, Post, Put, Delete) are promoted from the embedded *client.Client.
type Client struct {
	*client.Client
}

// NewClient creates a new Confluence API client using basic auth.
func NewClient(baseURL, email, apiToken string) *Client {
	return &Client{
		Client: client.New(baseURL, email, apiToken, nil),
	}
}

// NewBearerClient creates a new Confluence API client using bearer auth via the API gateway.
// The cloudID is used to construct the gateway URL: https://api.atlassian.com/ex/confluence/{cloudId}/wiki
func NewBearerClient(apiToken, cloudID string) (*Client, error) {
	if apiToken == "" {
		return nil, ErrAPITokenRequired
	}
	if cloudID == "" {
		return nil, ErrCloudIDRequired
	}
	gatewayBase := fmt.Sprintf("%s/ex/confluence/%s/wiki", client.GatewayBaseURL, cloudID)
	opts := &client.Options{
		AuthHeader: auth.BearerAuthHeader(apiToken),
	}
	return &Client{
		Client: client.New(gatewayBase, "", "", opts),
	}, nil
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
		return nil, fmt.Errorf("getting current user: %w", err)
	}

	var user User
	if err := json.Unmarshal(body, &user); err != nil {
		return nil, fmt.Errorf("decoding user response: %w", err)
	}

	return &user, nil
}
