package artifact

import (
	"fmt"
	"strings"

	"github.com/open-cli-collective/atlassian-go/artifact"

	"github.com/open-cli-collective/jira-ticket-cli/api"
)

// AutomationRuleArtifact is the projected output for an automation rule.
type AutomationRuleArtifact struct {
	// Agent fields - essential for triage
	ID               string `json:"id"`
	Name             string `json:"name"`
	State            string `json:"state"`
	ComponentSummary string `json:"componentSummary"`

	// Full-only fields
	Description string   `json:"description,omitempty"`
	Labels      []string `json:"labels,omitempty"`
	Tags        []string `json:"tags,omitempty"`
}

// ProjectAutomationRule projects an api.AutomationRule to an AutomationRuleArtifact.
func ProjectAutomationRule(rule *api.AutomationRule, mode artifact.Type) *AutomationRuleArtifact {
	a := &AutomationRuleArtifact{
		ID:               rule.Identifier(),
		Name:             rule.Name,
		State:            rule.State,
		ComponentSummary: summarizeComponents(rule.Components),
	}

	if mode.IsFull() {
		a.Description = rule.Description
		if len(rule.Labels) > 0 {
			a.Labels = rule.Labels
		}
		if len(rule.Tags) > 0 {
			a.Tags = rule.Tags
		}
	}

	return a
}

// summarizeComponents creates a compact summary of rule components.
// Format: "3 total — 1 trigger(s), 2 action(s)"
func summarizeComponents(components []api.RuleComponent) string {
	if len(components) == 0 {
		return "none"
	}

	triggers, conditions, actions, other := 0, 0, 0, 0
	for _, c := range components {
		switch c.Component {
		case "TRIGGER":
			triggers++
		case "CONDITION":
			conditions++
		case "ACTION":
			actions++
		default:
			other++
		}
	}

	parts := make([]string, 0, 4)
	if triggers > 0 {
		parts = append(parts, fmt.Sprintf("%d trigger(s)", triggers))
	}
	if conditions > 0 {
		parts = append(parts, fmt.Sprintf("%d condition(s)", conditions))
	}
	if actions > 0 {
		parts = append(parts, fmt.Sprintf("%d action(s)", actions))
	}
	if other > 0 {
		parts = append(parts, fmt.Sprintf("%d other", other))
	}

	return fmt.Sprintf("%d total — %s", len(components), strings.Join(parts, ", "))
}
