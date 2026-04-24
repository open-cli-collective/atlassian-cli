// Package present provides presenters that map domain types to presentation models.
package present

import (
	"fmt"
	"strings"

	"github.com/open-cli-collective/atlassian-go/present"

	"github.com/open-cli-collective/jira-ticket-cli/api"
)

// AutomationPresenter creates presentation models for automation rules.
type AutomationPresenter struct{}

// PresentDetail creates a detailed view of a single automation rule.
// When showComponents is true, a table of component details is appended.
func (AutomationPresenter) PresentDetail(rule *api.AutomationRule, showComponents bool) *present.OutputModel {
	fields := []present.Field{
		{Label: "Name", Value: rule.Name},
		{Label: "UUID", Value: rule.Identifier()},
		{Label: "State", Value: rule.State},
	}

	if rule.Description != "" {
		fields = append(fields, present.Field{Label: "Description", Value: rule.Description})
	}

	if len(rule.Labels) > 0 {
		fields = append(fields, present.Field{Label: "Labels", Value: strings.Join(rule.Labels, ", ")})
	}

	if len(rule.Tags) > 0 {
		fields = append(fields, present.Field{Label: "Tags", Value: strings.Join(rule.Tags, ", ")})
	}

	if len(rule.Projects) > 0 {
		projects := make([]string, 0, len(rule.Projects))
		for _, p := range rule.Projects {
			if p.ProjectKey != "" {
				projects = append(projects, p.ProjectKey)
			} else if p.ProjectName != "" {
				projects = append(projects, p.ProjectName)
			}
		}
		if len(projects) > 0 {
			fields = append(fields, present.Field{Label: "Projects", Value: strings.Join(projects, ", ")})
		}
	}

	fields = append(fields, present.Field{Label: "Components", Value: SummarizeComponents(rule.Components)})

	sections := []present.Section{&present.DetailSection{Fields: fields}}

	// Append component details table when requested
	if showComponents && len(rule.Components) > 0 {
		rows := make([]present.Row, len(rule.Components))
		for i, c := range rule.Components {
			rows[i] = present.Row{
				Cells: []string{fmt.Sprintf("%d", i+1), c.Component, c.Type},
			}
		}
		sections = append(sections, &present.TableSection{
			Headers: []string{"#", "COMPONENT", "TYPE"},
			Rows:    rows,
		})
	}

	return &present.OutputModel{Sections: sections}
}

// PresentStateChanged creates a success message for a state transition.
func (AutomationPresenter) PresentStateChanged(name, fromState, toState string) *present.OutputModel {
	return &present.OutputModel{
		Sections: []present.Section{
			&present.MessageSection{
				Kind:    present.MessageSuccess,
				Message: fmt.Sprintf("Rule %q: %s → %s", name, fromState, toState),
				Stream:  present.StreamStdout,
			},
		},
	}
}

// PresentNoChange creates an advisory message when rule is already in desired state.
// Routes to stderr because no mutation occurred.
func (AutomationPresenter) PresentNoChange(name, state string) *present.OutputModel {
	return &present.OutputModel{
		Sections: []present.Section{
			&present.MessageSection{
				Kind:    present.MessageInfo,
				Message: fmt.Sprintf("Rule %q is already %s", name, state),
				Stream:  present.StreamStderr,
			},
		},
	}
}

// PresentCreated creates a success message for rule creation with name.
func (AutomationPresenter) PresentCreated(name, uuid string) *present.OutputModel {
	return &present.OutputModel{
		Sections: []present.Section{
			&present.MessageSection{
				Kind:    present.MessageSuccess,
				Message: fmt.Sprintf("Created automation rule: %s (UUID: %s)", name, uuid),
				Stream:  present.StreamStdout,
			},
		},
	}
}

// PresentCreatedMinimal creates a success message for rule creation without name.
func (AutomationPresenter) PresentCreatedMinimal(uuid string) *present.OutputModel {
	return &present.OutputModel{
		Sections: []present.Section{
			&present.MessageSection{
				Kind:    present.MessageSuccess,
				Message: fmt.Sprintf("Created automation rule (UUID: %s)", uuid),
				Stream:  present.StreamStdout,
			},
		},
	}
}

