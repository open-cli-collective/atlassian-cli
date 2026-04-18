package api //nolint:revive // package name is intentional

import (
	"context"
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
	Versions       []Version   `json:"versions,omitempty"`
	Style          string      `json:"style,omitempty"`
	Simplified     *bool       `json:"simplified,omitempty"`
	IsPrivate      *bool       `json:"isPrivate,omitempty"`
	URL            string      `json:"url,omitempty"`
}

// Version represents a Jira project version. Only the fields needed for the
// `projects get` count-column rendering are decoded here; additional fields
// (releaseDate, released, archived, etc.) can be added when a version-focused
// command lands.
type Version struct {
	ID   string `json:"id"`
	Name string `json:"name"`
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
func (c *Client) ListProjects(ctx context.Context) ([]Project, error) {
	urlStr := fmt.Sprintf("%s/project", c.BaseURL)
	body, err := c.Get(ctx, urlStr)
	if err != nil {
		return nil, fmt.Errorf("listing projects: %w", err)
	}

	var projects []Project
	if err := json.Unmarshal(body, &projects); err != nil {
		return nil, fmt.Errorf("parsing projects: %w", err)
	}

	return projects, nil
}

// ProjectListExpand is the expand string that populates the extended-mode
// `jtk projects list --extended` columns (STYLE / ISSUE_TYPES / COMPONENTS).
// Style/simplified/isPrivate are top-level on /project/search and don't need
// expand. Kept as a package constant so commands can reuse it without
// hardcoding the wire format.
const ProjectListExpand = "description,lead,issueTypes,url,projectKeys"

// SearchProjects searches for projects with pagination. expand is passed
// through to the ?expand= query param untouched; callers decide what
// expansion they need based on the columns they render (API layer stays
// ignorant of presentation mode). Use ProjectListExpand for extended-mode
// list, or a narrower string like "lead" for default-mode. Empty expand
// sends no expand param.
func (c *Client) SearchProjects(ctx context.Context, query string, startAt, maxResults int, expand string) (*ProjectSearchResponse, error) {
	params := map[string]string{}
	if expand != "" {
		params["expand"] = expand
	}

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
	body, err := c.Get(ctx, urlStr)
	if err != nil {
		return nil, fmt.Errorf("searching projects: %w", err)
	}

	var result ProjectSearchResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parsing project search results: %w", err)
	}

	return &result, nil
}

// ProjectGetExpand is the expand string that populates `jtk projects get`
// default + extended output (component list, version count, style /
// simplified / isPrivate flags, description, lead, URL). Callers that only
// need a subset (e.g. `--id` wants nothing at all) should pass their own
// narrower string instead of this default.
const ProjectGetExpand = "description,lead,issueTypes,url,projectKeys,versions"

// GetProject retrieves a project by key or ID. expand is passed straight to
// ?expand= — callers choose their own expansion (API stays ignorant of
// presentation mode). Pass ProjectGetExpand for full `projects get` output,
// a narrower string for specific fields (e.g. "issueTypes" for
// `issues types`), or "" to skip expansion entirely (e.g. `--id` which only
// needs the canonical key).
func (c *Client) GetProject(ctx context.Context, projectKeyOrID, expand string) (*ProjectDetail, error) {
	if projectKeyOrID == "" {
		return nil, ErrProjectKeyRequired
	}

	params := map[string]string{}
	if expand != "" {
		params["expand"] = expand
	}
	urlStr := buildURL(
		fmt.Sprintf("%s/project/%s", c.BaseURL, url.PathEscape(projectKeyOrID)),
		params,
	)
	body, err := c.Get(ctx, urlStr)
	if err != nil {
		return nil, fmt.Errorf("fetching project: %w", err)
	}

	var project ProjectDetail
	if err := json.Unmarshal(body, &project); err != nil {
		return nil, fmt.Errorf("parsing project: %w", err)
	}

	return &project, nil
}

// CreateProject creates a new project
func (c *Client) CreateProject(ctx context.Context, req *CreateProjectRequest) (*ProjectDetail, error) {
	urlStr := fmt.Sprintf("%s/project", c.BaseURL)
	body, err := c.Post(ctx, urlStr, req)
	if err != nil {
		return nil, fmt.Errorf("creating project: %w", err)
	}

	var project ProjectDetail
	if err := json.Unmarshal(body, &project); err != nil {
		return nil, fmt.Errorf("parsing created project: %w", err)
	}

	return &project, nil
}

// UpdateProject updates an existing project
func (c *Client) UpdateProject(ctx context.Context, projectKeyOrID string, req *UpdateProjectRequest) (*ProjectDetail, error) {
	if projectKeyOrID == "" {
		return nil, ErrProjectKeyRequired
	}

	urlStr := fmt.Sprintf("%s/project/%s", c.BaseURL, url.PathEscape(projectKeyOrID))
	body, err := c.Put(ctx, urlStr, req)
	if err != nil {
		return nil, fmt.Errorf("updating project: %w", err)
	}

	var project ProjectDetail
	if err := json.Unmarshal(body, &project); err != nil {
		return nil, fmt.Errorf("parsing updated project: %w", err)
	}

	return &project, nil
}

// DeleteProject soft-deletes a project (moves to trash)
func (c *Client) DeleteProject(ctx context.Context, projectKeyOrID string) error {
	if projectKeyOrID == "" {
		return ErrProjectKeyRequired
	}

	urlStr := fmt.Sprintf("%s/project/%s", c.BaseURL, url.PathEscape(projectKeyOrID))
	_, err := c.Delete(ctx, urlStr)
	if err != nil {
		return fmt.Errorf("deleting project %s: %w", projectKeyOrID, err)
	}
	return nil
}

// RestoreProject restores a project from the trash
func (c *Client) RestoreProject(ctx context.Context, projectKeyOrID string) (*ProjectDetail, error) {
	if projectKeyOrID == "" {
		return nil, ErrProjectKeyRequired
	}

	urlStr := fmt.Sprintf("%s/project/%s/restore", c.BaseURL, url.PathEscape(projectKeyOrID))
	body, err := c.Post(ctx, urlStr, nil)
	if err != nil {
		return nil, fmt.Errorf("restoring project: %w", err)
	}

	var project ProjectDetail
	if err := json.Unmarshal(body, &project); err != nil {
		return nil, fmt.Errorf("parsing restored project: %w", err)
	}

	return &project, nil
}

// ListProjectTypes returns available project types
func (c *Client) ListProjectTypes(ctx context.Context) ([]ProjectType, error) {
	urlStr := fmt.Sprintf("%s/project/type", c.BaseURL)
	body, err := c.Get(ctx, urlStr)
	if err != nil {
		return nil, fmt.Errorf("fetching project types: %w", err)
	}

	var types []ProjectType
	if err := json.Unmarshal(body, &types); err != nil {
		return nil, fmt.Errorf("parsing project types: %w", err)
	}

	return types, nil
}
