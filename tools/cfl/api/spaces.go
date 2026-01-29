package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
)

// ListSpacesOptions contains options for listing spaces.
type ListSpacesOptions struct {
	Limit  int
	Cursor string
	Type   string   // global, personal
	Status string   // current, archived
	Keys   []string // Filter by space keys
}

// ListSpaces returns a list of spaces.
func (c *Client) ListSpaces(ctx context.Context, opts *ListSpacesOptions) (*PaginatedResponse[Space], error) {
	params := url.Values{}
	params.Set("limit", "25") // Default limit

	if opts != nil {
		if opts.Limit > 0 {
			params.Set("limit", strconv.Itoa(opts.Limit))
		}
		if opts.Cursor != "" {
			params.Set("cursor", opts.Cursor)
		}
		if opts.Type != "" {
			params.Set("type", opts.Type)
		}
		if opts.Status != "" {
			params.Set("status", opts.Status)
		}
		for _, key := range opts.Keys {
			params.Add("keys", key)
		}
	}

	path := "/api/v2/spaces?" + params.Encode()
	body, err := c.Get(ctx, path)
	if err != nil {
		return nil, err
	}

	var result PaginatedResponse[Space]
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse spaces response: %w", err)
	}

	return &result, nil
}

// GetSpace returns a single space by ID.
func (c *Client) GetSpace(ctx context.Context, spaceID string) (*Space, error) {
	path := fmt.Sprintf("/api/v2/spaces/%s", spaceID)
	body, err := c.Get(ctx, path)
	if err != nil {
		return nil, err
	}

	var space Space
	if err := json.Unmarshal(body, &space); err != nil {
		return nil, fmt.Errorf("failed to parse space response: %w", err)
	}

	return &space, nil
}

// GetSpaceByKey returns a space by its key.
func (c *Client) GetSpaceByKey(ctx context.Context, key string) (*Space, error) {
	opts := &ListSpacesOptions{
		Keys:  []string{key},
		Limit: 1,
	}
	result, err := c.ListSpaces(ctx, opts)
	if err != nil {
		return nil, err
	}

	if len(result.Results) == 0 {
		return nil, &ErrorResponse{
			StatusCode: 404,
			Message:    fmt.Sprintf("Space with key '%s' not found", key),
		}
	}

	return &result.Results[0], nil
}
