package api

import (
	"encoding/json"
	"fmt"
)

// GetCurrentUser returns the currently authenticated user
func (c *Client) GetCurrentUser() (*User, error) {
	urlStr := fmt.Sprintf("%s/myself", c.BaseURL)
	body, err := c.get(urlStr)
	if err != nil {
		return nil, err
	}

	var user User
	if err := json.Unmarshal(body, &user); err != nil {
		return nil, fmt.Errorf("failed to parse user: %w", err)
	}

	return &user, nil
}

// GetUser returns a user by their account ID
func (c *Client) GetUser(accountID string) (*User, error) {
	params := map[string]string{
		"accountId": accountID,
	}
	urlStr := buildURL(fmt.Sprintf("%s/user", c.BaseURL), params)
	body, err := c.get(urlStr)
	if err != nil {
		return nil, err
	}

	var user User
	if err := json.Unmarshal(body, &user); err != nil {
		return nil, fmt.Errorf("failed to parse user: %w", err)
	}

	return &user, nil
}

// SearchUsers searches for users by query string
func (c *Client) SearchUsers(query string, maxResults int) ([]User, error) {
	params := map[string]string{
		"query": query,
	}
	if maxResults > 0 {
		params["maxResults"] = fmt.Sprintf("%d", maxResults)
	}

	urlStr := buildURL(fmt.Sprintf("%s/user/search", c.BaseURL), params)
	body, err := c.get(urlStr)
	if err != nil {
		return nil, err
	}

	var users []User
	if err := json.Unmarshal(body, &users); err != nil {
		return nil, fmt.Errorf("failed to parse users: %w", err)
	}

	return users, nil
}
