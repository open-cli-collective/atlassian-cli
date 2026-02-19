package api //nolint:revive // package name is intentional

import (
	"context"
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
func (c *Client) CreateField(ctx context.Context, req *CreateFieldRequest) (*Field, error) {
	urlStr := fmt.Sprintf("%s/field", c.BaseURL)
	body, err := c.Post(ctx, urlStr, req)
	if err != nil {
		return nil, fmt.Errorf("creating field: %w", err)
	}

	var field Field
	if err := json.Unmarshal(body, &field); err != nil {
		return nil, fmt.Errorf("parsing created field: %w", err)
	}

	return &field, nil
}

// TrashField moves a custom field to the trash (soft delete)
func (c *Client) TrashField(ctx context.Context, fieldID string) error {
	if fieldID == "" {
		return ErrFieldIDRequired
	}

	urlStr := fmt.Sprintf("%s/field/%s/trash", c.BaseURL, url.PathEscape(fieldID))
	_, err := c.Post(ctx, urlStr, nil)
	if err != nil {
		return fmt.Errorf("trashing field %s: %w", fieldID, err)
	}
	return nil
}

// RestoreField restores a custom field from the trash
func (c *Client) RestoreField(ctx context.Context, fieldID string) error {
	if fieldID == "" {
		return ErrFieldIDRequired
	}

	urlStr := fmt.Sprintf("%s/field/%s/restore", c.BaseURL, url.PathEscape(fieldID))
	_, err := c.Post(ctx, urlStr, nil)
	if err != nil {
		return fmt.Errorf("restoring field %s: %w", fieldID, err)
	}
	return nil
}

// GetFieldContexts returns the contexts for a custom field
func (c *Client) GetFieldContexts(ctx context.Context, fieldID string) (*FieldContextsResponse, error) {
	if fieldID == "" {
		return nil, ErrFieldIDRequired
	}

	urlStr := fmt.Sprintf("%s/field/%s/context", c.BaseURL, url.PathEscape(fieldID))
	body, err := c.Get(ctx, urlStr)
	if err != nil {
		return nil, fmt.Errorf("fetching field contexts: %w", err)
	}

	var result FieldContextsResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parsing field contexts: %w", err)
	}

	return &result, nil
}

// GetDefaultFieldContext returns the first context for a field.
// Used when --context is omitted to auto-detect the default context.
func (c *Client) GetDefaultFieldContext(ctx context.Context, fieldID string) (*FieldContext, error) {
	result, err := c.GetFieldContexts(ctx, fieldID)
	if err != nil {
		return nil, fmt.Errorf("getting default field context for %s: %w", fieldID, err)
	}

	if len(result.Values) == 0 {
		return nil, fmt.Errorf("no contexts found for field %s", fieldID)
	}

	return &result.Values[0], nil
}

// CreateFieldContext creates a new context for a custom field
func (c *Client) CreateFieldContext(ctx context.Context, fieldID string, req *CreateFieldContextRequest) (*FieldContext, error) {
	if fieldID == "" {
		return nil, ErrFieldIDRequired
	}

	urlStr := fmt.Sprintf("%s/field/%s/context", c.BaseURL, url.PathEscape(fieldID))
	body, err := c.Post(ctx, urlStr, req)
	if err != nil {
		return nil, fmt.Errorf("creating field context: %w", err)
	}

	var result FieldContext
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parsing created field context: %w", err)
	}

	return &result, nil
}

// DeleteFieldContext deletes a field context
func (c *Client) DeleteFieldContext(ctx context.Context, fieldID, contextID string) error {
	if fieldID == "" {
		return ErrFieldIDRequired
	}

	urlStr := fmt.Sprintf("%s/field/%s/context/%s", c.BaseURL, url.PathEscape(fieldID), url.PathEscape(contextID))
	_, err := c.Delete(ctx, urlStr)
	if err != nil {
		return fmt.Errorf("deleting field context: %w", err)
	}
	return nil
}

// GetFieldContextOptions returns the options for a field context
func (c *Client) GetFieldContextOptions(ctx context.Context, fieldID, contextID string) (*FieldContextOptionsResponse, error) {
	if fieldID == "" {
		return nil, ErrFieldIDRequired
	}

	urlStr := fmt.Sprintf("%s/field/%s/context/%s/option", c.BaseURL, url.PathEscape(fieldID), url.PathEscape(contextID))
	body, err := c.Get(ctx, urlStr)
	if err != nil {
		return nil, fmt.Errorf("fetching field context options: %w", err)
	}

	var result FieldContextOptionsResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parsing field context options: %w", err)
	}

	return &result, nil
}

// CreateFieldContextOptions creates new options in a field context
func (c *Client) CreateFieldContextOptions(ctx context.Context, fieldID, contextID string, req *CreateFieldContextOptionsRequest) ([]FieldContextOption, error) {
	if fieldID == "" {
		return nil, ErrFieldIDRequired
	}

	urlStr := fmt.Sprintf("%s/field/%s/context/%s/option", c.BaseURL, url.PathEscape(fieldID), url.PathEscape(contextID))
	body, err := c.Post(ctx, urlStr, req)
	if err != nil {
		return nil, fmt.Errorf("creating field context options: %w", err)
	}

	var result FieldContextOptionsResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parsing created field context options: %w", err)
	}

	return result.Values, nil
}

// UpdateFieldContextOptions updates existing options in a field context
func (c *Client) UpdateFieldContextOptions(ctx context.Context, fieldID, contextID string, req *UpdateFieldContextOptionsRequest) ([]FieldContextOption, error) {
	if fieldID == "" {
		return nil, ErrFieldIDRequired
	}

	urlStr := fmt.Sprintf("%s/field/%s/context/%s/option", c.BaseURL, url.PathEscape(fieldID), url.PathEscape(contextID))
	body, err := c.Put(ctx, urlStr, req)
	if err != nil {
		return nil, fmt.Errorf("updating field context options: %w", err)
	}

	var result FieldContextOptionsResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parsing updated field context options: %w", err)
	}

	return result.Values, nil
}

// DeleteFieldContextOption deletes an option from a field context
func (c *Client) DeleteFieldContextOption(ctx context.Context, fieldID, contextID, optionID string) error {
	if fieldID == "" {
		return ErrFieldIDRequired
	}

	urlStr := fmt.Sprintf("%s/field/%s/context/%s/option/%s", c.BaseURL, url.PathEscape(fieldID), url.PathEscape(contextID), url.PathEscape(optionID))
	_, err := c.Delete(ctx, urlStr)
	if err != nil {
		return fmt.Errorf("deleting field context option: %w", err)
	}
	return nil
}
