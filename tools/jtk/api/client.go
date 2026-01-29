package api

import (
	"context"
	"fmt"
	"net/http"
	neturl "net/url"

	"github.com/open-cli-collective/atlassian-go/client"
	"github.com/open-cli-collective/atlassian-go/errors"
	"github.com/open-cli-collective/atlassian-go/url"
)

// Client is a Jira API client
type Client struct {
	*client.Client        // Embed shared client for HTTP methods
	URL            string // Base URL (e.g., https://mycompany.atlassian.net)
	BaseURL        string // REST API v3 URL
	AgileURL       string // Agile API URL
}

// ClientConfig contains configuration for creating a new client
type ClientConfig struct {
	URL      string // Full Jira URL (e.g., https://mycompany.atlassian.net or https://jira.internal.corp.com)
	Email    string
	APIToken string
	Verbose  bool
}

// New creates a new Jira API client from config
func New(cfg ClientConfig) (*Client, error) {
	if cfg.URL == "" {
		return nil, errors.ErrNotFound // Use generic error for now; specific errors defined below
	}
	if cfg.Email == "" {
		return nil, ErrEmailRequired
	}
	if cfg.APIToken == "" {
		return nil, ErrAPITokenRequired
	}

	// Normalize URL: ensure https and no trailing slash
	baseURL := url.NormalizeURL(cfg.URL)

	// Create shared client with verbose option
	var opts *client.Options
	if cfg.Verbose {
		opts = &client.Options{Verbose: true}
	}

	return &Client{
		Client:   client.New(baseURL, cfg.Email, cfg.APIToken, opts),
		URL:      baseURL,
		BaseURL:  baseURL + "/rest/api/3",
		AgileURL: baseURL + "/rest/agile/1.0",
	}, nil
}

// Validation errors
var (
	ErrURLRequired      = fmt.Errorf("URL is required")
	ErrEmailRequired    = fmt.Errorf("email is required")
	ErrAPITokenRequired = fmt.Errorf("API token is required")
)

// get performs a GET request to the specified URL
func (c *Client) get(urlStr string) ([]byte, error) {
	return c.Get(context.Background(), urlStr)
}

// post performs a POST request to the specified URL
func (c *Client) post(urlStr string, body interface{}) ([]byte, error) {
	return c.Post(context.Background(), urlStr, body)
}

// put performs a PUT request to the specified URL
func (c *Client) put(urlStr string, body interface{}) ([]byte, error) {
	return c.Put(context.Background(), urlStr, body)
}

// delete performs a DELETE request to the specified URL
func (c *Client) delete(urlStr string) ([]byte, error) {
	return c.Delete(context.Background(), urlStr)
}

// buildURL builds a URL with query parameters
func buildURL(base string, params map[string]string) string {
	if len(params) == 0 {
		return base
	}

	u, _ := neturl.Parse(base)
	q := u.Query()
	for k, v := range params {
		if v != "" {
			q.Set(k, v)
		}
	}
	u.RawQuery = q.Encode()
	return u.String()
}

// IssueURL returns the web URL for an issue
func (c *Client) IssueURL(issueKey string) string {
	return fmt.Sprintf("%s/browse/%s", c.URL, issueKey)
}

// GetHTTPClient returns the underlying HTTP client for custom requests.
func (c *Client) GetHTTPClient() *http.Client {
	return c.HTTPClient
}

// GetAuthHeader returns the authorization header value.
func (c *Client) GetAuthHeader() string {
	return c.AuthHeader
}
