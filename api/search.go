package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// SearchOptions contains options for searching Confluence content.
type SearchOptions struct {
	CQL   string // Raw CQL query (takes precedence if set)
	Text  string // Full-text search term
	Space string // Space key to filter results
	Type  string // Content type: page, blogpost, attachment, comment
	Title string // Title contains filter
	Label string // Label filter
	Limit int    // Max results (default 25, max 200)
}

// SearchResult represents a single search result from the v1 API.
type SearchResult struct {
	Content               SearchContent   `json:"content"`
	Title                 string          `json:"title"`
	Excerpt               string          `json:"excerpt"`
	URL                   string          `json:"url"`
	ResultGlobalContainer SearchContainer `json:"resultGlobalContainer"`
	LastModified          string          `json:"lastModified"`
	FriendlyLastModified  string          `json:"friendlyLastModified"`
}

// SearchContent contains the content details in a search result.
type SearchContent struct {
	ID     string `json:"id"`
	Type   string `json:"type"`
	Status string `json:"status"`
	Title  string `json:"title"`
}

// SearchContainer represents the space/container of a search result.
type SearchContainer struct {
	Title      string `json:"title"`
	DisplayURL string `json:"displayUrl"`
}

// SearchResponse represents the v1 search API response.
type SearchResponse struct {
	Results        []SearchResult `json:"results"`
	Start          int            `json:"start"`
	Limit          int            `json:"limit"`
	Size           int            `json:"size"`
	TotalSize      int            `json:"totalSize"`
	CQLQuery       string         `json:"cqlQuery"`
	SearchDuration int            `json:"searchDuration"`
}

// HasMore returns true if there are more results available.
func (r *SearchResponse) HasMore() bool {
	return r.Start+r.Size < r.TotalSize
}

// Search performs a Confluence search using CQL.
// Uses the v1 REST API: GET /rest/api/search
func (c *Client) Search(ctx context.Context, opts *SearchOptions) (*SearchResponse, error) {
	params := url.Values{}

	// Build CQL query
	cql := ""
	if opts != nil {
		cql = opts.CQL
		if cql == "" {
			cql = buildCQL(opts)
		}
	}

	if cql == "" {
		return nil, fmt.Errorf("search requires a query or filters")
	}

	params.Set("cql", cql)

	// Pagination
	if opts != nil && opts.Limit > 0 {
		params.Set("limit", strconv.Itoa(opts.Limit))
	} else {
		params.Set("limit", "25")
	}

	// Include excerpt for context
	params.Set("excerpt", "highlight")

	path := "/rest/api/search?" + params.Encode()
	body, err := c.Get(ctx, path)
	if err != nil {
		return nil, err
	}

	var result SearchResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse search response: %w", err)
	}

	return &result, nil
}

// buildCQL constructs a CQL query from search options.
func buildCQL(opts *SearchOptions) string {
	var clauses []string

	// Full-text search using text~
	if opts.Text != "" {
		clauses = append(clauses, fmt.Sprintf(`text ~ %q`, opts.Text))
	}

	// Space filter
	if opts.Space != "" {
		clauses = append(clauses, fmt.Sprintf(`space = %q`, opts.Space))
	}

	// Type filter
	if opts.Type != "" {
		clauses = append(clauses, fmt.Sprintf(`type = %q`, opts.Type))
	}

	// Title filter
	if opts.Title != "" {
		clauses = append(clauses, fmt.Sprintf(`title ~ %q`, opts.Title))
	}

	// Label filter
	if opts.Label != "" {
		clauses = append(clauses, fmt.Sprintf(`label = %q`, opts.Label))
	}

	if len(clauses) == 0 {
		return ""
	}

	return strings.Join(clauses, " AND ")
}
