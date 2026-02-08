package api

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
)

// ProjectDetail represents detailed project information
type ProjectDetail struct {
	ID             json.Number `json:"id"`
	Key            string      `json:"key"`
	Name           string      `json:"name"`
	Description    string      `json:"description,omitempty"`
	ProjectTypeKey string      `json:"projectTypeKey,omitempty"`
	Lead           *User       `json:"lead,omitempty"`
	IssueTypes     []IssueType `json:"issueTypes,omitempty"`
	Components     []Component `json:"components,omitempty"`
	URL            string      `json:"url,omitempty"`
}

// ProjectSearchResponse represents the paginated response from project search
type ProjectSearchResponse struct {
	MaxResults int             `json:"maxResults"`
	StartAt    int             `json:"startAt"`
	Total      int             `json:"total"`
	IsLast     bool            `json:"isLast"`
	Values     []ProjectDetail `json:"values"`
}

// CreateProjectRequest represents a request to create a project
type CreateProjectRequest struct {
	Key            string `json:"key"`
	Name           string `json:"name"`
	ProjectTypeKey string `json:"projectTypeKey"`
	LeadAccountID  string `json:"leadAccountId"`
	Description    string `json:"description,omitempty"`
}

// UpdateProjectRequest represents a request to update a project
type UpdateProjectRequest struct {
	Name           string `json:"name,omitempty"`
	Key            string `json:"key,omitempty"`
	Description    string `json:"description,omitempty"`
	LeadAccountID  string `json:"leadAccountId,omitempty"`
	ProjectTypeKey string `json:"projectTypeKey,omitempty"`
}

// ProjectType represents an available project type
type ProjectType struct {
	Key                string `json:"key"`
	FormattedKey       string `json:"formattedKey"`
	DescriptionI18nKey string `json:"descriptionI18nKey"`
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

// SearchProjects searches for projects with pagination
func (c *Client) SearchProjects(query string, startAt, maxResults int) (*ProjectSearchResponse, error) {
	params := map[string]string{}

	if query != "" {
		params["query"] = query
	}
	if startAt > 0 {
		params["startAt"] = strconv.Itoa(startAt)
	}
	if maxResults > 0 {
		params["maxResults"] = strconv.Itoa(maxResults)
	}

	urlStr := buildURL(fmt.Sprintf("%s/project/search", c.BaseURL), params)
	body, err := c.get(urlStr)
	if err != nil {
		return nil, err
	}

	var result ProjectSearchResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse project search results: %w", err)
	}

	return &result, nil
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

// CreateProject creates a new project
func (c *Client) CreateProject(req *CreateProjectRequest) (*ProjectDetail, error) {
	urlStr := fmt.Sprintf("%s/project", c.BaseURL)
	body, err := c.post(urlStr, req)
	if err != nil {
		return nil, err
	}

	var project ProjectDetail
	if err := json.Unmarshal(body, &project); err != nil {
		return nil, fmt.Errorf("failed to parse created project: %w", err)
	}

	return &project, nil
}

// UpdateProject updates an existing project
func (c *Client) UpdateProject(projectKeyOrID string, req *UpdateProjectRequest) (*ProjectDetail, error) {
	if projectKeyOrID == "" {
		return nil, ErrProjectKeyRequired
	}

	urlStr := fmt.Sprintf("%s/project/%s", c.BaseURL, url.PathEscape(projectKeyOrID))
	body, err := c.put(urlStr, req)
	if err != nil {
		return nil, err
	}

	var project ProjectDetail
	if err := json.Unmarshal(body, &project); err != nil {
		return nil, fmt.Errorf("failed to parse updated project: %w", err)
	}

	return &project, nil
}

// DeleteProject soft-deletes a project (moves to trash)
func (c *Client) DeleteProject(projectKeyOrID string) error {
	if projectKeyOrID == "" {
		return ErrProjectKeyRequired
	}

	urlStr := fmt.Sprintf("%s/project/%s", c.BaseURL, url.PathEscape(projectKeyOrID))
	_, err := c.delete(urlStr)
	return err
}

// RestoreProject restores a project from the trash
func (c *Client) RestoreProject(projectKeyOrID string) (*ProjectDetail, error) {
	if projectKeyOrID == "" {
		return nil, ErrProjectKeyRequired
	}

	urlStr := fmt.Sprintf("%s/project/%s/restore", c.BaseURL, url.PathEscape(projectKeyOrID))
	body, err := c.post(urlStr, nil)
	if err != nil {
		return nil, err
	}

	var project ProjectDetail
	if err := json.Unmarshal(body, &project); err != nil {
		return nil, fmt.Errorf("failed to parse restored project: %w", err)
	}

	return &project, nil
}

// ListProjectTypes returns available project types
func (c *Client) ListProjectTypes() ([]ProjectType, error) {
	urlStr := fmt.Sprintf("%s/project/type", c.BaseURL)
	body, err := c.get(urlStr)
	if err != nil {
		return nil, err
	}

	var types []ProjectType
	if err := json.Unmarshal(body, &types); err != nil {
		return nil, fmt.Errorf("failed to parse project types: %w", err)
	}

	return types, nil
}
