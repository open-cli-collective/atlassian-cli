package api //nolint:revive // package name is intentional

import (
	"encoding/json"
	"fmt"
	"sync/atomic"
)

// RuleBuilder constructs automation rule JSON for use with CreateAutomationRule.
type RuleBuilder struct {
	name        string
	description string
	state       string
	trigger     *RuleComponent
	components  []RuleComponent
	projectARIs []string

	authorAccountID string
	actorAccountID  string // defaults to authorAccountID if empty
	writeAccessType string // defaults to "UNRESTRICTED"

	canOtherRuleTrigger bool
	notifyOnError       string // FIRSTERROR, ALWAYS, NEVER
}

// NewRuleBuilder creates a new automation rule builder with the given name.
// Rules are created in DISABLED state by default.
func NewRuleBuilder(name string) *RuleBuilder {
	return &RuleBuilder{
		name:          name,
		state:         "DISABLED",
		notifyOnError: "FIRSTERROR",
	}
}

// WithDescription sets the rule description.
func (b *RuleBuilder) WithDescription(desc string) *RuleBuilder {
	b.description = desc
	return b
}

// WithState sets the rule state (ENABLED or DISABLED).
func (b *RuleBuilder) WithState(state string) *RuleBuilder {
	b.state = state
	return b
}

// WithTrigger sets the rule's trigger component.
func (b *RuleBuilder) WithTrigger(t RuleComponent) *RuleBuilder {
	b.trigger = &t
	return b
}

// AddComponent appends a condition, action, or branch component.
func (b *RuleBuilder) AddComponent(c RuleComponent) *RuleBuilder {
	b.components = append(b.components, c)
	return b
}

// ForProjects scopes the rule to specific projects by ARI.
// Example ARI: "ari:cloud:jira:CLOUD_ID:project/PROJECT_ID"
func (b *RuleBuilder) ForProjects(aris ...string) *RuleBuilder {
	b.projectARIs = append(b.projectARIs, aris...)
	return b
}

// AllowOtherRuleTrigger sets whether other automation rules can trigger this rule.
func (b *RuleBuilder) AllowOtherRuleTrigger(allow bool) *RuleBuilder {
	b.canOtherRuleTrigger = allow
	return b
}

// NotifyOnError sets the error notification policy: FIRSTERROR, ALWAYS, or NEVER.
func (b *RuleBuilder) NotifyOnError(policy string) *RuleBuilder {
	b.notifyOnError = policy
	return b
}

// WithAuthor sets the rule author's Atlassian account ID.
// This is required — Build() returns an error if not set.
func (b *RuleBuilder) WithAuthor(accountID string) *RuleBuilder {
	b.authorAccountID = accountID
	return b
}

// WithActor sets the account ID that the rule runs as.
// If not set, defaults to the author's account ID.
// The actor is often the "Automation for Jira" service account.
func (b *RuleBuilder) WithActor(accountID string) *RuleBuilder {
	b.actorAccountID = accountID
	return b
}

// WithWriteAccessType sets the write access type for the rule.
// Defaults to "UNRESTRICTED" if not set.
func (b *RuleBuilder) WithWriteAccessType(accessType string) *RuleBuilder {
	b.writeAccessType = accessType
	return b
}

// rulePayload is the JSON shape accepted by the Automation REST API for create/update.
// The export endpoint returns {"rule": {...}, "connections": [...]}, and the create
// endpoint accepts the same shape.
type rulePayload struct {
	Rule        ruleBody          `json:"rule"`
	Connections []json.RawMessage `json:"connections"`
}

type ruleBody struct {
	Name                string          `json:"name"`
	State               string          `json:"state"`
	Description         string          `json:"description,omitempty"`
	AuthorAccountID     string          `json:"authorAccountId"`
	Actor               ruleActor       `json:"actor"`
	WriteAccessType     string          `json:"writeAccessType"`
	Trigger             *RuleComponent  `json:"trigger,omitempty"`
	Components          []RuleComponent `json:"components"`
	CanOtherRuleTrigger bool            `json:"canOtherRuleTrigger"`
	NotifyOnError       string          `json:"notifyOnError"`
	RuleScopeARIs       []string        `json:"ruleScopeARIs,omitempty"`
}

// ruleActor identifies who the automation rule runs as.
type ruleActor struct {
	Type  string `json:"type"`
	Actor string `json:"actor"`
}