// PresentCreatedUnparsed creates a success message when response couldn't be parsed.
func (AutomationPresenter) PresentCreatedUnparsed() *present.OutputModel {
	return &present.OutputModel{
		Sections: []present.Section{
			&present.MessageSection{
				Kind:    present.MessageSuccess,
				Message: "Created automation rule (could not parse response for details)",
				Stream:  present.StreamStdout,
			},
		},
	}
}

// PresentUpdated creates a success message for rule update.
func (AutomationPresenter) PresentUpdated(ruleID string) *present.OutputModel {
	return &present.OutputModel{
		Sections: []present.Section{
			&present.MessageSection{
				Kind:    present.MessageSuccess,
				Message: fmt.Sprintf("Updated automation rule %s", ruleID),
				Stream:  present.StreamStdout,
			},
		},
	}
}

// PresentUpdateProgress creates an advisory message showing update progress.
func (AutomationPresenter) PresentUpdateProgress(name, uuid, state string) *present.OutputModel {
	return &present.OutputModel{
		Sections: []present.Section{
			&present.MessageSection{
				Kind:    present.MessageInfo,
				Message: fmt.Sprintf("Updating rule: %s (UUID: %s, State: %s)", name, uuid, state),
				Stream:  present.StreamStderr,
			},
		},
	}
}

// PresentUpdateComplete creates progress + success as single output.
func (AutomationPresenter) PresentUpdateComplete(name, uuid, state, ruleID string) *present.OutputModel {
	return &present.OutputModel{
		Sections: []present.Section{
			&present.MessageSection{
				Kind:    present.MessageInfo,
				Message: fmt.Sprintf("Updating rule: %s (UUID: %s, State: %s)", name, uuid, state),
				Stream:  present.StreamStderr,
			},
			&present.MessageSection{
				Kind:    present.MessageSuccess,
				Message: fmt.Sprintf("Updated automation rule %s", ruleID),
				Stream:  present.StreamStdout,
			},
		},
	}
}

// PresentDeleted creates a success message for rule deletion.
func (AutomationPresenter) PresentDeleted(ruleID string) *present.OutputModel {
	return &present.OutputModel{
		Sections: []present.Section{
			&present.MessageSection{
				Kind:    present.MessageSuccess,
				Message: fmt.Sprintf("Deleted automation %s", ruleID),
				Stream:  present.StreamStdout,
			},
		},
	}
}

// PresentDeleteCancelled creates an info message for cancelled deletion.
func (AutomationPresenter) PresentDeleteCancelled() *present.OutputModel {
	return &present.OutputModel{
		Sections: []present.Section{
			&present.MessageSection{
				Kind:    present.MessageInfo,
				Message: "Deletion cancelled.",
				Stream:  present.StreamStdout,
			},
		},
	}
}

// PresentEmpty creates an info message for empty automation list.
func (AutomationPresenter) PresentEmpty() *present.OutputModel {
	return &present.OutputModel{
		Sections: []present.Section{
			&present.MessageSection{
				Kind:    present.MessageInfo,
				Message: "No automation rules found",
				Stream:  present.StreamStdout,
			},
		},
	}
}

// PresentList creates a table view of automation rules.
func (AutomationPresenter) PresentList(rules []api.AutomationRuleSummary) *present.OutputModel {
	rows := make([]present.Row, len(rules))
	for i, r := range rules {
		rows[i] = present.Row{
			Cells: []string{r.Identifier(), r.Name, r.State},
		}
	}

	return &present.OutputModel{
		Sections: []present.Section{
			&present.TableSection{
				Headers: []string{"UUID", "NAME", "STATE"},
				Rows:    rows,
			},
		},
	}
}

// SummarizeComponents creates a compact summary of rule components.
// Exported for testing.
func SummarizeComponents(components []api.RuleComponent) string {
	if len(components) == 0 {
		return "none"
	}

	triggers, conditions, actions := 0, 0, 0
	for _, c := range components {
		switch c.Component {
		case "TRIGGER":
			triggers++
		case "CONDITION":
			conditions++
		case "ACTION":
			actions++
		}
	}

	parts := make([]string, 0, 3)
	if triggers > 0 {
		parts = append(parts, fmt.Sprintf("%d trigger(s)", triggers))
	}
	if conditions > 0 {
		parts = append(parts, fmt.Sprintf("%d condition(s)", conditions))
	}
	if actions > 0 {
		parts = append(parts, fmt.Sprintf("%d action(s)", actions))
	}

	return fmt.Sprintf("%d total — %s", len(components), strings.Join(parts, ", "))
}
