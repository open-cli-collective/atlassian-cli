package api

import (
	"encoding/json"
	"fmt"
	"net/url"
)

// ProjectDetail represents detailed project information
type ProjectDetail struct {
	ID          string      `json:"id"`
	Key         string      `json:"key"`
	Name        string      `json:"name"`
	Description string      `json:"description,omitempty"`
	Lead        *User       `json:"lead,omitempty"`
	IssueTypes  []IssueType `json:"issueTypes,omitempty"`
	Components  []Component `json:"components,omitempty"`
	URL         string      `json:"url,omitempty"`
}

// ListProjects returns all projects
func (c *Client) ListProjects() ([]Project, error) {
	urlStr := fmt.Sprintf("%s/project", c.BaseURL)
	body, err := c.get(urlStr)
	if err != nil {
		return nil, err
	}

	var projects []Project
	if err := json.Unmarshal(body, &projects); err != nil {
		return nil, fmt.Errorf("failed to parse projects: %w", err)
	}

	return projects, nil
}

// GetProject retrieves a project by key or ID
func (c *Client) GetProject(projectKeyOrID string) (*ProjectDetail, error) {
	if projectKeyOrID == "" {
		return nil, ErrProjectKeyRequired
	}

	urlStr := fmt.Sprintf("%s/project/%s", c.BaseURL, url.PathEscape(projectKeyOrID))
	body, err := c.get(urlStr)
	if err != nil {
		return nil, err
	}

	var project ProjectDetail
	if err := json.Unmarshal(body, &project); err != nil {
		return nil, fmt.Errorf("failed to parse project: %w", err)
	}

	return &project, nil
}