// Build produces the JSON payload for CreateAutomationRule.
// Returns the envelope format: {"rule": {...}, "connections": []}.
func (b *RuleBuilder) Build() (json.RawMessage, error) {
	if b.name == "" {
		return nil, fmt.Errorf("rule name is required")
	}
	if b.authorAccountID == "" {
		return nil, fmt.Errorf("authorAccountId is required; call WithAuthor()")
	}
	if b.trigger != nil {
		hasAction := false
		for _, c := range b.components {
			if c.Component == "ACTION" {
				hasAction = true
				break
			}
			// If/else blocks contain actions in their branches — verify at least one branch has actions.
			if c.Type == "jira.condition.container.block" && ifElseBlockHasActions(c) {
				hasAction = true
				break
			}
		}
		if !hasAction {
			return nil, fmt.Errorf("rules with a trigger must have at least one action component")
		}
	}

	if err := validateIfElseBlocks(b.components); err != nil {
		return nil, err
	}

	actorID := b.actorAccountID
	if actorID == "" {
		actorID = b.authorAccountID
	}

	writeAccess := b.writeAccessType
	if writeAccess == "" {
		writeAccess = "UNRESTRICTED"
	}

	body := ruleBody{
		Name:                b.name,
		State:               b.state,
		Description:         b.description,
		AuthorAccountID:     b.authorAccountID,
		Actor:               ruleActor{Type: "ACCOUNT_ID", Actor: actorID},
		WriteAccessType:     writeAccess,
		Trigger:             b.trigger,
		Components:          b.components,
		CanOtherRuleTrigger: b.canOtherRuleTrigger,
		NotifyOnError:       b.notifyOnError,
	}

	if len(b.projectARIs) > 0 {
		body.RuleScopeARIs = b.projectARIs
	}

	if body.Components == nil {
		body.Components = []RuleComponent{}
	}

	payload := rulePayload{
		Rule:        body,
		Connections: []json.RawMessage{},
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshalling rule: %w", err)
	}

	return data, nil
}

// ---------------------------------------------------------------------------
// Trigger builders
// ---------------------------------------------------------------------------

// IssueCreatedTrigger returns a trigger that fires when an issue is created.
// Optional projectARIs filter which projects trigger the rule.
func IssueCreatedTrigger(projectARIs ...string) RuleComponent {
	value := map[string]any{
		"eventKey":   "jira:issue_created",
		"issueEvent": "issue_created",
	}
	if len(projectARIs) > 0 {
		value["eventFilters"] = projectARIs
	}

	return RuleComponent{
		Component:     "TRIGGER",
		Type:          "jira.issue.event.trigger:created",
		SchemaVersion: 1,
		Value:         mustMarshal(value),
		Children:      emptyArray(),
		Conditions:    emptyArray(),
	}
}

// IssueTransitionedTrigger returns a trigger that fires when an issue transitions
// to one of the given status names.
func IssueTransitionedTrigger(toStatuses ...string) RuleComponent {
	statuses := make([]map[string]string, len(toStatuses))
	for i, s := range toStatuses {
		statuses[i] = map[string]string{"type": "NAME", "value": s}
	}

	value := map[string]any{
		"eventKey":   "jira:issue_updated",
		"issueEvent": "issue_generic",
		"fromStatus": []any{},
		"toStatus":   statuses,
	}

	return RuleComponent{
		Component:     "TRIGGER",
		Type:          "jira.issue.event.trigger:transitioned",
		SchemaVersion: 1,
		Value:         mustMarshal(value),
		Children:      emptyArray(),
		Conditions:    emptyArray(),
	}
}

// ManualTrigger returns a trigger that adds a manual run button.
// Optional groups restrict who can trigger it.
func ManualTrigger(groups ...string) RuleComponent {
	value := map[string]any{}
	if len(groups) > 0 {
		value["groups"] = groups
	}

	return RuleComponent{
		Component:     "TRIGGER",
		Type:          "jira.manual.trigger.issue",
		SchemaVersion: 1,
		Value:         mustMarshal(value),
		Children:      emptyArray(),
		Conditions:    emptyArray(),
	}
}

// ScheduledTrigger returns a trigger that runs on a cron schedule against a JQL query.
func ScheduledTrigger(jql string, cronExpression string) RuleComponent {
	value := map[string]any{
		"jql":  jql,
		"cron": cronExpression,
	}

	return RuleComponent{
		Component:     "TRIGGER",
		Type:          "jira.jql.scheduled",
		SchemaVersion: 1,
		Value:         mustMarshal(value),
		Children:      emptyArray(),
		Conditions:    emptyArray(),
	}
}

// FieldChangedTrigger returns a trigger that fires when a specific field value changes.
func FieldChangedTrigger(fieldID string) RuleComponent {
	value := map[string]any{
		"changeType": "ANY_CHANGE",
		"fields": []map[string]string{
			{"value": fieldID},
		},
	}

	return RuleComponent{
		Component:     "TRIGGER",
		Type:          "jira.issue.field.changed",
		SchemaVersion: 1,
		Value:         mustMarshal(value),
		Children:      emptyArray(),
		Conditions:    emptyArray(),
	}
}

