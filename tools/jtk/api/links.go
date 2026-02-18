package api

import (
	"encoding/json"
	"fmt"
	"net/url"
)

// IssueLinkType represents a type of link between issues (e.g., "Blocks", "Relates")
type IssueLinkType struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Inward  string `json:"inward"`
	Outward string `json:"outward"`
}

// IssueLink represents a link between two issues
type IssueLink struct {
	ID   string        `json:"id"`
	Type IssueLinkType `json:"type"`
	// Exactly one of InwardIssue or OutwardIssue will be set when reading links from an issue
	InwardIssue  *LinkedIssue `json:"inwardIssue,omitempty"`
	OutwardIssue *LinkedIssue `json:"outwardIssue,omitempty"`
}

// LinkedIssue represents the summary info of a linked issue
type LinkedIssue struct {
	ID     string `json:"id"`
	Key    string `json:"key"`
	Fields struct {
		Summary   string     `json:"summary"`
		Status    *Status    `json:"status,omitempty"`
		IssueType *IssueType `json:"issuetype,omitempty"`
	} `json:"fields"`
}

// CreateIssueLinkRequest represents a request to create a link between two issues
type CreateIssueLinkRequest struct {
	Type         IssueLinkTypeRef `json:"type"`
	InwardIssue  IssueRef         `json:"inwardIssue"`
	OutwardIssue IssueRef         `json:"outwardIssue"`
}

// IssueLinkTypeRef identifies a link type by name
type IssueLinkTypeRef struct {
	Name string `json:"name"`
}

// IssueRef identifies an issue by key
type IssueRef struct {
	Key string `json:"key"`
}

// GetIssueLinks returns the links on an issue by fetching the issue and extracting the issuelinks field
func (c *Client) GetIssueLinks(issueKey string) ([]IssueLink, error) {
	if issueKey == "" {
		return nil, ErrIssueKeyRequired
	}

	urlStr := buildURL(
		fmt.Sprintf("%s/issue/%s", c.BaseURL, url.PathEscape(issueKey)),
		map[string]string{"fields": "issuelinks"},
	)

	body, err := c.get(urlStr)
	if err != nil {
		return nil, err
	}

	var result struct {
		Fields struct {
			IssueLinks []IssueLink `json:"issuelinks"`
		} `json:"fields"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse issue links: %w", err)
	}

	return result.Fields.IssueLinks, nil
}

// CreateIssueLink creates a link between two issues
func (c *Client) CreateIssueLink(outwardKey, inwardKey, linkTypeName string) error {
	if outwardKey == "" || inwardKey == "" {
		return ErrIssueKeyRequired
	}
	if linkTypeName == "" {
		return fmt.Errorf("link type name is required")
	}

	urlStr := fmt.Sprintf("%s/issueLink", c.BaseURL)
	req := CreateIssueLinkRequest{
		Type:         IssueLinkTypeRef{Name: linkTypeName},
		OutwardIssue: IssueRef{Key: outwardKey},
		InwardIssue:  IssueRef{Key: inwardKey},
	}

	_, err := c.post(urlStr, req)
	return err
}

// DeleteIssueLink deletes an issue link by its ID
func (c *Client) DeleteIssueLink(linkID string) error {
	if linkID == "" {
		return fmt.Errorf("link ID is required")
	}

	urlStr := fmt.Sprintf("%s/issueLink/%s", c.BaseURL, url.PathEscape(linkID))
	_, err := c.delete(urlStr)
	return err
}

// GetIssueLinkTypes returns all available issue link types
func (c *Client) GetIssueLinkTypes() ([]IssueLinkType, error) {
	urlStr := fmt.Sprintf("%s/issueLinkType", c.BaseURL)

	body, err := c.get(urlStr)
	if err != nil {
		return nil, err
	}

	var result struct {
		IssueLinkTypes []IssueLinkType `json:"issueLinkTypes"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse link types: %w", err)
	}

	return result.IssueLinkTypes, nil
}
