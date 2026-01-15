package api

import (
	"encoding/json"
	"fmt"
	"strconv"
)

// ListSprints returns sprints for a board
func (c *Client) ListSprints(boardID int, state string, startAt, maxResults int) (*SprintsResponse, error) {
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
	body, err := c.get(urlStr)
	if err != nil {
		return nil, err
	}

	var result SprintsResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse sprints: %w", err)
	}

	return &result, nil
}

// GetSprint retrieves a sprint by ID
func (c *Client) GetSprint(sprintID int) (*Sprint, error) {
	urlStr := fmt.Sprintf("%s/sprint/%d", c.AgileURL, sprintID)
	body, err := c.get(urlStr)
	if err != nil {
		return nil, err
	}

	var sprint Sprint
	if err := json.Unmarshal(body, &sprint); err != nil {
		return nil, fmt.Errorf("failed to parse sprint: %w", err)
	}

	return &sprint, nil
}

// GetSprintIssues returns issues in a sprint
func (c *Client) GetSprintIssues(sprintID int, startAt, maxResults int) (*SearchResult, error) {
	params := map[string]string{}

	if startAt > 0 {
		params["startAt"] = strconv.Itoa(startAt)
	}
	if maxResults > 0 {
		params["maxResults"] = strconv.Itoa(maxResults)
	}

	urlStr := buildURL(fmt.Sprintf("%s/sprint/%d/issue", c.AgileURL, sprintID), params)
	body, err := c.get(urlStr)
	if err != nil {
		return nil, err
	}

	var result SearchResult
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse sprint issues: %w", err)
	}

	return &result, nil
}

// GetCurrentSprint returns the active sprint for a board
func (c *Client) GetCurrentSprint(boardID int) (*Sprint, error) {
	result, err := c.ListSprints(boardID, "active", 0, 1)
	if err != nil {
		return nil, err
	}

	if len(result.Values) == 0 {
		return nil, fmt.Errorf("no active sprint found for board %d", boardID)
	}

	return &result.Values[0], nil
}

// MoveIssuesToSprint moves issues to a sprint
func (c *Client) MoveIssuesToSprint(sprintID int, issueKeys []string) error {
	urlStr := fmt.Sprintf("%s/sprint/%d/issue", c.AgileURL, sprintID)
	req := map[string]interface{}{
		"issues": issueKeys,
	}

	_, err := c.post(urlStr, req)
	return err
}
