package api

import (
	"encoding/json"
	"time"
)

// Issue represents a Jira issue
type Issue struct {
	ID     string      `json:"id"`
	Key    string      `json:"key"`
	Self   string      `json:"self"`
	Fields IssueFields `json:"fields"`
}

// IssueFields contains the fields of a Jira issue
type IssueFields struct {
	Summary     string       `json:"summary"`
	Description *Description `json:"description,omitempty"`
	Status      *Status      `json:"status,omitempty"`
	IssueType   *IssueType   `json:"issuetype,omitempty"`
	Priority    *Priority    `json:"priority,omitempty"`
	Assignee    *User        `json:"assignee,omitempty"`
	Reporter    *User        `json:"reporter,omitempty"`
	Project     *Project     `json:"project,omitempty"`
	Created     string       `json:"created,omitempty"`
	Updated     string       `json:"updated,omitempty"`
	Labels      []string     `json:"labels,omitempty"`
	Components  []Component  `json:"components,omitempty"`
	Sprint      *Sprint      `json:"sprint,omitempty"`
	Parent      *Issue       `json:"parent,omitempty"`

	// CustomFields holds any fields not mapped to struct fields (e.g., customfield_10001)
	CustomFields map[string]interface{} `json:"-"`
}

// knownFieldKeys lists JSON keys for typed struct fields
var knownFieldKeys = map[string]bool{
	"summary": true, "description": true, "status": true,
	"issuetype": true, "priority": true, "assignee": true,
	"reporter": true, "project": true, "created": true,
	"updated": true, "labels": true, "components": true,
	"sprint": true, "parent": true,
}

// UnmarshalJSON custom unmarshaler to capture custom fields
func (f *IssueFields) UnmarshalJSON(data []byte) error {
	// First, unmarshal into a temp struct to get typed fields
	type Alias IssueFields
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(f),
	}
	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}

	// Then unmarshal into a map to capture all fields
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	// Extract custom fields (those not in knownFieldKeys)
	f.CustomFields = make(map[string]interface{})
	for key, value := range raw {
		if !knownFieldKeys[key] {
			var v interface{}
			if err := json.Unmarshal(value, &v); err == nil {
				f.CustomFields[key] = v
			}
		}
	}

	return nil
}

// MarshalJSON custom marshaler to include custom fields
func (f IssueFields) MarshalJSON() ([]byte, error) {
	// Start with typed fields
	result := make(map[string]interface{})

	result["summary"] = f.Summary
	if f.Description != nil {
		result["description"] = f.Description
	}
	if f.Status != nil {
		result["status"] = f.Status
	}
	if f.IssueType != nil {
		result["issuetype"] = f.IssueType
	}
	if f.Priority != nil {
		result["priority"] = f.Priority
	}
	if f.Assignee != nil {
		result["assignee"] = f.Assignee
	}
	if f.Reporter != nil {
		result["reporter"] = f.Reporter
	}
	if f.Project != nil {
		result["project"] = f.Project
	}
	if f.Created != "" {
		result["created"] = f.Created
	}
	if f.Updated != "" {
		result["updated"] = f.Updated
	}
	if len(f.Labels) > 0 {
		result["labels"] = f.Labels
	}
	if len(f.Components) > 0 {
		result["components"] = f.Components
	}
	if f.Sprint != nil {
		result["sprint"] = f.Sprint
	}
	if f.Parent != nil {
		result["parent"] = f.Parent
	}

	// Add custom fields
	for key, value := range f.CustomFields {
		result[key] = value
	}

	return json.Marshal(result)
}

// Description can be either a string (Agile API) or ADF document (REST API v3)
type Description struct {
	Text string       // Plain text (from string or extracted from ADF)
	ADF  *ADFDocument // Original ADF document if available
}

