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

// SearchPage searches for issues and returns results with pagination metadata.
// When MaxResults > 0 and exceeds the per-request page size, it automatically
// paginates through multiple API calls to collect up to MaxResults issues.
func (c *Client) SearchPage(ctx context.Context, opts SearchPageOptions) (*PaginatedIssues, error) {
	maxResults := opts.MaxResults
	pageSize := opts.PageSize

	// Determine effective page size
	if pageSize <= 0 {
		if maxResults > 0 {
			pageSize = maxResults
		} else {
			pageSize = 25
		}
	}
	if pageSize > 100 {
		pageSize = 100 // Jira API cap
	}

	// Single-page mode: no MaxResults or fits in one page
	if maxResults <= 0 || maxResults <= pageSize {
		effectiveSize := pageSize
		if maxResults > 0 && maxResults < effectiveSize {
			effectiveSize = maxResults
		}

		result, err := c.Search(ctx, SearchOptions{
			JQL:           opts.JQL,
			MaxResults:    effectiveSize,
			Fields:        opts.Fields,
			NextPageToken: opts.NextPageToken,
		})
		if err != nil {
			return nil, err
		}

		return &PaginatedIssues{
			Issues: result.Issues,
			Pagination: PaginationInfo{
				Total:         len(result.Issues),
				PageSize:      effectiveSize,
				IsLast:        result.IsLast,
				NextPageToken: result.NextPageToken,
			},
		}, nil
	}

	// Multi-page mode: loop until we have maxResults or hit the last page
	var allIssues []Issue
	nextToken := opts.NextPageToken
	isLast := false

	for len(allIssues) < maxResults {
		if err := ctx.Err(); err != nil {
			return nil, fmt.Errorf("searching issues: %w", err)
		}

		remaining := maxResults - len(allIssues)
		fetchSize := pageSize
		if remaining < fetchSize {
			fetchSize = remaining
		}

		result, err := c.Search(ctx, SearchOptions{
			JQL:           opts.JQL,
			MaxResults:    fetchSize,
			Fields:        opts.Fields,
			NextPageToken: nextToken,
		})
		if err != nil {
			return nil, err
		}

		allIssues = append(allIssues, result.Issues...)

		if result.IsLast || len(result.Issues) == 0 {
			isLast = true
			nextToken = ""
			break
		}

		nextToken = result.NextPageToken
		if nextToken == "" {
			isLast = true
			break
		}
	}

	// Trim to maxResults if the last page overshot
	if len(allIssues) > maxResults {
		allIssues = allIssues[:maxResults]
		isLast = false // truncated, so more results exist
	}

	return &PaginatedIssues{
		Issues: allIssues,
		Pagination: PaginationInfo{
			Total:         len(allIssues),
			PageSize:      pageSize,
			IsLast:        isLast,
			NextPageToken: nextToken,
		},
	}, nil
}
