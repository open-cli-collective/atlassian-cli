package api //nolint:revive // package name is intentional

import (
	"context"
	"encoding/json"
	"fmt"
)

// ListResolutions returns all resolutions defined in the instance.
// GET /rest/api/3/resolution
func (c *Client) ListResolutions(ctx context.Context) ([]Resolution, error) {
	urlStr := fmt.Sprintf("%s/resolution", c.BaseURL)
	body, err := c.Get(ctx, urlStr)
	if err != nil {
		return nil, fmt.Errorf("fetching resolutions: %w", err)
	}

	var resolutions []Resolution
	if err := json.Unmarshal(body, &resolutions); err != nil {
		return nil, fmt.Errorf("parsing resolutions: %w", err)
	}

	return resolutions, nil
}
