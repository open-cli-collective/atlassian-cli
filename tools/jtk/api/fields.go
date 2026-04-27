package api //nolint:revive // package name is intentional

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// GetFields returns all field definitions
func (c *Client) GetFields(ctx context.Context) ([]Field, error) {
	urlStr := fmt.Sprintf("%s/field", c.BaseURL)
	body, err := c.Get(ctx, urlStr)
	if err != nil {
		return nil, fmt.Errorf("fetching fields: %w", err)
	}

	var fields []Field
	if err := json.Unmarshal(body, &fields); err != nil {
		return nil, fmt.Errorf("parsing fields: %w", err)
	}

	return fields, nil
}

// GetCustomFields returns only custom field definitions
func (c *Client) GetCustomFields(ctx context.Context) ([]Field, error) {
	fields, err := c.GetFields(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting custom fields: %w", err)
	}

	var customFields []Field
	for _, f := range fields {
		if f.Custom {
			customFields = append(customFields, f)
		}
	}

	return customFields, nil
}

// FindFieldByName finds a field by name (case-insensitive)
func FindFieldByName(fields []Field, name string) *Field {
	nameLower := strings.ToLower(name)
	for i := range fields {
		if strings.ToLower(fields[i].Name) == nameLower {
			return &fields[i]
		}
	}
	return nil
}

// FindFieldByID finds a field by ID
func FindFieldByID(fields []Field, id string) *Field {
	for i := range fields {
		if fields[i].ID == id {
			return &fields[i]
		}
	}
	return nil
}

// ResolveFieldID resolves a field name or ID to its ID
func ResolveFieldID(fields []Field, nameOrID string) (string, error) {
	if f := FindFieldByID(fields, nameOrID); f != nil {
		return f.ID, nil
	}

	if f := FindFieldByName(fields, nameOrID); f != nil {
		return f.ID, nil
	}

	return "", fmt.Errorf("field not found: %s", nameOrID)
}

// IsNullValue returns true if the value represents an explicit null/clear intent.
// Accepted values: "none", "null", or empty string (case-insensitive, whitespace-trimmed).
func IsNullValue(v string) bool {
	lower := strings.ToLower(strings.TrimSpace(v))
	return lower == "none" || lower == "null" || lower == ""
}

// FormatFieldValue formats a field value based on its type for the Jira API.
// It handles special cases like:
//   - option fields: wraps value as {"value": "..."}
//   - array fields: wraps value as [{"value": "..."}] or []string{...}
//   - user fields: wraps value as {"accountId": "..."}
//   - number fields: converts string to float64
//   - issuelink fields (e.g., parent): wraps value as {"key": "..."} or {"id": "..."}
//   - textarea custom fields: converts to ADF document
func FormatFieldValue(field *Field, value string) any {
	if field == nil {
		return value
	}

	if field.Schema.Custom == "com.atlassian.jira.plugin.system.customfieldtypes:textarea" {
		return NewADFDocument(value)
	}

	// The parent field requires {"key": "..."} format but Jira reports an empty
	// schema type for it, so handle it before the type switch.
	if field.ID == "parent" {
		if _, err := strconv.Atoi(value); err == nil {
			return map[string]string{"id": value}
		}
		return map[string]string{"key": value}
	}

	switch field.Schema.Type {
	case "option":
		return map[string]string{"value": value}
	case "array":
		if field.Schema.Items == "option" {
			return []map[string]string{{"value": value}}
		}
		return []string{value}
	case "user":
		if IsNullValue(value) {
			return nil
		}
		return map[string]string{"accountId": value}
	case "number":
		if n, err := strconv.ParseFloat(value, 64); err == nil {
			return n
		}
		return value
	case "issuelink":
		// Issue link fields (e.g., parent) need {"key": "..."} or {"id": "..."} format
		if _, err := strconv.Atoi(value); err == nil {
			return map[string]string{"id": value}
		}
		return map[string]string{"key": value}
	case "priority", "resolution", "status", "issuetype", "securitylevel":
		if _, err := strconv.Atoi(value); err == nil {
			return map[string]string{"id": value}
		}
		return map[string]string{"name": value}
	default:
		return value
	}
}

// MergeFieldValues merges a new formatted field value into an existing one.
// For array fields (e.g., multi-checkbox, labels), this appends to the existing
// array. For non-array fields, the new value replaces the existing one.
func MergeFieldValues(existing, newVal any) any {
	// Try to merge []map[string]string (option arrays like multi-checkbox)
	if existArr, ok := existing.([]map[string]string); ok {
		if newArr, ok := newVal.([]map[string]string); ok {
			return append(existArr, newArr...)
		}
	}

	// Try to merge []string (string arrays like labels)
	if existArr, ok := existing.([]string); ok {
		if newArr, ok := newVal.([]string); ok {
			return append(existArr, newArr...)
		}
	}

	// Non-array field: new value wins
	return newVal
}

