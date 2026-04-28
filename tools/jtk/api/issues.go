package api //nolint:revive // package name is intentional

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strings"
)

// GetIssue retrieves an issue by key
func (c *Client) GetIssue(ctx context.Context, issueKey string) (*Issue, error) {
	if issueKey == "" {
		return nil, ErrIssueKeyRequired
	}

	urlStr := fmt.Sprintf("%s/issue/%s", c.BaseURL, url.PathEscape(issueKey))
	body, err := c.Get(ctx, urlStr)
	if err != nil {
		return nil, fmt.Errorf("fetching issue: %w", err)
	}

	var issue Issue
	if err := json.Unmarshal(body, &issue); err != nil {
		return nil, fmt.Errorf("parsing issue: %w", err)
	}

	return &issue, nil
}

// CreateIssue creates a new issue
func (c *Client) CreateIssue(ctx context.Context, req *CreateIssueRequest) (*Issue, error) {
	urlStr := fmt.Sprintf("%s/issue", c.BaseURL)
	body, err := c.Post(ctx, urlStr, req)
	if err != nil {
		return nil, fmt.Errorf("creating issue: %w", err)
	}

	var issue Issue
	if err := json.Unmarshal(body, &issue); err != nil {
		return nil, fmt.Errorf("parsing created issue: %w", err)
	}

	return &issue, nil
}

// UpdateIssue updates an existing issue
func (c *Client) UpdateIssue(ctx context.Context, issueKey string, req *UpdateIssueRequest) error {
	if issueKey == "" {
		return ErrIssueKeyRequired
	}

	urlStr := fmt.Sprintf("%s/issue/%s", c.BaseURL, url.PathEscape(issueKey))
	_, err := c.Put(ctx, urlStr, req)
	if err != nil {
		return fmt.Errorf("updating issue %s: %w", issueKey, err)
	}
	return nil
}

// DeleteIssue deletes an issue
func (c *Client) DeleteIssue(ctx context.Context, issueKey string) error {
	if issueKey == "" {
		return ErrIssueKeyRequired
	}

	urlStr := fmt.Sprintf("%s/issue/%s", c.BaseURL, url.PathEscape(issueKey))
	_, err := c.Delete(ctx, urlStr)
	if err != nil {
		return fmt.Errorf("deleting issue %s: %w", issueKey, err)
	}
	return nil
}

// AssignIssue assigns an issue to a user
func (c *Client) AssignIssue(ctx context.Context, issueKey, accountID string) error {
	if issueKey == "" {
		return ErrIssueKeyRequired
	}

	urlStr := fmt.Sprintf("%s/issue/%s/assignee", c.BaseURL, url.PathEscape(issueKey))

	body := map[string]any{}
	if accountID != "" {
		body["accountId"] = accountID
	} else {
		// Setting to null unassigns the issue
		body["accountId"] = nil
	}

	_, err := c.Put(ctx, urlStr, body)
	if err != nil {
		return fmt.Errorf("assigning issue %s: %w", issueKey, err)
	}
	return nil
}

// GetIssueEditMeta returns the edit metadata for an issue
func (c *Client) GetIssueEditMeta(ctx context.Context, issueKey string) (map[string]any, error) {
	if issueKey == "" {
		return nil, ErrIssueKeyRequired
	}

	urlStr := fmt.Sprintf("%s/issue/%s/editmeta", c.BaseURL, url.PathEscape(issueKey))
	body, err := c.Get(ctx, urlStr)
	if err != nil {
		return nil, fmt.Errorf("fetching edit metadata: %w", err)
	}

	var result map[string]any
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parsing edit metadata: %w", err)
	}

	return result, nil
}

// BuildCreateRequest builds a create issue request
func BuildCreateRequest(projectKey, issueType, summary, description string, extraFields map[string]any) *CreateIssueRequest {
	fields := map[string]any{
		"project":   map[string]string{"key": projectKey},
		"issuetype": map[string]string{"name": issueType},
		"summary":   summary,
	}

	if description != "" {
		fields["description"] = NewADFDocument(description)
	}

	for k, v := range extraFields {
		fields[k] = v
	}

	return &CreateIssueRequest{Fields: fields}
}

// BuildUpdateRequest builds an update issue request
func BuildUpdateRequest(fields map[string]any) *UpdateIssueRequest {
	return &UpdateIssueRequest{Fields: fields}
}

// EditFieldMeta represents field metadata from issue edit metadata API.
type EditFieldMeta struct {
	ID       string
	Name     string
	Type     string // from schema.type
	Required bool
}

