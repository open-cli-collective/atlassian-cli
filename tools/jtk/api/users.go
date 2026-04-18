package api //nolint:revive // package name is intentional

import (
	"context"
	"encoding/json"
	"fmt"
)

// GetCurrentUser returns the currently authenticated user
func (c *Client) GetCurrentUser(ctx context.Context) (*User, error) {
	urlStr := fmt.Sprintf("%s/myself", c.BaseURL)
	body, err := c.Get(ctx, urlStr)
	if err != nil {
		return nil, fmt.Errorf("getting current user: %w", err)
	}

	var user User
	if err := json.Unmarshal(body, &user); err != nil {
		return nil, fmt.Errorf("parsing user: %w", err)
	}

	return &user, nil
}

// GetUser returns a user by their account ID
func (c *Client) GetUser(ctx context.Context, accountID string) (*User, error) {
	params := map[string]string{
		"accountId": accountID,
	}
	urlStr := buildURL(fmt.Sprintf("%s/user", c.BaseURL), params)
	body, err := c.Get(ctx, urlStr)
	if err != nil {
		return nil, fmt.Errorf("getting user %s: %w", accountID, err)
	}

	var user User
	if err := json.Unmarshal(body, &user); err != nil {
		return nil, fmt.Errorf("parsing user: %w", err)
	}

	return &user, nil
}

// ListUsersPage returns a page of users for bulk enumeration.
// Hits GET /rest/api/3/users with startAt and maxResults.
// Note: Jira caps total user enumeration at ~1000; callers stop paging once
// a page returns fewer than maxResults.
func (c *Client) ListUsersPage(ctx context.Context, startAt, maxResults int) ([]User, error) {
	params := map[string]string{}
	if startAt > 0 {
		params["startAt"] = fmt.Sprintf("%d", startAt)
	}
	if maxResults > 0 {
		params["maxResults"] = fmt.Sprintf("%d", maxResults)
	}

	urlStr := buildURL(fmt.Sprintf("%s/users", c.BaseURL), params)
	body, err := c.Get(ctx, urlStr)
	if err != nil {
		return nil, fmt.Errorf("listing users: %w", err)
	}

	var users []User
	if err := json.Unmarshal(body, &users); err != nil {
		return nil, fmt.Errorf("parsing users: %w", err)
	}

	return users, nil
}

// SearchUsers searches for users by query string
func (c *Client) SearchUsers(ctx context.Context, query string, maxResults int) ([]User, error) {
	params := map[string]string{
		"query": query,
	}
	if maxResults > 0 {
		params["maxResults"] = fmt.Sprintf("%d", maxResults)
	}

	urlStr := buildURL(fmt.Sprintf("%s/user/search", c.BaseURL), params)
	body, err := c.Get(ctx, urlStr)
	if err != nil {
		return nil, fmt.Errorf("searching users: %w", err)
	}

	var users []User
	if err := json.Unmarshal(body, &users); err != nil {
		return nil, fmt.Errorf("parsing users: %w", err)
	}

	return users, nil
}