// ---------------------------------------------------------------------------
// Condition builders
// ---------------------------------------------------------------------------

// JQLCondition returns a condition that evaluates a JQL query.
// The issue must match the JQL for the rule to continue.
func JQLCondition(jql string) RuleComponent {
	return RuleComponent{
		Component:     "CONDITION",
		Type:          "jira.jql.condition",
		SchemaVersion: 1,
		Value:         mustMarshal(jql),
		Children:      emptyArray(),
		Conditions:    emptyArray(),
	}
}

// FieldRef identifies a field by ID or name.
type FieldRef struct {
	Type  string `json:"type"`  // "ID" or "NAME"
	Value string `json:"value"` // e.g., "customfield_10100" or "Banking Platform"
}

// ComparatorConditionValue is the value schema for jira.comparator.condition.
type ComparatorConditionValue struct {
	First    string `json:"first"`
	Second   string `json:"second"`
	Operator string `json:"operator"`
}

// ComparatorCondition returns a condition that compares two values.
// Typically used with smart values (e.g., "{{bankPlatform}}" EQUALS "Q2").
//
// Operators: EQUALS, NOT_EQUALS, GREATER_THAN, LESS_THAN, CONTAINS, NOT_CONTAINS
func ComparatorCondition(first, second, operator string) RuleComponent {
	val := ComparatorConditionValue{
		First:    first,
		Second:   second,
		Operator: operator,
	}

	return RuleComponent{
		Component:     "CONDITION",
		Type:          "jira.comparator.condition",
		SchemaVersion: 1,
		Value:         mustMarshal(val),
		Children:      emptyArray(),
		Conditions:    emptyArray(),
	}
}

// IfBlock represents one branch in an if/else block.
type IfBlock struct {
	// MatchType controls how conditions combine: "ALL" (AND) or "ANY" (OR).
	MatchType  string
	Conditions []RuleComponent
	Actions    []RuleComponent
}

// IfElseBlock returns a condition container with if/else-if/else branches.
// Each IfBlock has a match type, conditions, and actions.
// The last block with no conditions acts as the "else" branch.
func IfElseBlock(blocks ...IfBlock) RuleComponent {
	children := make([]json.RawMessage, len(blocks))
	for i, block := range blocks {
		child := map[string]any{
			"component":     "CONDITION_BLOCK",
			"type":          "jira.condition.if.block",
			"schemaVersion": 1,
			"value": map[string]string{
				"conditionMatchType": block.MatchType,
			},
			"conditions":   marshalComponents(block.Conditions),
			"children":     marshalComponents(block.Actions),
			"connectionId": nil,
		}
		children[i] = mustMarshal(child)
	}

	return RuleComponent{
		Component:     "CONDITION",
		Type:          "jira.condition.container.block",
		SchemaVersion: 1,
		Value:         mustMarshal(map[string]any{}),
		Children:      mustMarshal(children),
		Conditions:    emptyArray(),
	}
}

// ---------------------------------------------------------------------------
// Action builders
// ---------------------------------------------------------------------------

// CreateVariable returns an action that extracts a value into a named variable.
// Use smart values to reference issue fields: "{{triggerIssue.customField_10037}}"
func CreateVariable(name, smartValue string) RuleComponent {
	val := map[string]any{
		"id": nextSmartValueID(),
		"name": map[string]string{
			"type":  "FREE",
			"value": name,
		},
		"type": "SMART",
		"query": map[string]string{
			"type":  "SMART",
			"value": smartValue,
		},
		"lazy": false,
	}

	return RuleComponent{
		Component:     "ACTION",
		Type:          "jira.create.variable",
		SchemaVersion: 1,
		Value:         mustMarshal(val),
		Children:      emptyArray(),
		Conditions:    emptyArray(),
	}
}

// FieldOperation describes a single field edit within an EditIssueFields or CreateIssue action.
type FieldOperation struct {
	// FieldRef identifies the field by ID or NAME.
	Field FieldRef `json:"field"`
	// FieldType is the Jira field type plugin key.
	FieldType string `json:"fieldType"`
	// OperationType is SET, ADD, or REMOVE.
	OperationType string `json:"type"`
	// Value is the field value to set. The type depends on the field.
	Value any `json:"value"`
}

// SetField creates a FieldOperation that sets a field by ID.
func SetField(fieldID, fieldType string, value any) FieldOperation {
	return FieldOperation{
		Field:         FieldRef{Type: "ID", Value: fieldID},
		FieldType:     fieldType,
		OperationType: "SET",
		Value:         value,
	}
}

// SetFieldByName creates a FieldOperation that sets a field by name.
func SetFieldByName(fieldName, fieldType string, value any) FieldOperation {
	return FieldOperation{
		Field:         FieldRef{Type: "NAME", Value: fieldName},
		FieldType:     fieldType,
		OperationType: "SET",
		Value:         value,
	}
}

