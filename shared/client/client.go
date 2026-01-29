// Package client provides an HTTP client for Atlassian REST APIs.
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/open-cli-collective/atlassian-go/auth"
	"github.com/open-cli-collective/atlassian-go/errors"
)

// Client is an HTTP client for Atlassian APIs.
type Client struct {
	// BaseURL is the base URL for API requests (e.g., "https://example.atlassian.net/wiki").
	BaseURL string

	// AuthHeader is the pre-computed Authorization header value.
	AuthHeader string

	// HTTPClient is the underlying HTTP client.
	HTTPClient *http.Client

	// Verbose enables request/response logging.
	Verbose bool

	// VerboseOut is the writer for verbose output.
	VerboseOut io.Writer
}

// New creates a new API client.
//
// The baseURL should include any required path prefix (e.g., "/wiki" for Confluence).
// The email and apiToken are used to generate Basic authentication.
func New(baseURL, email, apiToken string, opts *Options) *Client {
	baseURL = strings.TrimSuffix(baseURL, "/")

	var timeout = DefaultTimeout
	var verbose bool
	var verboseOut io.Writer = os.Stderr

	if opts != nil {
		timeout = opts.timeoutOrDefault()
		verbose = opts.Verbose
		if opts.VerboseOut != nil {
			verboseOut = opts.VerboseOut
		}
	}

	return &Client{
		BaseURL:    baseURL,
		AuthHeader: auth.BasicAuthHeader(email, apiToken),
		HTTPClient: &http.Client{
			Timeout: timeout,
		},
		Verbose:    verbose,
		VerboseOut: verboseOut,
	}
}

// Do executes an HTTP request with the given method, path, and optional body.
//
// The path can be either relative to the BaseURL (e.g., "/rest/api/3/issue")
// or an absolute URL (e.g., "https://example.com/api/resource").
// If body is not nil, it will be JSON-encoded.
// Returns the response body or an error (which may be an *errors.APIError).
func (c *Client) Do(ctx context.Context, method, path string, body interface{}) ([]byte, error) {
	var url string

	// Check if path is an absolute URL
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		url = path
	} else {
		// Ensure path starts with /
		if !strings.HasPrefix(path, "/") {
			path = "/" + path
		}
		url = c.BaseURL + path
	}

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
	req.Header.Set("Authorization", c.AuthHeader)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	if c.Verbose {
		_, _ = fmt.Fprintf(c.VerboseOut, "→ %s %s\n", method, url)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if c.Verbose {
		_, _ = fmt.Fprintf(c.VerboseOut, "← %d %s\n", resp.StatusCode, http.StatusText(resp.StatusCode))
	}

	// Handle error responses
	if resp.StatusCode >= 400 {
		return nil, errors.ParseAPIError(resp.StatusCode, respBody)
	}

	return respBody, nil
}

// Get performs a GET request.
func (c *Client) Get(ctx context.Context, path string) ([]byte, error) {
	return c.Do(ctx, http.MethodGet, path, nil)
}

// Post performs a POST request with a JSON body.
func (c *Client) Post(ctx context.Context, path string, body interface{}) ([]byte, error) {
	return c.Do(ctx, http.MethodPost, path, body)
}

// Put performs a PUT request with a JSON body.
func (c *Client) Put(ctx context.Context, path string, body interface{}) ([]byte, error) {
	return c.Do(ctx, http.MethodPut, path, body)
}

// Delete performs a DELETE request.
func (c *Client) Delete(ctx context.Context, path string) ([]byte, error) {
	return c.Do(ctx, http.MethodDelete, path, nil)
}
