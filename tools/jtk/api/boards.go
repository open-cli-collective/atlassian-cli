package api //nolint:revive // package name is intentional

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
)

// ListBoards returns boards, optionally filtered by project
func (c *Client) ListBoards(ctx context.Context, projectKeyOrID string, startAt, maxResults int) (*BoardsResponse, error) {
	params := map[string]string{}

	if projectKeyOrID != "" {
		params["projectKeyOrId"] = projectKeyOrID
	}
	if startAt > 0 {
		params["startAt"] = strconv.Itoa(startAt)
	}
	if maxResults > 0 {
		params["maxResults"] = strconv.Itoa(maxResults)
	}

	urlStr := buildURL(fmt.Sprintf("%s/board", c.AgileURL), params)
	body, err := c.Get(ctx, urlStr)
	if err != nil {
		return nil, fmt.Errorf("listing boards: %w", err)
	}

	var result BoardsResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parsing boards: %w", err)
	}

	return &result, nil
}

// GetBoardConfiguration retrieves the configuration (filter, columns) for a board.
func (c *Client) GetBoardConfiguration(ctx context.Context, boardID int) (*BoardConfiguration, error) {
	urlStr := fmt.Sprintf("%s/board/%d/configuration", c.AgileURL, boardID)
	body, err := c.Get(ctx, urlStr)
	if err != nil {
		return nil, fmt.Errorf("getting board %d configuration: %w", boardID, err)
	}

	var config BoardConfiguration
	if err := json.Unmarshal(body, &config); err != nil {
		return nil, fmt.Errorf("parsing board configuration: %w", err)
	}

	return &config, nil
}

// Filter represents a Jira saved filter.
type Filter struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// GetFilter retrieves a saved filter by ID.
func (c *Client) GetFilter(ctx context.Context, filterID string) (*Filter, error) {
	urlStr := fmt.Sprintf("%s/filter/%s", c.BaseURL, url.PathEscape(filterID))
	body, err := c.Get(ctx, urlStr)
	if err != nil {
		return nil, fmt.Errorf("getting filter %s: %w", filterID, err)
	}

	var f Filter
	if err := json.Unmarshal(body, &f); err != nil {
		return nil, fmt.Errorf("parsing filter: %w", err)
	}

	return &f, nil
}

// GetBoard retrieves a board by ID
func (c *Client) GetBoard(ctx context.Context, boardID int) (*Board, error) {
	urlStr := fmt.Sprintf("%s/board/%d", c.AgileURL, boardID)
	body, err := c.Get(ctx, urlStr)
	if err != nil {
		return nil, fmt.Errorf("getting board %d: %w", boardID, err)
	}

	var board Board
	if err := json.Unmarshal(body, &board); err != nil {
		return nil, fmt.Errorf("parsing board: %w", err)
	}

	return &board, nil
}