// EditIssueFields returns an action that edits fields on the current issue.
func EditIssueFields(sendNotifications bool, operations ...FieldOperation) RuleComponent {
	val := map[string]any{
		"operations":        operations,
		"advancedFields":    nil,
		"sendNotifications": sendNotifications,
	}

	return RuleComponent{
		Component:     "ACTION",
		Type:          "jira.issue.edit",
		SchemaVersion: 10,
		Value:         mustMarshal(val),
		Children:      emptyArray(),
		Conditions:    emptyArray(),
	}
}

// TransitionIssue returns an action that transitions the issue to a named status.
func TransitionIssue(statusName string) RuleComponent {
	val := map[string]any{
		"destinationStatus": map[string]string{
			"type":  "NAME",
			"value": statusName,
		},
	}

	return RuleComponent{
		Component:     "ACTION",
		Type:          "jira.issue.transition",
		SchemaVersion: 1,
		Value:         mustMarshal(val),
		Children:      emptyArray(),
		Conditions:    emptyArray(),
	}
}

// CommentOnIssue returns an action that adds a comment to the current issue.
func CommentOnIssue(body string) RuleComponent {
	val := map[string]any{
		"comment": body,
	}

	return RuleComponent{
		Component:     "ACTION",
		Type:          "jira.issue.comment",
		SchemaVersion: 1,
		Value:         mustMarshal(val),
		Children:      emptyArray(),
		Conditions:    emptyArray(),
	}
}

// AssignIssue returns an action that assigns the issue.
// Use smart values for dynamic assignment: "{{reporter.accountId}}"
func AssignIssue(accountID string) RuleComponent {
	val := map[string]any{
		"assignee": map[string]string{
			"type":  "SMART",
			"value": accountID,
		},
	}

	return RuleComponent{
		Component:     "ACTION",
		Type:          "jira.issue.assign",
		SchemaVersion: 1,
		Value:         mustMarshal(val),
		Children:      emptyArray(),
		Conditions:    emptyArray(),
	}
}

// LookupIssues returns an action that runs a JQL query and stores results in a named variable.
func LookupIssues(name, jql string) RuleComponent {
	val := map[string]any{
		"id": nextSmartValueID(),
		"name": map[string]string{
			"type":  "FREE",
			"value": name,
		},
		"type": "JQL",
		"query": map[string]string{
			"type":  "SMART",
			"value": jql,
		},
		"lazy": false,
	}

	return RuleComponent{
		Component:     "ACTION",
		Type:          "jira.lookup.issues",
		SchemaVersion: 1,
		Value:         mustMarshal(val),
		Children:      emptyArray(),
		Conditions:    emptyArray(),
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// smartValueIDCounter generates unique IDs for smart value variables.
var smartValueIDCounter atomic.Int64

func nextSmartValueID() string {
	n := smartValueIDCounter.Add(1)
	return fmt.Sprintf("_customsmartvalue_id_%d", n)
}

// ifElseBlockHasActions checks whether an IfElseBlock component contains at least
// one branch with actions. The Children field holds serialized CONDITION_BLOCK entries
// whose "children" arrays contain the branch's actions.
func ifElseBlockHasActions(c RuleComponent) bool {
	var blocks []struct {
		Children []json.RawMessage `json:"children"`
	}
	if err := json.Unmarshal(c.Children, &blocks); err != nil {
		return false
	}
	for _, block := range blocks {
		if len(block.Children) > 0 {
			return true
		}
	}
	return false
}

// validateIfElseBlocks checks that all IfElseBlock components have valid MatchType values.
func validateIfElseBlocks(components []RuleComponent) error {
	for _, c := range components {
		if c.Type != "jira.condition.container.block" {
			continue
		}
		var blocks []struct {
			Value struct {
				MatchType string `json:"conditionMatchType"`
			} `json:"value"`
		}
		if err := json.Unmarshal(c.Children, &blocks); err != nil {
			continue
		}
		for _, block := range blocks {
			if block.Value.MatchType != "ALL" && block.Value.MatchType != "ANY" {
				return fmt.Errorf("IfBlock.MatchType must be \"ALL\" or \"ANY\", got %q", block.Value.MatchType)
			}
		}
	}
	return nil
}

func mustMarshal(v any) json.RawMessage {
	data, err := json.Marshal(v)
	if err != nil {
		panic(fmt.Sprintf("automation builder: failed to marshal %T: %v", v, err))
	}
	return data
}

func emptyArray() json.RawMessage {
	return json.RawMessage("[]")
}

func marshalComponents(components []RuleComponent) []json.RawMessage {
	result := make([]json.RawMessage, len(components))
	for i, c := range components {
		result[i] = mustMarshal(c)
	}
	return result
}
