// Package api provides a client for the Confluence REST API.
package api

import (
	"context"
	"net/http"

	"github.com/open-cli-collective/atlassian-go/client"
)

// Client is the Confluence Cloud API client.
type Client struct {
	*client.Client // Embed shared client for HTTP methods
}

// NewClient creates a new Confluence API client.
func NewClient(baseURL, email, apiToken string) *Client {
	return &Client{
		Client: client.New(baseURL, email, apiToken, nil),
	}
}

// Get performs a GET request.
func (c *Client) Get(ctx context.Context, path string) ([]byte, error) {
	return c.Client.Get(ctx, path)
}

// Post performs a POST request.
func (c *Client) Post(ctx context.Context, path string, body interface{}) ([]byte, error) {
	return c.Client.Post(ctx, path, body)
}

// Put performs a PUT request.
func (c *Client) Put(ctx context.Context, path string, body interface{}) ([]byte, error) {
	return c.Client.Put(ctx, path, body)
}

// Delete performs a DELETE request.
func (c *Client) Delete(ctx context.Context, path string) ([]byte, error) {
	return c.Client.Delete(ctx, path)
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
