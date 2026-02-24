package api //nolint:revive // package name is intentional

import (
	"context"
	"encoding/json"
	"fmt"
)

// SearchOptions contains options for JQL search.
type SearchOptions struct {
	JQL           string
	MaxResults    int
	Fields        []string
	NextPageToken string
}

// SearchRequest is the request body for the /search/jql endpoint.
type SearchRequest struct {
	JQL           string   `json:"jql"`
	MaxResults    int      `json:"maxResults,omitempty"`
	Fields        []string `json:"fields,omitempty"`
	NextPageToken string   `json:"nextPageToken,omitempty"`
}

// DefaultSearchFields are the fields returned by default in search results.
var DefaultSearchFields = []string{
	"summary",
	"status",
	"assignee",
	"issuetype",
	"priority",
	"project",
	"created",
	"updated",
	"description",
	"labels",
	"components",
	"reporter",
	"parent",
}

// ListSearchFields are lightweight fields for list/search commands (no description).
var ListSearchFields = []string{
	"summary",
	"status",
	"assignee",
	"issuetype",
	"priority",
	"project",
	"labels",
	"created",
	"updated",
}

// Search searches for issues using JQL (uses /search/jql endpoint).
func (c *Client) Search(ctx context.Context, opts SearchOptions) (*JQLSearchResult, error) {
	req := SearchRequest{
		JQL: opts.JQL,
	}

	if opts.MaxResults > 0 {
		req.MaxResults = opts.MaxResults
	} else {
		req.MaxResults = 50
	}

	if opts.NextPageToken != "" {
		req.NextPageToken = opts.NextPageToken
	}

	// Use default fields if none specified - new API requires explicit field selection
	if len(opts.Fields) > 0 {
		req.Fields = opts.Fields
	} else {
		req.Fields = DefaultSearchFields
	}

	urlStr := fmt.Sprintf("%s/search/jql", c.BaseURL)
	body, err := c.Post(ctx, urlStr, req)
	if err != nil {
		return nil, fmt.Errorf("searching issues: %w", err)
	}

	var result JQLSearchResult
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parsing search results: %w", err)
	}

	return &result, nil
}

// SearchAll searches for all issues matching JQL (handles cursor-based pagination).
func (c *Client) SearchAll(ctx context.Context, jql string, maxResults int) ([]Issue, error) {
	if maxResults <= 0 {
		maxResults = 1000
	}

	var allIssues []Issue
	pageSize := 100
	nextPageToken := ""

	for {
		if err := ctx.Err(); err != nil {
			return nil, fmt.Errorf("searching all issues: %w", err)
		}

		result, err := c.Search(ctx, SearchOptions{
			JQL:           jql,
			MaxResults:    pageSize,
			NextPageToken: nextPageToken,
		})
		if err != nil {
			return nil, fmt.Errorf("searching all issues: %w", err)
		}

		allIssues = append(allIssues, result.Issues...)

		if result.IsLast || len(allIssues) >= maxResults {
			break
		}

		nextPageToken = result.NextPageToken
		if nextPageToken == "" || len(result.Issues) == 0 {
			break
		}
	}

	if len(allIssues) > maxResults {
		allIssues = allIssues[:maxResults]
	}

	return allIssues, nil
}

// SearchPage searches for a single page of issues and returns results with pagination metadata.
func (c *Client) SearchPage(ctx context.Context, opts SearchPageOptions) (*PaginatedIssues, error) {
	pageSize := opts.PageSize
	if pageSize <= 0 {
		pageSize = 25
	}

	result, err := c.Search(ctx, SearchOptions{
		JQL:           opts.JQL,
		MaxResults:    pageSize,
		Fields:        opts.Fields,
		NextPageToken: opts.NextPageToken,
	})
	if err != nil {
		return nil, err
	}

	return &PaginatedIssues{
		Issues: result.Issues,
		Pagination: PaginationInfo{
			PageSize:      pageSize,
			IsLast:        result.IsLast,
			NextPageToken: result.NextPageToken,
		},
	}, nil
}