// UnmarshalJSON handles both string and ADF document formats
func (d *Description) UnmarshalJSON(data []byte) error {
	// Try as string first
	var str string
	if err := json.Unmarshal(data, &str); err == nil {
		d.Text = str
		return nil
	}

	// Try as ADF document
	var adf ADFDocument
	if err := json.Unmarshal(data, &adf); err == nil {
		d.ADF = &adf
		d.Text = adf.ToPlainText()
		return nil
	}

	// If neither works, just ignore (null or empty)
	return nil
}

// MarshalJSON always outputs ADF format for API compatibility
func (d *Description) MarshalJSON() ([]byte, error) {
	if d.ADF != nil {
		return json.Marshal(d.ADF)
	}
	if d.Text != "" {
		return json.Marshal(NewADFDocument(d.Text))
	}
	return []byte("null"), nil
}

// ToPlainText returns the plain text content
func (d *Description) ToPlainText() string {
	if d == nil {
		return ""
	}
	return d.Text
}

// ADFDocument represents Atlassian Document Format content
type ADFDocument struct {
	Type    string    `json:"type"`
	Version int       `json:"version,omitempty"`
	Content []ADFNode `json:"content,omitempty"`
}

// ADFNode represents a node in an ADF document
type ADFNode struct {
	Type    string                 `json:"type"`
	Text    string                 `json:"text,omitempty"`
	Content []ADFNode              `json:"content,omitempty"`
	Attrs   map[string]interface{} `json:"attrs,omitempty"`
	Marks   []ADFMark              `json:"marks,omitempty"`
}

// ADFMark represents text formatting in ADF
type ADFMark struct {
	Type  string                 `json:"type"`
	Attrs map[string]interface{} `json:"attrs,omitempty"`
}

// NewADFDocument creates an ADF document from markdown text.
// Supports headings, bold, italic, code, code blocks, lists, links, and blockquotes.
func NewADFDocument(text string) *ADFDocument {
	return MarkdownToADF(text)
}

// ToPlainText extracts plain text from an ADF document
func (d *ADFDocument) ToPlainText() string {
	if d == nil {
		return ""
	}
	return extractText(d.Content)
}

func extractText(nodes []ADFNode) string {
	var result string
	for _, node := range nodes {
		if node.Text != "" {
			result += node.Text
		}
		if len(node.Content) > 0 {
			result += extractText(node.Content)
		}
		if node.Type == "paragraph" || node.Type == "hardBreak" {
			result += "\n"
		}
	}
	return result
}

// Status represents an issue status
type Status struct {
	ID             string         `json:"id"`
	Name           string         `json:"name"`
	Description    string         `json:"description,omitempty"`
	StatusCategory StatusCategory `json:"statusCategory,omitempty"`
}

// StatusCategory represents a status category
type StatusCategory struct {
	ID   int    `json:"id"`
	Key  string `json:"key"`
	Name string `json:"name"`
}

// IssueType represents an issue type
type IssueType struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Subtask     bool   `json:"subtask"`
}

// Priority represents an issue priority
type Priority struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// User represents a Jira user
type User struct {
	AccountID    string            `json:"accountId"`
	DisplayName  string            `json:"displayName"`
	EmailAddress string            `json:"emailAddress,omitempty"`
	Active       bool              `json:"active"`
	AvatarURLs   map[string]string `json:"avatarUrls,omitempty"`
}

// Project represents a Jira project
type Project struct {
	ID         string            `json:"id"`
	Key        string            `json:"key"`
	Name       string            `json:"name"`
	AvatarURLs map[string]string `json:"avatarUrls,omitempty"`
}

// Component represents a project component
type Component struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Sprint represents an agile sprint
type Sprint struct {
	ID            int        `json:"id"`
	Name          string     `json:"name"`
	State         string     `json:"state"`
	StartDate     *time.Time `json:"startDate,omitempty"`
	EndDate       *time.Time `json:"endDate,omitempty"`
	CompleteDate  *time.Time `json:"completeDate,omitempty"`
	OriginBoardID int        `json:"originBoardId,omitempty"`
	Goal          string     `json:"goal,omitempty"`
}

// Board represents an agile board
type Board struct {
	ID       int           `json:"id"`
	Name     string        `json:"name"`
	Type     string        `json:"type"`
	Location BoardLocation `json:"location,omitempty"`
}

