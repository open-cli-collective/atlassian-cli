package api

import "encoding/json"

// AutomationRule represents a full automation rule with all components.
type AutomationRule struct {
	ID              json.Number     `json:"id"`
	UUID            string          `json:"ruleKey,omitempty"`
	Name            string          `json:"name"`
	State           string          `json:"state"`
	Description     string          `json:"description,omitempty"`
	AuthorAccountID string          `json:"authorAccountId,omitempty"`
	Labels          []string        `json:"labels,omitempty"`
	Tags            []string        `json:"tags,omitempty"`
	Projects        []RuleProject   `json:"projects,omitempty"`
	Components      []RuleComponent `json:"components,omitempty"`
	RuleScope       json.RawMessage `json:"ruleScope,omitempty"`

	// Preserve unknown fields for round-trip fidelity
	Extra map[string]json.RawMessage `json:"-"`
}

// RuleProject identifies a project associated with a rule.
type RuleProject struct {
	ProjectID   string `json:"projectId,omitempty"`
	ProjectKey  string `json:"projectKey,omitempty"`
	ProjectName string `json:"projectName,omitempty"`
}

// RuleComponent represents a trigger, condition, or action in an automation rule.
// The Value field is kept as raw JSON because component schemas are undocumented.
type RuleComponent struct {
	ID            string          `json:"id,omitempty"`
	Component     string          `json:"component"`
	Type          string          `json:"type"`
	Value         json.RawMessage `json:"value,omitempty"`
	SchemaVersion int             `json:"schemaVersion,omitempty"`
	ParentID      string          `json:"parentId,omitempty"`
	Children      json.RawMessage `json:"children,omitempty"`
	Conditions    json.RawMessage `json:"conditions,omitempty"`
	ConnectionID  string          `json:"connectionId,omitempty"`
}

// AutomationRuleSummary is the lighter representation returned by the list/summary endpoint.
type AutomationRuleSummary struct {
	ID              json.Number   `json:"id"`
	UUID            string        `json:"ruleKey,omitempty"`
	Name            string        `json:"name"`
	State           string        `json:"state"`
	Description     string        `json:"description,omitempty"`
	AuthorAccountID string        `json:"authorAccountId,omitempty"`
	Labels          []string      `json:"labels,omitempty"`
	Tags            []string      `json:"tags,omitempty"`
	Projects        []RuleProject `json:"projects,omitempty"`
}

// AutomationRuleSummaryResponse is the paginated list response.
type AutomationRuleSummaryResponse struct {
	Total  int                     `json:"total"`
	Values []AutomationRuleSummary `json:"values"`
	Next   string                  `json:"next,omitempty"`
}

// AutomationStateUpdate represents a request to enable or disable a rule.
type AutomationStateUpdate struct {
	RuleState string `json:"ruleState"`
}
