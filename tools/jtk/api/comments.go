package api

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
)

// GetComments returns comments for an issue
func (c *Client) GetComments(issueKey string, startAt, maxResults int) (*CommentsResponse, error) {
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
	body, err := c.get(urlStr)
	if err != nil {
		return nil, err
	}

	var result CommentsResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse comments: %w", err)
	}

	return &result, nil
}

// AddComment adds a comment to an issue
func (c *Client) AddComment(issueKey, commentBody string) (*Comment, error) {
	if issueKey == "" {
		return nil, ErrIssueKeyRequired
	}

	urlStr := fmt.Sprintf("%s/issue/%s/comment", c.BaseURL, url.PathEscape(issueKey))
	req := AddCommentRequest{
		Body: NewADFDocument(commentBody),
	}

	body, err := c.post(urlStr, req)
	if err != nil {
		return nil, err
	}

	var comment Comment
	if err := json.Unmarshal(body, &comment); err != nil {
		return nil, fmt.Errorf("failed to parse comment: %w", err)
	}

	return &comment, nil
}

// DeleteComment deletes a comment from an issue
func (c *Client) DeleteComment(issueKey, commentID string) error {
	if issueKey == "" {
		return ErrIssueKeyRequired
	}
	if commentID == "" {
		return fmt.Errorf("comment ID is required")
	}

	urlStr := fmt.Sprintf("%s/issue/%s/comment/%s", c.BaseURL, url.PathEscape(issueKey), url.PathEscape(commentID))
	_, err := c.delete(urlStr)
	return err
}
