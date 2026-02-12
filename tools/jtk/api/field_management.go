package api

import (
	"encoding/json"
	"fmt"
	"net/url"
)

// CreateFieldRequest represents a request to create a custom field
type CreateFieldRequest struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description,omitempty"`
	SearcherKey string `json:"searcherKey,omitempty"`
}

// FieldContext represents a custom field context
type FieldContext struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	Description     string `json:"description,omitempty"`
	IsGlobalContext bool   `json:"isGlobalContext"`
	IsAnyIssueType  bool   `json:"isAnyIssueType"`
}

// FieldContextsResponse represents the paginated response from listing contexts
type FieldContextsResponse struct {
	MaxResults int            `json:"maxResults"`
	StartAt    int            `json:"startAt"`
	Total      int            `json:"total"`
	IsLast     bool           `json:"isLast"`
	Values     []FieldContext `json:"values"`
}

// CreateFieldContextRequest represents a request to create a field context
type CreateFieldContextRequest struct {
	Name         string   `json:"name"`
	Description  string   `json:"description,omitempty"`
	ProjectIDs   []string `json:"projectIds,omitempty"`
	IssueTypeIDs []string `json:"issueTypeIds,omitempty"`
}

// FieldContextOption represents a single option in a context
type FieldContextOption struct {
	ID       string `json:"id"`
	Value    string `json:"value"`
	Disabled bool   `json:"disabled"`
}

// FieldContextOptionsResponse represents the paginated response from listing context options
type FieldContextOptionsResponse struct {
	MaxResults int                  `json:"maxResults"`
	StartAt    int                  `json:"startAt"`
	Total      int                  `json:"total"`
	IsLast     bool                 `json:"isLast"`
	Values     []FieldContextOption `json:"values"`
}

// CreateFieldContextOptionsRequest represents a request to create options
type CreateFieldContextOptionsRequest struct {
	Options []CreateFieldContextOptionEntry `json:"options"`
}

// CreateFieldContextOptionEntry represents a single option to create
type CreateFieldContextOptionEntry struct {
	Value    string `json:"value"`
	Disabled bool   `json:"disabled,omitempty"`
}

// UpdateFieldContextOptionsRequest represents a request to update options
type UpdateFieldContextOptionsRequest struct {
	Options []UpdateFieldContextOptionEntry `json:"options"`
}

// UpdateFieldContextOptionEntry represents a single option to update
type UpdateFieldContextOptionEntry struct {
	ID       string `json:"id"`
	Value    string `json:"value,omitempty"`
	Disabled bool   `json:"disabled,omitempty"`
}

// CreateField creates a new custom field
func (c *Client) CreateField(req *CreateFieldRequest) (*Field, error) {
	urlStr := fmt.Sprintf("%s/field", c.BaseURL)
	body, err := c.post(urlStr, req)
	if err != nil {
		return nil, err
	}

	var field Field
	if err := json.Unmarshal(body, &field); err != nil {
		return nil, fmt.Errorf("failed to parse created field: %w", err)
	}

	return &field, nil
}

// TrashField moves a custom field to the trash (soft delete)
func (c *Client) TrashField(fieldID string) error {
	if fieldID == "" {
		return ErrFieldIDRequired
	}

	urlStr := fmt.Sprintf("%s/field/%s/trash", c.BaseURL, url.PathEscape(fieldID))
	_, err := c.post(urlStr, nil)
	return err
}

// RestoreField restores a custom field from the trash
func (c *Client) RestoreField(fieldID string) error {
	if fieldID == "" {
		return ErrFieldIDRequired
	}

	urlStr := fmt.Sprintf("%s/field/%s/restore", c.BaseURL, url.PathEscape(fieldID))
	_, err := c.post(urlStr, nil)
	return err
}

// GetFieldContexts returns the contexts for a custom field
func (c *Client) GetFieldContexts(fieldID string) (*FieldContextsResponse, error) {
	if fieldID == "" {
		return nil, ErrFieldIDRequired
	}

	urlStr := fmt.Sprintf("%s/field/%s/context", c.BaseURL, url.PathEscape(fieldID))
	body, err := c.get(urlStr)
	if err != nil {
		return nil, err
	}

	var result FieldContextsResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse field contexts: %w", err)
	}

	return &result, nil
}

