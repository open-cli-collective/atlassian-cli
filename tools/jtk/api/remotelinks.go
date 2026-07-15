package api //nolint:revive // package name is intentional

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
)

// RemoteLink represents a Jira issue remote link (a "Web Link" in the Jira UI):
// an external URL attached to an issue and shown in the issue's links sidebar.
// See POST/GET /rest/api/3/issue/{issueIdOrKey}/remotelink.
type RemoteLink struct {
	ID           int              `json:"id"`
	Self         string           `json:"self,omitempty"`
	GlobalID     string           `json:"globalId,omitempty"`
	Relationship string           `json:"relationship,omitempty"`
	Application  *RemoteLinkApp   `json:"application,omitempty"`
	Object       RemoteLinkObject `json:"object"`
}

// RemoteLinkApp identifies the application a remote link belongs to.
type RemoteLinkApp struct {
	Type string `json:"type,omitempty"`
	Name string `json:"name,omitempty"`
}

// RemoteLinkObject holds the external resource a remote link points at.
type RemoteLinkObject struct {
	URL     string `json:"url"`
	Title   string `json:"title"`
	Summary string `json:"summary,omitempty"`
}

// CreateRemoteLinkRequest is the body for creating/updating a remote link.
type CreateRemoteLinkRequest struct {
	GlobalID     string           `json:"globalId,omitempty"`
	Relationship string           `json:"relationship,omitempty"`
	Object       RemoteLinkObject `json:"object"`
}

// remoteLinkCreateResponse is the slim response Jira returns from create:
// it identifies the link but does not echo back the full object.
type remoteLinkCreateResponse struct {
	ID   int    `json:"id"`
	Self string `json:"self"`
}

// GetRemoteLinks returns the remote (web) links on an issue.
func (c *Client) GetRemoteLinks(ctx context.Context, issueKey string) ([]RemoteLink, error) {
	if issueKey == "" {
		return nil, ErrIssueKeyRequired
	}

	urlStr := fmt.Sprintf("%s/issue/%s/remotelink", c.BaseURL, url.PathEscape(issueKey))
	body, err := c.Get(ctx, urlStr)
	if err != nil {
		return nil, fmt.Errorf("fetching remote links: %w", err)
	}

	var links []RemoteLink
	if err := json.Unmarshal(body, &links); err != nil {
		return nil, fmt.Errorf("parsing remote links: %w", err)
	}

	return links, nil
}

// AddRemoteLink creates a remote (web) link on an issue pointing at url with
// the given title. Jira's create response is slim (id + self only), so the
// returned RemoteLink echoes back the input object alongside the new ID.
func (c *Client) AddRemoteLink(ctx context.Context, issueKey string, req CreateRemoteLinkRequest) (*RemoteLink, error) {
	if issueKey == "" {
		return nil, ErrIssueKeyRequired
	}
	if req.Object.URL == "" {
		return nil, ErrRemoteLinkURLRequired
	}
	if req.Object.Title == "" {
		return nil, ErrRemoteLinkTitleRequired
	}

	urlStr := fmt.Sprintf("%s/issue/%s/remotelink", c.BaseURL, url.PathEscape(issueKey))
	body, err := c.Post(ctx, urlStr, req)
	if err != nil {
		return nil, fmt.Errorf("adding remote link: %w", err)
	}

	var resp remoteLinkCreateResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parsing remote link: %w", err)
	}

	return &RemoteLink{
		ID:           resp.ID,
		Self:         resp.Self,
		GlobalID:     req.GlobalID,
		Relationship: req.Relationship,
		Object:       req.Object,
	}, nil
}

// DeleteRemoteLink deletes a remote link from an issue by its link ID. linkID
// is an int to match the RemoteLink.ID domain type returned by AddRemoteLink
// and GetRemoteLinks, so a list-then-delete flow needs no string conversion.
func (c *Client) DeleteRemoteLink(ctx context.Context, issueKey string, linkID int) error {
	if issueKey == "" {
		return ErrIssueKeyRequired
	}
	if linkID <= 0 {
		return ErrRemoteLinkIDRequired
	}

	urlStr := fmt.Sprintf("%s/issue/%s/remotelink/%d", c.BaseURL, url.PathEscape(issueKey), linkID)
	if _, err := c.Delete(ctx, urlStr); err != nil {
		return fmt.Errorf("deleting remote link %d: %w", linkID, err)
	}
	return nil
}