// ParseEditMeta extracts field metadata from raw edit metadata response.
// The input is the "fields" map from the edit metadata API response.
func ParseEditMeta(fieldsData map[string]any) []EditFieldMeta {
	result := make([]EditFieldMeta, 0, len(fieldsData))

	for id, data := range fieldsData {
		fieldData, ok := data.(map[string]any)
		if !ok {
			continue
		}

		name := safeString(fieldData["name"])
		required := false
		if req, ok := fieldData["required"].(bool); ok && req {
			required = true
		}

		fieldType := ""
		if schema, ok := fieldData["schema"].(map[string]any); ok {
			fieldType = safeString(schema["type"])
		}

		result = append(result, EditFieldMeta{
			ID:       id,
			Name:     name,
			Type:     fieldType,
			Required: required,
		})
	}

	return result
}

// ArchiveError represents a single error category from the archive response.
type ArchiveError struct {
	Count          int      `json:"count"`
	IssueIdsOrKeys []string `json:"issueIdsOrKeys"`
	Message        string   `json:"message"`
}

// ArchiveResult represents the response from archiving issues.
type ArchiveResult struct {
	NumberUpdated int                     `json:"numberOfIssuesUpdated"`
	Errors        map[string]ArchiveError `json:"errors,omitempty"`
}

// ArchiveIssues archives one or more issues by key.
func (c *Client) ArchiveIssues(ctx context.Context, keys []string) (*ArchiveResult, error) {
	if len(keys) == 0 {
		return nil, ErrIssueKeyRequired
	}

	urlStr := fmt.Sprintf("%s/issue/archive", c.BaseURL)
	body, err := c.Put(ctx, urlStr, map[string]any{
		"issueIdsOrKeys": keys,
	})
	if err != nil {
		return nil, fmt.Errorf("archiving issues: %w", err)
	}

	var result ArchiveResult
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parsing archive response: %w", err)
	}
	return &result, nil
}

// WatchersInfo represents the watchers summary for an issue.
type WatchersInfo struct {
	WatchCount int  `json:"watchCount"`
	IsWatching bool `json:"isWatching"`
}

// GetWatchers returns watchers info for an issue.
func (c *Client) GetWatchers(ctx context.Context, issueKey string) (*WatchersInfo, error) {
	if issueKey == "" {
		return nil, ErrIssueKeyRequired
	}

	urlStr := fmt.Sprintf("%s/issue/%s/watchers", c.BaseURL, url.PathEscape(issueKey))
	body, err := c.Get(ctx, urlStr)
	if err != nil {
		return nil, fmt.Errorf("fetching watchers: %w", err)
	}

	var info WatchersInfo
	if err := json.Unmarshal(body, &info); err != nil {
		return nil, fmt.Errorf("parsing watchers: %w", err)
	}
	return &info, nil
}

// IssueFieldEntry represents a single field's current value on an issue,
// used by `issues fields <key>` to display FIELD_ID|NAME|TYPE|VALUE.
type IssueFieldEntry struct {
	ID    string
	Name  string
	Type  string
	Value string
}

// ExtractIssueFieldValues builds a sorted slice of IssueFieldEntry from an
// issue's typed struct fields and its CustomFields map. Field metadata (name,
// type) comes from the fields cache.
func ExtractIssueFieldValues(issue *Issue, fields []Field) []IssueFieldEntry {
	fieldIndex := make(map[string]*Field, len(fields))
	for i := range fields {
		fieldIndex[fields[i].ID] = &fields[i]
	}

	var entries []IssueFieldEntry

	for id, extract := range knownFieldExtractors {
		val := extract(issue)
		if val == "" {
			continue
		}
		name, typ := id, ""
		if f := fieldIndex[id]; f != nil {
			name = f.Name
			typ = f.Schema.Type
		}
		entries = append(entries, IssueFieldEntry{ID: id, Name: name, Type: typ, Value: val})
	}

	for id, raw := range issue.Fields.CustomFields {
		if raw == nil {
			continue
		}
		val := FormatCustomFieldValue(raw)
		if val == "" {
			continue
		}
		name, typ := id, ""
		if f := fieldIndex[id]; f != nil {
			name = f.Name
			typ = f.Schema.Type
		}
		entries = append(entries, IssueFieldEntry{ID: id, Name: name, Type: typ, Value: val})
	}

	sort.Slice(entries, func(i, j int) bool { return entries[i].ID < entries[j].ID })
	return entries
}

