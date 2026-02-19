// Package api provides a client for the Jira REST API.
package api //nolint:revive // package name is intentional

import (
	"context"
	"encoding/json"
	stderrors "errors"
	"fmt"
	"net/http"
	neturl "net/url"
	"sync"

	"github.com/open-cli-collective/atlassian-go/client"
	"github.com/open-cli-collective/atlassian-go/url"
)

// Client is a Jira API client
type Client struct {
	*client.Client        // Embed shared client for HTTP methods
	URL            string // Base URL (e.g., https://mycompany.atlassian.net)
	BaseURL        string // REST API v3 URL
	AgileURL       string // Agile API URL

	cloudID   string
	cloudOnce sync.Once
	cloudErr  error
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
		return nil, ErrURLRequired
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
	ErrURLRequired      = stderrors.New("URL is required")
	ErrEmailRequired    = stderrors.New("email is required")
	ErrAPITokenRequired = stderrors.New("API token is required")
)

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

// tenantInfo is the response from /_edge/tenant_info
type tenantInfo struct {
	CloudID string `json:"cloudId"`
}

// GetCloudID returns the Atlassian cloud ID for this site, fetching it on first call.
func (c *Client) GetCloudID(ctx context.Context) (string, error) {
	c.cloudOnce.Do(func() {
		urlStr := fmt.Sprintf("%s/_edge/tenant_info", c.URL)
		body, err := c.Get(ctx, urlStr)
		if err != nil {
			c.cloudErr = fmt.Errorf("fetching cloud ID from %s: %w", urlStr, err)
			return
		}

		var info tenantInfo
		if err := json.Unmarshal(body, &info); err != nil {
			c.cloudErr = fmt.Errorf("parsing tenant info: %w", err)
			return
		}

		if info.CloudID == "" {
			c.cloudErr = stderrors.New("tenant info returned empty cloud ID")
			return
		}

		c.cloudID = info.CloudID
	})

	return c.cloudID, c.cloudErr
}

// AutomationBaseURL returns the base URL for the Jira Automation REST API.
func (c *Client) AutomationBaseURL(ctx context.Context) (string, error) {
	cloudID, err := c.GetCloudID(ctx)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s/gateway/api/automation/public/jira/%s/rest/v1", c.URL, cloudID), nil
}
