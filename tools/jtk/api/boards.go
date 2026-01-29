package api

import (
	"encoding/json"
	"fmt"
	"strconv"
)

// ListBoards returns boards, optionally filtered by project
func (c *Client) ListBoards(projectKeyOrID string, startAt, maxResults int) (*BoardsResponse, error) {
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
	body, err := c.get(urlStr)
	if err != nil {
		return nil, err
	}

	var result BoardsResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse boards: %w", err)
	}

	return &result, nil
}

// GetBoard retrieves a board by ID
func (c *Client) GetBoard(boardID int) (*Board, error) {
	urlStr := fmt.Sprintf("%s/board/%d", c.AgileURL, boardID)
	body, err := c.get(urlStr)
	if err != nil {
		return nil, err
	}

	var board Board
	if err := json.Unmarshal(body, &board); err != nil {
		return nil, fmt.Errorf("failed to parse board: %w", err)
	}

	return &board, nil
}
