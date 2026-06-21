package api //nolint:revive // package name is intentional

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
)

// IssueChangelogOptions controls Jira issue changelog pagination.
type IssueChangelogOptions struct {
	StartAt    int
	MaxResults int
}

// IssueChangelogPage represents one page from Jira's issue changelog API.
type IssueChangelogPage struct {
	StartAt    int                     `json:"startAt"`
	MaxResults int                     `json:"maxResults"`
	Total      int                     `json:"total"`
	Histories  []IssueChangelogHistory `json:"values"`
}

// IssueChangelogHistory is one Jira changelog group.
type IssueChangelogHistory struct {
	ID      string               `json:"id"`
	Author  *User                `json:"author,omitempty"`
	Created string               `json:"created"`
	Items   []IssueChangelogItem `json:"items"`
}

// IssueChangelogItem describes one changed field within a changelog group.
type IssueChangelogItem struct {
	Field      string `json:"field"`
	FieldType  string `json:"fieldtype"`
	FieldID    string `json:"fieldId,omitempty"`
	From       string `json:"from"`
	FromString string `json:"fromString"`
	To         string `json:"to"`
	ToString   string `json:"toString"`
}

// GetIssueChangelog retrieves one page of issue changelog history.
func (c *Client) GetIssueChangelog(ctx context.Context, issueKey string, opts IssueChangelogOptions) (*IssueChangelogPage, error) {
	if issueKey == "" {
		return nil, ErrIssueKeyRequired
	}

	params := map[string]string{}
	if opts.StartAt > 0 {
		params["startAt"] = strconv.Itoa(opts.StartAt)
	}
	if opts.MaxResults > 0 {
		params["maxResults"] = strconv.Itoa(opts.MaxResults)
	}

	urlStr := buildURL(fmt.Sprintf("%s/issue/%s/changelog", c.BaseURL, url.PathEscape(issueKey)), params)
	body, err := c.Get(ctx, urlStr)
	if err != nil {
		return nil, fmt.Errorf("fetching issue changelog: %w", err)
	}

	var page IssueChangelogPage
	if err := json.Unmarshal(body, &page); err != nil {
		return nil, fmt.Errorf("parsing issue changelog: %w", err)
	}

	return &page, nil
}
