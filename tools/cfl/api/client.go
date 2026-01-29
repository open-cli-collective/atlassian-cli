// Package api provides a client for the Confluence REST API.
package api

import (
	"net/http"

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
