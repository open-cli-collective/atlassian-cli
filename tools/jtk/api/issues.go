package api

import (
	"encoding/json"
	"fmt"
	"net/url"
)

// GetIssue retrieves an issue by key
func (c *Client) GetIssue(issueKey string) (*Issue, error) {
	if issueKey == "" {
		return nil, ErrIssueKeyRequired
	}

	urlStr := fmt.Sprintf("%s/issue/%s", c.BaseURL, url.PathEscape(issueKey))
	body, err := c.get(urlStr)
	if err != nil {
		return nil, err
	}

	var issue Issue
	if err := json.Unmarshal(body, &issue); err != nil {
		return nil, fmt.Errorf("failed to parse issue: %w", err)
	}

	return &issue, nil
}

// CreateIssue creates a new issue
func (c *Client) CreateIssue(req *CreateIssueRequest) (*Issue, error) {
	urlStr := fmt.Sprintf("%s/issue", c.BaseURL)
	body, err := c.post(urlStr, req)
	if err != nil {
		return nil, err
	}

	var issue Issue
	if err := json.Unmarshal(body, &issue); err != nil {
		return nil, fmt.Errorf("failed to parse created issue: %w", err)
	}

	return &issue, nil
}

// UpdateIssue updates an existing issue
func (c *Client) UpdateIssue(issueKey string, req *UpdateIssueRequest) error {
	if issueKey == "" {
		return ErrIssueKeyRequired
	}

	urlStr := fmt.Sprintf("%s/issue/%s", c.BaseURL, url.PathEscape(issueKey))
	_, err := c.put(urlStr, req)
	return err
}

// DeleteIssue deletes an issue
func (c *Client) DeleteIssue(issueKey string) error {
	if issueKey == "" {
		return ErrIssueKeyRequired
	}

	urlStr := fmt.Sprintf("%s/issue/%s", c.BaseURL, url.PathEscape(issueKey))
	_, err := c.delete(urlStr)
	return err
}

// AssignIssue assigns an issue to a user
func (c *Client) AssignIssue(issueKey, accountID string) error {
	if issueKey == "" {
		return ErrIssueKeyRequired
	}

	urlStr := fmt.Sprintf("%s/issue/%s/assignee", c.BaseURL, url.PathEscape(issueKey))

	body := map[string]interface{}{}
	if accountID != "" {
		body["accountId"] = accountID
	} else {
		// Setting to null unassigns the issue
		body["accountId"] = nil
	}

	_, err := c.put(urlStr, body)
	return err
}

// GetIssueEditMeta returns the edit metadata for an issue
func (c *Client) GetIssueEditMeta(issueKey string) (map[string]interface{}, error) {
	if issueKey == "" {
		return nil, ErrIssueKeyRequired
	}

	urlStr := fmt.Sprintf("%s/issue/%s/editmeta", c.BaseURL, url.PathEscape(issueKey))
	body, err := c.get(urlStr)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse edit metadata: %w", err)
	}

	return result, nil
}

// BuildCreateRequest builds a create issue request
func BuildCreateRequest(projectKey, issueType, summary, description string, extraFields map[string]interface{}) *CreateIssueRequest {
	fields := map[string]interface{}{
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
func BuildUpdateRequest(fields map[string]interface{}) *UpdateIssueRequest {
	return &UpdateIssueRequest{Fields: fields}
}
