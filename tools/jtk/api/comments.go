package api //nolint:revive // package name is intentional

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
)

// GetComments returns comments for an issue
func (c *Client) GetComments(ctx context.Context, issueKey string, startAt, maxResults int) (*CommentsResponse, error) {
	if issueKey == "" {
		return nil, ErrIssueKeyRequired
	}

	params := map[string]string{}
	if startAt > 0 {
		params["startAt"] = strconv.Itoa(startAt)
	}
	if maxResults > 0 {
		params["maxResults"] = strconv.Itoa(maxResults)
	}

	urlStr := buildURL(fmt.Sprintf("%s/issue/%s/comment", c.BaseURL, url.PathEscape(issueKey)), params)
	body, err := c.Get(ctx, urlStr)
	if err != nil {
		return nil, fmt.Errorf("fetching comments: %w", err)
	}

	var result CommentsResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parsing comments: %w", err)
	}

	return &result, nil
}

// AddComment adds a comment to an issue
func (c *Client) AddComment(ctx context.Context, issueKey, commentBody string) (*Comment, error) {
	if issueKey == "" {
		return nil, ErrIssueKeyRequired
	}

	urlStr := fmt.Sprintf("%s/issue/%s/comment", c.BaseURL, url.PathEscape(issueKey))
	req := AddCommentRequest{
		Body: NewADFDocument(commentBody),
	}

	body, err := c.Post(ctx, urlStr, req)
	if err != nil {
		return nil, fmt.Errorf("adding comment: %w", err)
	}

	var comment Comment
	if err := json.Unmarshal(body, &comment); err != nil {
		return nil, fmt.Errorf("parsing comment: %w", err)
	}

	return &comment, nil
}

// DeleteComment deletes a comment from an issue
func (c *Client) DeleteComment(ctx context.Context, issueKey, commentID string) error {
	if issueKey == "" {
		return ErrIssueKeyRequired
	}
	if commentID == "" {
		return ErrCommentIDRequired
	}

	urlStr := fmt.Sprintf("%s/issue/%s/comment/%s", c.BaseURL, url.PathEscape(issueKey), url.PathEscape(commentID))
	_, err := c.Delete(ctx, urlStr)
	if err != nil {
		return fmt.Errorf("deleting comment %s: %w", commentID, err)
	}
	return nil
}
