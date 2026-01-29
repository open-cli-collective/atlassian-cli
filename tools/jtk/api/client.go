package api

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// Client is a Jira API client
type Client struct {
	URL        string // Base URL (e.g., https://mycompany.atlassian.net)
	Email      string
	APIToken   string
	BaseURL    string // REST API v3 URL
	AgileURL   string // Agile API URL
	HTTPClient *http.Client
	Verbose    bool
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
	baseURL := cfg.URL
	if !hasScheme(baseURL) {
		baseURL = "https://" + baseURL
	}
	baseURL = trimTrailingSlash(baseURL)

	return &Client{
		URL:      baseURL,
		Email:    cfg.Email,
		APIToken: cfg.APIToken,
		BaseURL:  baseURL + "/rest/api/3",
		AgileURL: baseURL + "/rest/agile/1.0",
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		Verbose: cfg.Verbose,
	}, nil
}

// hasScheme checks if a URL has an http or https scheme
func hasScheme(u string) bool {
	return len(u) >= 7 && (u[:7] == "http://" || (len(u) >= 8 && u[:8] == "https://"))
}

// trimTrailingSlash removes trailing slashes from a URL
func trimTrailingSlash(u string) string {
	for len(u) > 0 && u[len(u)-1] == '/' {
		u = u[:len(u)-1]
	}
	return u
}

// authHeader returns the Basic Auth header value
func (c *Client) authHeader() string {
	auth := fmt.Sprintf("%s:%s", c.Email, c.APIToken)
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(auth))
}

// doRequest performs an HTTP request with authentication
func (c *Client) doRequest(method, urlStr string, body interface{}) ([]byte, error) {
	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequest(method, urlStr, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", c.authHeader())
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	if c.Verbose {
		fmt.Printf("→ %s %s\n", method, urlStr)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if c.Verbose {
		fmt.Printf("← %d %s\n", resp.StatusCode, http.StatusText(resp.StatusCode))
	}

	if resp.StatusCode >= 400 {
		return nil, ParseAPIError(resp, respBody)
	}

	return respBody, nil
}

// get performs a GET request
func (c *Client) get(urlStr string) ([]byte, error) {
	return c.doRequest(http.MethodGet, urlStr, nil)
}

// post performs a POST request
func (c *Client) post(urlStr string, body interface{}) ([]byte, error) {
	return c.doRequest(http.MethodPost, urlStr, body)
}

// put performs a PUT request
func (c *Client) put(urlStr string, body interface{}) ([]byte, error) {
	return c.doRequest(http.MethodPut, urlStr, body)
}

// delete performs a DELETE request
func (c *Client) delete(urlStr string) ([]byte, error) {
	return c.doRequest(http.MethodDelete, urlStr, nil)
}

// buildURL builds a URL with query parameters
func buildURL(base string, params map[string]string) string {
	if len(params) == 0 {
		return base
	}

	u, _ := url.Parse(base)
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
