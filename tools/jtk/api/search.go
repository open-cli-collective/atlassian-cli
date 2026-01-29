package api

import (
	"encoding/json"
	"fmt"
)

// SearchOptions contains options for JQL search
type SearchOptions struct {
	JQL        string
	StartAt    int
	MaxResults int
	Fields     []string
}

// SearchRequest is the request body for the new JQL search API
type SearchRequest struct {
	JQL        string   `json:"jql"`
	StartAt    int      `json:"startAt,omitempty"`
	MaxResults int      `json:"maxResults,omitempty"`
	Fields     []string `json:"fields,omitempty"`
}

// DefaultSearchFields are the fields returned by default in search results
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

// Search searches for issues using JQL (uses new /search/jql endpoint)
func (c *Client) Search(opts SearchOptions) (*SearchResult, error) {
	req := SearchRequest{
		JQL: opts.JQL,
	}

	if opts.StartAt > 0 {
		req.StartAt = opts.StartAt
	}

	if opts.MaxResults > 0 {
		req.MaxResults = opts.MaxResults
	} else {
		req.MaxResults = 50
	}

	// Use default fields if none specified - new API requires explicit field selection
	if len(opts.Fields) > 0 {
		req.Fields = opts.Fields
	} else {
		req.Fields = DefaultSearchFields
	}

	urlStr := fmt.Sprintf("%s/search/jql", c.BaseURL)
	body, err := c.post(urlStr, req)
	if err != nil {
		return nil, err
	}

	var result SearchResult
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse search results: %w", err)
	}

	return &result, nil
}

// SearchAll searches for all issues matching JQL (handles pagination)
func (c *Client) SearchAll(jql string, maxResults int) ([]Issue, error) {
	if maxResults <= 0 {
		maxResults = 1000
	}

	var allIssues []Issue
	startAt := 0
	pageSize := 100

	for {
		result, err := c.Search(SearchOptions{
			JQL:        jql,
			StartAt:    startAt,
			MaxResults: pageSize,
		})
		if err != nil {
			return nil, err
		}

		allIssues = append(allIssues, result.Issues...)

		if len(allIssues) >= result.Total || len(allIssues) >= maxResults {
			break
		}

		startAt += len(result.Issues)
		if len(result.Issues) == 0 {
			break
		}
	}

	if len(allIssues) > maxResults {
		allIssues = allIssues[:maxResults]
	}

	return allIssues, nil
}
