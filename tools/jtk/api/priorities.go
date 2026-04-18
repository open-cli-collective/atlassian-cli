package api //nolint:revive // package name is intentional

import (
	"context"
	"encoding/json"
	"fmt"
)

// ListPriorities returns all priorities defined in the instance.
// GET /rest/api/3/priority
func (c *Client) ListPriorities(ctx context.Context) ([]Priority, error) {
	urlStr := fmt.Sprintf("%s/priority", c.BaseURL)
	body, err := c.Get(ctx, urlStr)
	if err != nil {
		return nil, fmt.Errorf("fetching priorities: %w", err)
	}

	var priorities []Priority
	if err := json.Unmarshal(body, &priorities); err != nil {
		return nil, fmt.Errorf("parsing priorities: %w", err)
	}

	return priorities, nil
}