// BoardLocation contains project info for a board
type BoardLocation struct {
	ProjectID   int    `json:"projectId"`
	ProjectKey  string `json:"projectKey"`
	ProjectName string `json:"projectName"`
}

// Transition represents a workflow transition
type Transition struct {
	ID     string                     `json:"id"`
	Name   string                     `json:"name"`
	To     Status                     `json:"to"`
	Fields map[string]TransitionField `json:"fields,omitempty"`
}

// TransitionField represents field metadata for a transition
type TransitionField struct {
	Required      bool          `json:"required"`
	Name          string        `json:"name"`
	Schema        FieldSchema   `json:"schema,omitempty"`
	AllowedValues []FieldOption `json:"allowedValues,omitempty"`
}

// FieldOption represents an allowed value for a field
type FieldOption struct {
	ID    string `json:"id,omitempty"`
	Name  string `json:"name,omitempty"`
	Value string `json:"value,omitempty"`
}

// Comment represents an issue comment
type Comment struct {
	ID      string       `json:"id"`
	Author  User         `json:"author"`
	Body    *ADFDocument `json:"body"`
	Created string       `json:"created"`
	Updated string       `json:"updated"`
}

// Field represents a Jira field definition
type Field struct {
	ID          string      `json:"id"`
	Key         string      `json:"key"`
	Name        string      `json:"name"`
	Custom      bool        `json:"custom"`
	Orderable   bool        `json:"orderable"`
	Navigable   bool        `json:"navigable"`
	Searchable  bool        `json:"searchable"`
	Schema      FieldSchema `json:"schema,omitempty"`
	ClauseNames []string    `json:"clauseNames,omitempty"`
}

// FieldSchema describes the data type of a field
type FieldSchema struct {
	Type     string `json:"type"`
	Items    string `json:"items,omitempty"`
	System   string `json:"system,omitempty"`
	Custom   string `json:"custom,omitempty"`
	CustomID int    `json:"customId,omitempty"`
}

// SearchResult represents search results from JQL
type SearchResult struct {
	StartAt    int     `json:"startAt"`
	MaxResults int     `json:"maxResults"`
	Total      int     `json:"total"`
	Issues     []Issue `json:"issues"`
}

// BoardsResponse represents the response from listing boards
type BoardsResponse struct {
	MaxResults int     `json:"maxResults"`
	StartAt    int     `json:"startAt"`
	Total      int     `json:"total"`
	IsLast     bool    `json:"isLast"`
	Values     []Board `json:"values"`
}

// SprintsResponse represents the response from listing sprints
type SprintsResponse struct {
	MaxResults int      `json:"maxResults"`
	StartAt    int      `json:"startAt"`
	IsLast     bool     `json:"isLast"`
	Values     []Sprint `json:"values"`
}

// TransitionsResponse represents available transitions
type TransitionsResponse struct {
	Transitions []Transition `json:"transitions"`
}

// CommentsResponse represents issue comments
type CommentsResponse struct {
	StartAt    int       `json:"startAt"`
	MaxResults int       `json:"maxResults"`
	Total      int       `json:"total"`
	Comments   []Comment `json:"comments"`
}

// CreateIssueRequest represents a request to create an issue
type CreateIssueRequest struct {
	Fields map[string]interface{} `json:"fields"`
}

// UpdateIssueRequest represents a request to update an issue
type UpdateIssueRequest struct {
	Fields map[string]interface{} `json:"fields,omitempty"`
	Update map[string]interface{} `json:"update,omitempty"`
}

// TransitionRequest represents a request to transition an issue
type TransitionRequest struct {
	Transition TransitionID           `json:"transition"`
	Fields     map[string]interface{} `json:"fields,omitempty"`
}

// TransitionID wraps a transition ID
type TransitionID struct {
	ID string `json:"id"`
}

// AddCommentRequest represents a request to add a comment
type AddCommentRequest struct {
	Body *ADFDocument `json:"body"`
}
