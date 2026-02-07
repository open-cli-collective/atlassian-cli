package api

import "encoding/json"

// AutomationRule represents a full automation rule.
//
// Note: Jira's Automation API shapes vary between endpoints and have changed
// over time. We keep fields that cover both the documented/legacy responses
// and the actual responses observed in the wild.
type AutomationRule struct {
	// Legacy numeric ID (may be absent).
	ID json.Number `json:"id,omitempty"`

	// Preferred identifier in newer Cloud responses.
	UUID string `json:"uuid,omitempty"`

	// Legacy UUID field name used in some older responses.
	RuleKey string `json:"ruleKey,omitempty"`

	Name            string          `json:"name"`
	State           string          `json:"state"`
	Description     string          `json:"description,omitempty"`
	AuthorAccountID string          `json:"authorAccountId,omitempty"`
	ActorAccountID  string          `json:"actorAccountId,omitempty"`
	Labels          []string        `json:"labels,omitempty"`
	Tags            []string        `json:"tags,omitempty"`
	Projects        []RuleProject   `json:"projects,omitempty"`
	Trigger         *RuleComponent  `json:"trigger,omitempty"`
	Components      []RuleComponent `json:"components,omitempty"`

	// Newer Cloud responses represent scope as ARIs rather than projects.
	RuleScopeARIs []string `json:"ruleScopeARIs,omitempty"`

	// Preserve unknown fields for round-trip fidelity.
	Extra map[string]json.RawMessage `json:"-"`
}

func (r AutomationRule) Identifier() string {
	if r.UUID != "" {
		return r.UUID
	}
	if r.RuleKey != "" {
		return r.RuleKey
	}
	if r.ID.String() != "" {
		return r.ID.String()
	}
	return ""
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
	// Legacy numeric ID (may be absent).
	ID json.Number `json:"id,omitempty"`

	// Preferred identifier in newer Cloud responses.
	UUID string `json:"uuid,omitempty"`

	// Legacy UUID field name used in some older responses.
	RuleKey string `json:"ruleKey,omitempty"`

	Name            string        `json:"name"`
	State           string        `json:"state"`
	Description     string        `json:"description,omitempty"`
	AuthorAccountID string        `json:"authorAccountId,omitempty"`
	ActorAccountID  string        `json:"actorAccountId,omitempty"`
	Labels          []string      `json:"labels,omitempty"`
	Tags            []string      `json:"tags,omitempty"`
	Projects        []RuleProject `json:"projects,omitempty"`
	RuleScopeARIs   []string      `json:"ruleScopeARIs,omitempty"`
}

func (s AutomationRuleSummary) Identifier() string {
	if s.UUID != "" {
		return s.UUID
	}
	if s.RuleKey != "" {
		return s.RuleKey
	}
	if s.ID.String() != "" {
		return s.ID.String()
	}
	return ""
}

type automationLinks struct {
	Self *string `json:"self"`
	Next *string `json:"next"`
	Prev *string `json:"prev"`
}

// AutomationRuleSummaryResponse is the paginated list response.
//
// Observed Cloud shape:
//
//	{"links": {"self": null, "next": null, "prev": null}, "data": [...]}
//
// Legacy/documented shape (kept for compatibility):
//
//	{"total": 2, "values": [...], "next": "..."}
type AutomationRuleSummaryResponse struct {
	// Newer Cloud shape.
	Links automationLinks         `json:"links"`
	Data  []AutomationRuleSummary `json:"data"`

	// Legacy shape.
	Total  int                     `json:"total"`
	Values []AutomationRuleSummary `json:"values"`
	Next   string                  `json:"next,omitempty"`
}

func (r AutomationRuleSummaryResponse) Items() []AutomationRuleSummary {
	if len(r.Data) > 0 {
		return r.Data
	}
	return r.Values
}

func (r AutomationRuleSummaryResponse) NextURL() string {
	if r.Links.Next != nil {
		return *r.Links.Next
	}
	return r.Next
}

// AutomationStateUpdate represents a request to enable or disable a rule.
type AutomationStateUpdate struct {
	RuleState string `json:"ruleState"`
}