// GetDefaultFieldContext returns the first context for a field.
// Used when --context is omitted to auto-detect the default context.
func (c *Client) GetDefaultFieldContext(fieldID string) (*FieldContext, error) {
	result, err := c.GetFieldContexts(fieldID)
	if err != nil {
		return nil, err
	}

	if len(result.Values) == 0 {
		return nil, fmt.Errorf("no contexts found for field %s", fieldID)
	}

	return &result.Values[0], nil
}

// CreateFieldContext creates a new context for a custom field
func (c *Client) CreateFieldContext(fieldID string, req *CreateFieldContextRequest) (*FieldContext, error) {
	if fieldID == "" {
		return nil, ErrFieldIDRequired
	}

	urlStr := fmt.Sprintf("%s/field/%s/context", c.BaseURL, url.PathEscape(fieldID))
	body, err := c.post(urlStr, req)
	if err != nil {
		return nil, err
	}

	var result FieldContext
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse created field context: %w", err)
	}

	return &result, nil
}

// DeleteFieldContext deletes a field context
func (c *Client) DeleteFieldContext(fieldID, contextID string) error {
	if fieldID == "" {
		return ErrFieldIDRequired
	}

	urlStr := fmt.Sprintf("%s/field/%s/context/%s", c.BaseURL, url.PathEscape(fieldID), url.PathEscape(contextID))
	_, err := c.delete(urlStr)
	return err
}

// GetFieldContextOptions returns the options for a field context
func (c *Client) GetFieldContextOptions(fieldID, contextID string) (*FieldContextOptionsResponse, error) {
	if fieldID == "" {
		return nil, ErrFieldIDRequired
	}

	urlStr := fmt.Sprintf("%s/field/%s/context/%s/option", c.BaseURL, url.PathEscape(fieldID), url.PathEscape(contextID))
	body, err := c.get(urlStr)
	if err != nil {
		return nil, err
	}

	var result FieldContextOptionsResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse field context options: %w", err)
	}

	return &result, nil
}

// CreateFieldContextOptions creates new options in a field context
func (c *Client) CreateFieldContextOptions(fieldID, contextID string, req *CreateFieldContextOptionsRequest) ([]FieldContextOption, error) {
	if fieldID == "" {
		return nil, ErrFieldIDRequired
	}

	urlStr := fmt.Sprintf("%s/field/%s/context/%s/option", c.BaseURL, url.PathEscape(fieldID), url.PathEscape(contextID))
	body, err := c.post(urlStr, req)
	if err != nil {
		return nil, err
	}

	var result FieldContextOptionsResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse created field context options: %w", err)
	}

	return result.Values, nil
}

// UpdateFieldContextOptions updates existing options in a field context
func (c *Client) UpdateFieldContextOptions(fieldID, contextID string, req *UpdateFieldContextOptionsRequest) ([]FieldContextOption, error) {
	if fieldID == "" {
		return nil, ErrFieldIDRequired
	}

	urlStr := fmt.Sprintf("%s/field/%s/context/%s/option", c.BaseURL, url.PathEscape(fieldID), url.PathEscape(contextID))
	body, err := c.put(urlStr, req)
	if err != nil {
		return nil, err
	}

	var result FieldContextOptionsResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse updated field context options: %w", err)
	}

	return result.Values, nil
}

// DeleteFieldContextOption deletes an option from a field context
func (c *Client) DeleteFieldContextOption(fieldID, contextID, optionID string) error {
	if fieldID == "" {
		return ErrFieldIDRequired
	}

	urlStr := fmt.Sprintf("%s/field/%s/context/%s/option/%s", c.BaseURL, url.PathEscape(fieldID), url.PathEscape(contextID), url.PathEscape(optionID))
	_, err := c.delete(urlStr)
	return err
}
