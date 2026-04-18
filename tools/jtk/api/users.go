package api //nolint:revive // package name is intentional

import (
	"context"
	"encoding/json"
	"fmt"
)

// UserExtendedExpand is the canonical expand string that populates
// `--extended` user output (Groups / Application Roles size blocks).
// Callers pass this to GetCurrentUser / GetUser when they intend to render
// extended fields; pass "" otherwise to avoid the wasted payload.
const UserExtendedExpand = "groups,applicationRoles"

// GetCurrentUser returns the currently authenticated user. expand is passed
// verbatim to ?expand= — callers decide which expansions they need. Use
// UserExtendedExpand for --extended; "" for default / --id callers.
func (c *Client) GetCurrentUser(ctx context.Context, expand string) (*User, error) {
	params := map[string]string{}
	if expand != "" {
		params["expand"] = expand
	}
	urlStr := buildURL(fmt.Sprintf("%s/myself", c.BaseURL), params)
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

// GetUser returns a user by their account ID. expand is passed verbatim to
// ?expand= — callers supply their intent. Use UserExtendedExpand for
// --extended output; "" for default/--id where the Size blocks would be
// discarded. timeZone / locale on the returned user may be empty on
// instances that redact other-user personal information — presenters render
// them as `-` in that case.
func (c *Client) GetUser(ctx context.Context, accountID, expand string) (*User, error) {
	params := map[string]string{"accountId": accountID}
	if expand != "" {
		params["expand"] = expand
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

// SearchUsers searches for users by query string.
// startAt is the 0-based offset into the result set; callers passing 0 get the
// first page. /user/search does not return isLast, so callers infer terminal
// state from `len(users) < maxResults`.
func (c *Client) SearchUsers(ctx context.Context, query string, startAt, maxResults int) ([]User, error) {
	params := map[string]string{
		"query": query,
	}
	if startAt > 0 {
		params["startAt"] = fmt.Sprintf("%d", startAt)
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
