package api //nolint:revive // package name is intentional

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
)

// ListSprints returns sprints for a board
func (c *Client) ListSprints(ctx context.Context, boardID int, state string, startAt, maxResults int) (*SprintsResponse, error) {
	params := map[string]string{}

	if state != "" {
		params["state"] = state
	}
	if startAt > 0 {
		params["startAt"] = strconv.Itoa(startAt)
	}
	if maxResults > 0 {
		params["maxResults"] = strconv.Itoa(maxResults)
	}

	urlStr := buildURL(fmt.Sprintf("%s/board/%d/sprint", c.AgileURL, boardID), params)
	body, err := c.Get(ctx, urlStr)
	if err != nil {
		return nil, fmt.Errorf("listing sprints: %w", err)
	}

	var result SprintsResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parsing sprints: %w", err)
	}

	return &result, nil
}

// GetSprint retrieves a sprint by ID
func (c *Client) GetSprint(ctx context.Context, sprintID int) (*Sprint, error) {
	urlStr := fmt.Sprintf("%s/sprint/%d", c.AgileURL, sprintID)
	body, err := c.Get(ctx, urlStr)
	if err != nil {
		return nil, fmt.Errorf("fetching sprint: %w", err)
	}

	var sprint Sprint
	if err := json.Unmarshal(body, &sprint); err != nil {
		return nil, fmt.Errorf("parsing sprint: %w", err)
	}

	return &sprint, nil
}

// GetSprintIssues returns issues in a sprint
func (c *Client) GetSprintIssues(ctx context.Context, sprintID int, startAt, maxResults int) (*SearchResult, error) {
	params := map[string]string{}

	if startAt > 0 {
		params["startAt"] = strconv.Itoa(startAt)
	}
	if maxResults > 0 {
		params["maxResults"] = strconv.Itoa(maxResults)
	}

	urlStr := buildURL(fmt.Sprintf("%s/sprint/%d/issue", c.AgileURL, sprintID), params)
	body, err := c.Get(ctx, urlStr)
	if err != nil {
		return nil, fmt.Errorf("fetching sprint issues: %w", err)
	}

	var result SearchResult
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parsing sprint issues: %w", err)
	}

	return &result, nil
}

// GetCurrentSprint returns the active sprint for a board
func (c *Client) GetCurrentSprint(ctx context.Context, boardID int) (*Sprint, error) {
	result, err := c.ListSprints(ctx, boardID, "active", 0, 1)
	if err != nil {
		return nil, fmt.Errorf("getting current sprint for board %d: %w", boardID, err)
	}

	if len(result.Values) == 0 {
		return nil, fmt.Errorf("no active sprint found for board %d", boardID)
	}

	return &result.Values[0], nil
}

// MoveIssuesToSprint moves issues to a sprint
func (c *Client) MoveIssuesToSprint(ctx context.Context, sprintID int, issueKeys []string) error {
	urlStr := fmt.Sprintf("%s/sprint/%d/issue", c.AgileURL, sprintID)
	req := map[string]any{
		"issues": issueKeys,
	}

	_, err := c.Post(ctx, urlStr, req)
	if err != nil {
		return fmt.Errorf("moving issues to sprint %d: %w", sprintID, err)
	}
	return nil
}
