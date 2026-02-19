package api //nolint:revive // package name is intentional

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

// GetTransitions returns available transitions for an issue
func (c *Client) GetTransitions(ctx context.Context, issueKey string) ([]Transition, error) {
	return c.GetTransitionsWithFields(ctx, issueKey, false)
}

// GetTransitionsWithFields returns available transitions for an issue,
// optionally including field metadata (required fields, allowed values)
func (c *Client) GetTransitionsWithFields(ctx context.Context, issueKey string, includeFields bool) ([]Transition, error) {
	if issueKey == "" {
		return nil, ErrIssueKeyRequired
	}

	urlStr := fmt.Sprintf("%s/issue/%s/transitions", c.BaseURL, url.PathEscape(issueKey))
	if includeFields {
		urlStr += "?expand=transitions.fields"
	}

	body, err := c.Get(ctx, urlStr)
	if err != nil {
		return nil, fmt.Errorf("fetching transitions: %w", err)
	}

	var result TransitionsResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parsing transitions: %w", err)
	}

	return result.Transitions, nil
}

// DoTransition performs a transition on an issue with optional fields
func (c *Client) DoTransition(ctx context.Context, issueKey, transitionID string, fields map[string]any) error {
	if issueKey == "" {
		return ErrIssueKeyRequired
	}

	urlStr := fmt.Sprintf("%s/issue/%s/transitions", c.BaseURL, url.PathEscape(issueKey))
	req := TransitionRequest{
		Transition: TransitionID{ID: transitionID},
		Fields:     fields,
	}

	_, err := c.Post(ctx, urlStr, req)
	if err != nil {
		return fmt.Errorf("performing transition: %w", err)
	}
	return nil
}

// FindTransitionByName finds a transition by name (case-insensitive)
func FindTransitionByName(transitions []Transition, name string) *Transition {
	nameLower := strings.ToLower(name)
	for i := range transitions {
		if strings.ToLower(transitions[i].Name) == nameLower {
			return &transitions[i]
		}
	}
	return nil
}