// FieldOptionsResponse represents the response from field options endpoint
type FieldOptionsResponse struct {
	Options []FieldOptionValue `json:"values"`
	Total   int                `json:"total"`
}

// FieldOptionValue represents a single field option value
type FieldOptionValue struct {
	ID       string `json:"id,omitempty"`
	Value    string `json:"value,omitempty"`
	Name     string `json:"name,omitempty"`
	Disabled bool   `json:"disabled,omitempty"`
}

// GetFieldOptions returns allowed values for a custom field
func (c *Client) GetFieldOptions(ctx context.Context, fieldID string) ([]FieldOptionValue, error) {
	if fieldID == "" {
		return nil, ErrFieldIDRequired
	}

	urlStr := fmt.Sprintf("%s/field/%s/context/defaultValue", c.BaseURL, fieldID)
	body, err := c.Get(ctx, urlStr)
	if err != nil {
		urlStr = fmt.Sprintf("%s/field/%s/option", c.BaseURL, fieldID)
		body, err = c.Get(ctx, urlStr)
		if err != nil {
			return nil, fmt.Errorf("fetching field options: %w", err)
		}
	}

	var result FieldOptionsResponse
	if err := json.Unmarshal(body, &result); err != nil {
		var options []FieldOptionValue
		if err2 := json.Unmarshal(body, &options); err2 != nil {
			return nil, fmt.Errorf("parsing field options: %w", err)
		}
		return options, nil
	}

	return result.Options, nil
}

// GetFieldOptionsFromEditMeta returns allowed values for a field from issue edit metadata
func (c *Client) GetFieldOptionsFromEditMeta(ctx context.Context, issueKey, fieldID string) ([]FieldOptionValue, error) {
	meta, err := c.GetIssueEditMeta(ctx, issueKey)
	if err != nil {
		return nil, fmt.Errorf("getting field options from edit metadata for %s: %w", issueKey, err)
	}

	fieldsData, ok := meta["fields"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("no fields found in edit metadata")
	}

	fieldData, ok := fieldsData[fieldID].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrFieldNotInEditMeta, fieldID)
	}

	allowedValues, ok := fieldData["allowedValues"].([]any)
	if !ok {
		return nil, fmt.Errorf("no allowed values found for field %s", fieldID)
	}

	var options []FieldOptionValue
	for _, av := range allowedValues {
		if opt, ok := av.(map[string]any); ok {
			option := FieldOptionValue{}
			if id, ok := opt["id"].(string); ok {
				option.ID = id
			}
			if value, ok := opt["value"].(string); ok {
				option.Value = value
			}
			if name, ok := opt["name"].(string); ok {
				option.Name = name
			}
			if disabled, ok := opt["disabled"].(bool); ok {
				option.Disabled = disabled
			}
			options = append(options, option)
		}
	}

	return options, nil
}

// ResolveFieldOptions returns allowed values for a field in the context of a
// specific issue. It uses a three-tier resolution strategy:
//  1. Edit metadata (reliable for editable fields, already project-scoped)
//  2. Default field context + context options (handles read-only custom fields)
//  3. Global field options (last resort)
func ResolveFieldOptions(ctx context.Context, c *Client, issueKey, fieldID string) ([]FieldOptionValue, error) {
	opts, editErr := c.GetFieldOptionsFromEditMeta(ctx, issueKey, fieldID)
	if editErr == nil {
		return opts, nil
	}
	if !errors.Is(editErr, ErrFieldNotInEditMeta) {
		return nil, fmt.Errorf("resolving field options for %s on %s: %w", fieldID, issueKey, editErr)
	}

	ctxResult, ctxErr := c.GetDefaultFieldContext(ctx, fieldID)
	if ctxErr == nil {
		allOpts, pageErr := getAllContextOptions(ctx, c, fieldID, ctxResult.ID)
		if pageErr != nil {
			return nil, fmt.Errorf("fetching context options for %s: %w", fieldID, pageErr)
		}
		if len(allOpts) > 0 {
			return allOpts, nil
		}
	}

	globalOpts, globalErr := c.GetFieldOptions(ctx, fieldID)
	if globalErr == nil && len(globalOpts) > 0 {
		return globalOpts, nil
	}

	return nil, fmt.Errorf("no options found for field %s on %s", fieldID, issueKey)
}

// ErrFieldNotInEditMeta is returned when a field is not present in the issue's
// edit metadata (i.e., it is not editable for that issue).
var ErrFieldNotInEditMeta = errors.New("field not found in edit metadata")

func getAllContextOptions(ctx context.Context, c *Client, fieldID, contextID string) ([]FieldOptionValue, error) {
	opts, err := c.GetAllFieldContextOptions(ctx, fieldID, contextID)
	if err != nil {
		return nil, err
	}
	result := make([]FieldOptionValue, len(opts))
	for i, o := range opts {
		result[i] = FieldOptionValue{ID: o.ID, Value: o.Value, Disabled: o.Disabled}
	}
	return result, nil
}
