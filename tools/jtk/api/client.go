// Package api provides a client for the Jira REST API.
package api //nolint:revive // package name is intentional

import (
	"context"
	"encoding/json"
	stderrors "errors"
	"fmt"
	"net/http"
	neturl "net/url"
	"strings"
	"sync"

	"github.com/open-cli-collective/atlassian-go/auth"
	"github.com/open-cli-collective/atlassian-go/client"
	"github.com/open-cli-collective/atlassian-go/url"
)

// Client is a Jira API client.
//
// The embedded *client.Client receives the same REST API URL as BaseURL,
// so both c.Client.BaseURL and c.BaseURL resolve to the same value.
// Callers should use the outer BaseURL field when constructing API paths.
// URL holds the instance URL for browse links (e.g., IssueURL).
type Client struct {
	*client.Client        // Embed shared client for HTTP methods (BaseURL == outer BaseURL)
	URL            string // Instance URL for browse links (e.g., https://mycompany.atlassian.net)
	BaseURL        string // REST API v3 URL (matches embedded client.Client.BaseURL)
	AgileURL       string // Agile API URL (empty for bearer auth)

	cloudID   string
	cloudOnce sync.Once
	cloudErr  error
}

// ClientConfig contains configuration for creating a new client
type ClientConfig struct {
	URL        string // Full Jira URL (e.g., https://mycompany.atlassian.net or https://jira.internal.corp.com)
	Email      string
	APIToken   string
	Verbose    bool
	AuthMethod string // "basic" (default) or "bearer"
	CloudID    string // Required for bearer auth (used to construct gateway URL)
}

// New creates a new Jira API client from config.
// For bearer auth: URL + API token + Cloud ID are required (no email).
// For basic auth: URL + email + API token are required.
func New(cfg ClientConfig) (*Client, error) {
	if cfg.URL == "" {
		return nil, ErrURLRequired
	}
	if cfg.APIToken == "" {
		return nil, ErrAPITokenRequired
	}

	// Validate auth method if explicitly set
	if cfg.AuthMethod != "" {
		if err := auth.ValidateAuthMethod(cfg.AuthMethod); err != nil {
			return nil, err
		}
	}

	if cfg.AuthMethod == auth.AuthMethodBearer {
		return newBearerClient(cfg)
	}

	// Basic auth (default)
	if cfg.Email == "" {
		return nil, ErrEmailRequired
	}

	// Normalize URL: ensure https and no trailing slash
	baseURL := url.NormalizeURL(cfg.URL)
	restURL := baseURL + "/rest/api/3"

	// Create shared client with verbose option
	var opts *client.Options
	if cfg.Verbose {
		opts = &client.Options{Verbose: true}
	}

	// Pass restURL to client.New so embedded BaseURL matches outer BaseURL.
	return &Client{
		Client:   client.New(restURL, cfg.Email, cfg.APIToken, opts),
		URL:      baseURL,
		BaseURL:  restURL,
		AgileURL: baseURL + "/rest/agile/1.0",
	}, nil
}

// newBearerClient creates a client configured for bearer auth via the API gateway.
func newBearerClient(cfg ClientConfig) (*Client, error) {
	if cfg.CloudID == "" {
		return nil, ErrCloudIDRequired
	}

	// The instance URL is kept for IssueURL() (browse links)
	instanceURL := url.NormalizeURL(cfg.URL)

	// Gateway URLs for bearer auth
	gatewayBase := fmt.Sprintf("%s/ex/jira/%s", client.GatewayBaseURL, cfg.CloudID)
	restURL := gatewayBase + "/rest/api/3"

	opts := &client.Options{
		AuthHeader: auth.BearerAuthHeader(cfg.APIToken),
	}
	if cfg.Verbose {
		opts.Verbose = true
	}

	// AgileURL is empty for bearer auth — scoped tokens lack Agile API scopes.
	// Use SupportsAgile() to check before making Agile calls.
	// Pass restURL to client.New so embedded BaseURL matches outer BaseURL.
	return &Client{
		Client:  client.New(restURL, "", "", opts),
		URL:     instanceURL,
		BaseURL: restURL,
	}, nil
}

// SupportsAgile returns true if the client can access the Agile REST API.
// Bearer auth clients (service accounts with scoped tokens) cannot access
// the Agile API because Atlassian does not provide an Agile scope.
func (c *Client) SupportsAgile() bool {
	return c.AgileURL != ""
}

// IsBearerAuth returns true if the client uses bearer authentication.
// Bearer auth clients (service accounts) lack scopes for the Agile,
// Automation, and Dashboard APIs.
func (c *Client) IsBearerAuth() bool {
	return strings.HasPrefix(c.GetAuthHeader(), "Bearer ")
}

// Validation errors
var (
	ErrURLRequired      = stderrors.New("URL is required")
	ErrEmailRequired    = stderrors.New("email is required")
	ErrAPITokenRequired = stderrors.New("API token is required")
	ErrCloudIDRequired  = stderrors.New("cloud ID is required for bearer auth")
)

// ErrAgileUnavailable is returned when a command requires the Agile API
// but the client does not support it (e.g., bearer auth with scoped tokens).
var ErrAgileUnavailable = stderrors.New("this command requires the Agile API, which is not available with bearer auth (scoped tokens lack the Agile scope)")

// ErrAutomationUnavailable is returned when a command requires the Automation API
// but the client does not support it (e.g., bearer auth with scoped tokens).
var ErrAutomationUnavailable = stderrors.New("this command requires the Automation API, which is not available with bearer auth (scoped tokens lack the Automation scope)")

// ErrDashboardUnavailable is returned when a command requires the Dashboard API
// but the client does not support it (e.g., bearer auth with scoped tokens).
var ErrDashboardUnavailable = stderrors.New("this command requires the Dashboard API, which is not available with bearer auth (scoped tokens lack the Dashboard scope)")

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
