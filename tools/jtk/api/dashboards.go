package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
)

// Dashboard represents a Jira dashboard
type Dashboard struct {
	ID          string      `json:"id"`
	Name        string      `json:"name"`
	Description string      `json:"description,omitempty"`
	Owner       *User       `json:"owner,omitempty"`
	View        string      `json:"view,omitempty"`
	IsFavourite bool        `json:"isFavourite,omitempty"`
	Popularity  int         `json:"popularity,omitempty"`
	EditPerm    []SharePerm `json:"editPermissions,omitempty"`
	SharePerm   []SharePerm `json:"sharePermissions,omitempty"`
}

// SharePerm represents a dashboard sharing permission
type SharePerm struct {
	Type string `json:"type"` // "global", "project", "group", etc.
}

// DashboardGadget represents a gadget on a dashboard
type DashboardGadget struct {
	ID       int                `json:"id"`
	Title    string             `json:"title"`
	ModuleID string             `json:"moduleKey,omitempty"`
	URI      string             `json:"uri,omitempty"`
	Color    string             `json:"color,omitempty"`
	Position DashboardGadgetPos `json:"position,omitempty"`
	Props    map[string]any     `json:"properties,omitempty"`
}

// DashboardGadgetPos represents the position of a gadget on a dashboard
type DashboardGadgetPos struct {
	Row    int `json:"row"`
	Column int `json:"column"`
}

// DashboardsResponse represents a paginated list of dashboards
type DashboardsResponse struct {
	StartAt    int         `json:"startAt"`
	MaxResults int         `json:"maxResults"`
	Total      int         `json:"total"`
	Dashboards []Dashboard `json:"dashboards"`
}

// DashboardGadgetsResponse represents a list of gadgets on a dashboard
type DashboardGadgetsResponse struct {
	Gadgets []DashboardGadget `json:"gadgets"`
}

// CreateDashboardRequest represents a request to create a dashboard
type CreateDashboardRequest struct {
	Name             string      `json:"name"`
	Description      string      `json:"description,omitempty"`
	EditPermissions  []SharePerm `json:"editPermissions"`
	SharePermissions []SharePerm `json:"sharePermissions"`
}

// AddDashboardGadgetRequest represents a request to add a gadget to a dashboard
type AddDashboardGadgetRequest struct {
	ModuleKey string              `json:"moduleKey,omitempty"`
	Title     string              `json:"title,omitempty"`
	Color     string              `json:"color,omitempty"`
	Position  *DashboardGadgetPos `json:"position,omitempty"`
	URI       string              `json:"uri,omitempty"`
}

// DashboardSearchResponse represents the response from dashboard search
type DashboardSearchResponse struct {
	StartAt    int         `json:"startAt"`
	MaxResults int         `json:"maxResults"`
	Total      int         `json:"total"`
	Values     []Dashboard `json:"values"`
}

// GetDashboards returns a paginated list of dashboards
func (c *Client) GetDashboards(startAt, maxResults int) (*DashboardsResponse, error) {
	params := map[string]string{}
	if startAt > 0 {
		params["startAt"] = strconv.Itoa(startAt)
	}
	if maxResults > 0 {
		params["maxResults"] = strconv.Itoa(maxResults)
	}

	urlStr := buildURL(fmt.Sprintf("%s/dashboard", c.BaseURL), params)

	body, err := c.Get(context.Background(), urlStr)
	if err != nil {
		return nil, err
	}

	var result DashboardsResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parsing dashboards: %w", err)
	}

	return &result, nil
}

// SearchDashboards searches for dashboards by name
func (c *Client) SearchDashboards(name string, maxResults int) (*DashboardSearchResponse, error) {
	params := map[string]string{}
	if name != "" {
		params["dashboardName"] = name
	}
	if maxResults > 0 {
		params["maxResults"] = strconv.Itoa(maxResults)
	}

	urlStr := buildURL(fmt.Sprintf("%s/dashboard/search", c.BaseURL), params)

	body, err := c.Get(context.Background(), urlStr)
	if err != nil {
		return nil, err
	}

	var result DashboardSearchResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parsing dashboard search: %w", err)
	}

	return &result, nil
}

// GetDashboard returns a dashboard by ID
func (c *Client) GetDashboard(dashboardID string) (*Dashboard, error) {
	if dashboardID == "" {
		return nil, fmt.Errorf("dashboard ID is required")
	}

	urlStr := fmt.Sprintf("%s/dashboard/%s", c.BaseURL, url.PathEscape(dashboardID))

	body, err := c.Get(context.Background(), urlStr)
	if err != nil {
		return nil, err
	}

	var dash Dashboard
	if err := json.Unmarshal(body, &dash); err != nil {
		return nil, fmt.Errorf("parsing dashboard: %w", err)
	}

	return &dash, nil
}

// CreateDashboard creates a new dashboard
func (c *Client) CreateDashboard(req CreateDashboardRequest) (*Dashboard, error) {
	urlStr := fmt.Sprintf("%s/dashboard", c.BaseURL)

	body, err := c.Post(context.Background(), urlStr, req)
	if err != nil {
		return nil, err
	}

	var dash Dashboard
	if err := json.Unmarshal(body, &dash); err != nil {
		return nil, fmt.Errorf("parsing dashboard: %w", err)
	}

	return &dash, nil
}

// DeleteDashboard deletes a dashboard by ID
func (c *Client) DeleteDashboard(dashboardID string) error {
	if dashboardID == "" {
		return fmt.Errorf("dashboard ID is required")
	}

	urlStr := fmt.Sprintf("%s/dashboard/%s", c.BaseURL, url.PathEscape(dashboardID))
	_, err := c.Delete(context.Background(), urlStr)
	return err
}

// GetDashboardGadgets returns the gadgets on a dashboard
func (c *Client) GetDashboardGadgets(dashboardID string) (*DashboardGadgetsResponse, error) {
	if dashboardID == "" {
		return nil, fmt.Errorf("dashboard ID is required")
	}

	urlStr := fmt.Sprintf("%s/dashboard/%s/gadget", c.BaseURL, url.PathEscape(dashboardID))

	body, err := c.Get(context.Background(), urlStr)
	if err != nil {
		return nil, err
	}

	var result DashboardGadgetsResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parsing gadgets: %w", err)
	}

	return &result, nil
}

// RemoveDashboardGadget removes a gadget from a dashboard
func (c *Client) RemoveDashboardGadget(dashboardID string, gadgetID int) error {
	if dashboardID == "" {
		return fmt.Errorf("dashboard ID is required")
	}

	urlStr := fmt.Sprintf("%s/dashboard/%s/gadget/%d", c.BaseURL, url.PathEscape(dashboardID), gadgetID)
	_, err := c.Delete(context.Background(), urlStr)
	return err
}

// AddDashboardGadget adds a gadget to a dashboard
func (c *Client) AddDashboardGadget(dashboardID string, req AddDashboardGadgetRequest) (*DashboardGadget, error) {
	if dashboardID == "" {
		return nil, fmt.Errorf("dashboard ID is required")
	}

	urlStr := fmt.Sprintf("%s/dashboard/%s/gadget", c.BaseURL, url.PathEscape(dashboardID))

	body, err := c.Post(context.Background(), urlStr, req)
	if err != nil {
		return nil, err
	}

	var gadget DashboardGadget
	if err := json.Unmarshal(body, &gadget); err != nil {
		return nil, fmt.Errorf("parsing gadget: %w", err)
	}

	return &gadget, nil
}
