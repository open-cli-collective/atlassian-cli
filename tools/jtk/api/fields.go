package api

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// GetFields returns all field definitions
func (c *Client) GetFields() ([]Field, error) {
	urlStr := fmt.Sprintf("%s/field", c.BaseURL)
	body, err := c.get(urlStr)
	if err != nil {
		return nil, err
	}

	var fields []Field
	if err := json.Unmarshal(body, &fields); err != nil {
		return nil, fmt.Errorf("failed to parse fields: %w", err)
	}

	return fields, nil
}

// GetCustomFields returns only custom field definitions
func (c *Client) GetCustomFields() ([]Field, error) {
	fields, err := c.GetFields()
	if err != nil {
		return nil, err
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

// FormatFieldValue formats a field value based on its type for the Jira API.
// It handles special cases like:
//   - option fields: wraps value as {"value": "..."}
//   - array fields: wraps value as [{"value": "..."}] or []string{...}
//   - user fields: wraps value as {"accountId": "..."}
//   - number fields: converts string to float64
//   - issuelink fields (e.g., parent): wraps value as {"key": "..."} or {"id": "..."}
//   - textarea custom fields: converts to ADF document
func FormatFieldValue(field *Field, value string) interface{} {
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
func (c *Client) GetFieldOptions(fieldID string) ([]FieldOptionValue, error) {
	if fieldID == "" {
		return nil, fmt.Errorf("field ID is required")
	}

	urlStr := fmt.Sprintf("%s/field/%s/context/defaultValue", c.BaseURL, fieldID)
	body, err := c.get(urlStr)
	if err != nil {
		urlStr = fmt.Sprintf("%s/field/%s/option", c.BaseURL, fieldID)
		body, err = c.get(urlStr)
		if err != nil {
			return nil, err
		}
	}

	var result FieldOptionsResponse
	if err := json.Unmarshal(body, &result); err != nil {
		var options []FieldOptionValue
		if err2 := json.Unmarshal(body, &options); err2 != nil {
			return nil, fmt.Errorf("failed to parse field options: %w", err)
		}
		return options, nil
	}

	return result.Options, nil
}

// GetFieldOptionsFromEditMeta returns allowed values for a field from issue edit metadata
func (c *Client) GetFieldOptionsFromEditMeta(issueKey, fieldID string) ([]FieldOptionValue, error) {
	meta, err := c.GetIssueEditMeta(issueKey)
	if err != nil {
		return nil, err
	}

	fieldsData, ok := meta["fields"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("no fields found in edit metadata")
	}

	fieldData, ok := fieldsData[fieldID].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("field %s not found in edit metadata", fieldID)
	}

	allowedValues, ok := fieldData["allowedValues"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("no allowed values found for field %s", fieldID)
	}

	var options []FieldOptionValue
	for _, av := range allowedValues {
		if opt, ok := av.(map[string]interface{}); ok {
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