// knownFieldExtractors maps typed IssueFields struct field IDs to value
// extractors. Must stay in sync with knownFieldKeys (enforced by test).
var knownFieldExtractors = map[string]func(*Issue) string{
	"summary": func(i *Issue) string { return i.Fields.Summary },
	"description": func(i *Issue) string {
		if i.Fields.Description == nil {
			return ""
		}
		t := i.Fields.Description.ToPlainText()
		runes := []rune(t)
		if len(runes) > 80 {
			return string(runes[:80]) + "..."
		}
		return t
	},
	"status": func(i *Issue) string {
		if i.Fields.Status == nil {
			return ""
		}
		return i.Fields.Status.Name
	},
	"issuetype": func(i *Issue) string {
		if i.Fields.IssueType == nil {
			return ""
		}
		return i.Fields.IssueType.Name
	},
	"priority": func(i *Issue) string {
		if i.Fields.Priority == nil {
			return ""
		}
		return i.Fields.Priority.Name
	},
	"assignee":   func(i *Issue) string { return displayNameOrEmpty(i.Fields.Assignee) },
	"reporter":   func(i *Issue) string { return displayNameOrEmpty(i.Fields.Reporter) },
	"project":    func(i *Issue) string { return projectKeyOrEmpty(i.Fields.Project) },
	"created":    func(i *Issue) string { return i.Fields.Created },
	"updated":    func(i *Issue) string { return i.Fields.Updated },
	"labels":     func(i *Issue) string { return strings.Join(i.Fields.Labels, ", ") },
	"components": func(i *Issue) string { return componentNames(i.Fields.Components) },
	"sprint": func(i *Issue) string {
		if i.Fields.Sprint == nil {
			return ""
		}
		return i.Fields.Sprint.Name
	},
	"parent": func(i *Issue) string {
		if i.Fields.Parent == nil {
			return ""
		}
		return i.Fields.Parent.Key
	},
	"resolution": func(i *Issue) string {
		if i.Fields.Resolution == nil {
			return ""
		}
		return i.Fields.Resolution.Name
	},
	"fixVersions": func(i *Issue) string { return versionNames(i.Fields.FixVersions) },
}

func displayNameOrEmpty(u *User) string {
	if u == nil {
		return ""
	}
	return u.DisplayName
}

func projectKeyOrEmpty(p *Project) string {
	if p == nil {
		return ""
	}
	return p.Key
}

func componentNames(cs []Component) string {
	if len(cs) == 0 {
		return ""
	}
	names := make([]string, len(cs))
	for i, c := range cs {
		names[i] = c.Name
	}
	return strings.Join(names, ", ")
}

func versionNames(vs []Version) string {
	if len(vs) == 0 {
		return ""
	}
	names := make([]string, len(vs))
	for i, v := range vs {
		names[i] = v.Name
	}
	return strings.Join(names, ", ")
}

// ExtractFieldValue returns a display-ready string for any Jira field on an issue.
// Typed struct fields (status, assignee, etc.) use knownFieldExtractors;
// everything else falls back to FormatCustomFieldValue on the CustomFields map.
func ExtractFieldValue(issue *Issue, fieldID string) string {
	if extract, ok := knownFieldExtractors[fieldID]; ok {
		return extract(issue)
	}
	if issue.Fields.CustomFields != nil {
		if raw, ok := issue.Fields.CustomFields[fieldID]; ok {
			return FormatCustomFieldValue(raw)
		}
	}
	return ""
}

// FormatCustomFieldValue formats an arbitrary custom field value as a display string.
func FormatCustomFieldValue(v any) string {
	if v == nil {
		return ""
	}
	switch val := v.(type) {
	case string:
		if strings.Contains(val, "={") {
			return ""
		}
		return val
	case float64:
		if val == float64(int64(val)) {
			return fmt.Sprintf("%d", int64(val))
		}
		return fmt.Sprintf("%g", val)
	case map[string]any:
		if s, ok := val["value"].(string); ok {
			return s
		}
		if s, ok := val["name"].(string); ok {
			return s
		}
		if s, ok := val["displayName"].(string); ok {
			return s
		}
		return ""
	case []any:
		parts := make([]string, 0, len(val))
		for _, item := range val {
			switch elem := item.(type) {
			case string:
				parts = append(parts, elem)
			case map[string]any:
				if s, ok := elem["value"].(string); ok {
					parts = append(parts, s)
				} else if s, ok := elem["name"].(string); ok {
					parts = append(parts, s)
				}
			}
		}
		return strings.Join(parts, ", ")
	case bool:
		if val {
			return "yes"
		}
		return "no"
	default:
		return ""
	}
}

// safeString extracts a string from an interface value.
func safeString(v any) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", v)
}
