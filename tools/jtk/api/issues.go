package api //nolint:revive // package name is intentional

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
)

// GetIssue retrieves an issue by key
func (c *Client) GetIssue(ctx context.Context, issueKey string) (*Issue, error) {
	if issueKey == "" {
		return nil, ErrIssueKeyRequired
	}

	urlStr := fmt.Sprintf("%s/issue/%s", c.BaseURL, url.PathEscape(issueKey))
	body, err := c.Get(ctx, urlStr)
	if err != nil {
		return nil, fmt.Errorf("fetching issue: %w", err)
	}

	var issue Issue
	if err := json.Unmarshal(body, &issue); err != nil {
		return nil, fmt.Errorf("parsing issue: %w", err)
	}

	return &issue, nil
}

// CreateIssue creates a new issue
func (c *Client) CreateIssue(ctx context.Context, req *CreateIssueRequest) (*Issue, error) {
	urlStr := fmt.Sprintf("%s/issue", c.BaseURL)
	body, err := c.Post(ctx, urlStr, req)
	if err != nil {
		return nil, fmt.Errorf("creating issue: %w", err)
	}

	var issue Issue
	if err := json.Unmarshal(body, &issue); err != nil {
		return nil, fmt.Errorf("parsing created issue: %w", err)
	}

	return &issue, nil
}

// UpdateIssue updates an existing issue
func (c *Client) UpdateIssue(ctx context.Context, issueKey string, req *UpdateIssueRequest) error {
	if issueKey == "" {
		return ErrIssueKeyRequired
	}

	urlStr := fmt.Sprintf("%s/issue/%s", c.BaseURL, url.PathEscape(issueKey))
	_, err := c.Put(ctx, urlStr, req)
	if err != nil {
		return fmt.Errorf("updating issue %s: %w", issueKey, err)
	}
	return nil
}

// DeleteIssue deletes an issue
func (c *Client) DeleteIssue(ctx context.Context, issueKey string) error {
	if issueKey == "" {
		return ErrIssueKeyRequired
	}

	urlStr := fmt.Sprintf("%s/issue/%s", c.BaseURL, url.PathEscape(issueKey))
	_, err := c.Delete(ctx, urlStr)
	if err != nil {
		return fmt.Errorf("deleting issue %s: %w", issueKey, err)
	}
	return nil
}

// AssignIssue assigns an issue to a user
func (c *Client) AssignIssue(ctx context.Context, issueKey, accountID string) error {
	if issueKey == "" {
		return ErrIssueKeyRequired
	}

	urlStr := fmt.Sprintf("%s/issue/%s/assignee", c.BaseURL, url.PathEscape(issueKey))

	body := map[string]any{}
	if accountID != "" {
		body["accountId"] = accountID
	} else {
		// Setting to null unassigns the issue
		body["accountId"] = nil
	}

	_, err := c.Put(ctx, urlStr, body)
	if err != nil {
		return fmt.Errorf("assigning issue %s: %w", issueKey, err)
	}
	return nil
}

// GetIssueEditMeta returns the edit metadata for an issue
func (c *Client) GetIssueEditMeta(ctx context.Context, issueKey string) (map[string]any, error) {
	if issueKey == "" {
		return nil, ErrIssueKeyRequired
	}

	urlStr := fmt.Sprintf("%s/issue/%s/editmeta", c.BaseURL, url.PathEscape(issueKey))
	body, err := c.Get(ctx, urlStr)
	if err != nil {
		return nil, fmt.Errorf("fetching edit metadata: %w", err)
	}

	var result map[string]any
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parsing edit metadata: %w", err)
	}

	return result, nil
}

// BuildCreateRequest builds a create issue request
func BuildCreateRequest(projectKey, issueType, summary, description string, extraFields map[string]any) *CreateIssueRequest {
	fields := map[string]any{
		"project":   map[string]string{"key": projectKey},
		"issuetype": map[string]string{"name": issueType},
		"summary":   summary,
	}

	if description != "" {
		fields["description"] = NewADFDocument(description)
	}

	for k, v := range extraFields {
		fields[k] = v
	}

	return &CreateIssueRequest{Fields: fields}
}

// BuildUpdateRequest builds an update issue request
func BuildUpdateRequest(fields map[string]any) *UpdateIssueRequest {
	return &UpdateIssueRequest{Fields: fields}
}
