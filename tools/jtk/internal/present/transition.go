package present

import (
	"fmt"
	"sort"
	"strings"

	"github.com/open-cli-collective/atlassian-go/present"

	"github.com/open-cli-collective/jira-ticket-cli/api"
	"github.com/open-cli-collective/jira-ticket-cli/internal/present/projection"
)

// TransitionPresenter creates presentation models for transition data.
type TransitionPresenter struct{}

// TransitionListSpec declares the columns emitted by PresentList. Default
// order per #230 is ID|NAME|TO_STATUS; extended adds STATUS_CATEGORY,
// HAS_SCREEN, CONDITIONAL, and REQUIRED_FIELDS.
var TransitionListSpec = projection.Registry{
	{Header: "ID", Identity: true},
	{Header: "NAME"},
	{Header: "TO_STATUS"},
	{Header: "STATUS_CATEGORY", Extended: true},
	{Header: "HAS_SCREEN", Extended: true},
	{Header: "CONDITIONAL", Extended: true},
	{Header: "REQUIRED_FIELDS", Extended: true},
}

// PresentList creates a table view for a list of transitions. Default
// order is ID|NAME|TO_STATUS; --extended adds STATUS_CATEGORY, HAS_SCREEN,
// CONDITIONAL, and REQUIRED_FIELDS.
func (TransitionPresenter) PresentList(transitions []api.Transition, extended bool) *present.OutputModel {
	var headers []string
	if extended {
		headers = []string{"ID", "NAME", "TO_STATUS", "STATUS_CATEGORY", "HAS_SCREEN", "CONDITIONAL", "REQUIRED_FIELDS"}
	} else {
		headers = []string{"ID", "NAME", "TO_STATUS"}
	}

	rows := make([]present.Row, len(transitions))
	for i, t := range transitions {
		toStatus := OrDash(t.To.Name)
		if extended {
			rows[i] = present.Row{
				Cells: []string{
					t.ID,
					t.Name,
					toStatus,
					OrDash(t.To.StatusCategory.Name),
					BoolString(t.HasScreen),
					BoolString(t.IsConditional),
					GetRequiredFieldsForTransition(t),
				},
			}
		} else {
			rows[i] = present.Row{
				Cells: []string{t.ID, t.Name, toStatus},
			}
		}
	}
	return &present.OutputModel{
		Sections: []present.Section{
			&present.TableSection{Headers: headers, Rows: rows},
		},
	}
}

// GetRequiredFieldsForTransition returns a comma-separated list of required field names.
func GetRequiredFieldsForTransition(t api.Transition) string {
	var required []string
	for _, field := range t.Fields {
		if field.Required {
			required = append(required, field.Name)
		}
	}
	if len(required) == 0 {
		return "-"
	}
	sort.Strings(required)
	return strings.Join(required, ", ")
}

// PresentTransitioned creates a success message for a completed transition.
func (TransitionPresenter) PresentTransitioned(issueKey string) *present.OutputModel {
	return &present.OutputModel{
		Sections: []present.Section{
			&present.MessageSection{
				Kind:    present.MessageSuccess,
				Message: fmt.Sprintf("Transitioned %s", issueKey),
				Stream:  present.StreamStdout,
			},
		},
	}
}

// PresentEmpty creates an info message when no transitions are available.
func (TransitionPresenter) PresentEmpty(issueKey string) *present.OutputModel {
	return &present.OutputModel{
		Sections: []present.Section{
			&present.MessageSection{
				Kind:    present.MessageInfo,
				Message: fmt.Sprintf("No transitions available for %s", issueKey),
				Stream:  present.StreamStdout,
			},
		},
	}
}

// PresentStatusNotFound creates an error explaining that no available
// transition lands on the requested target status. It lists the available
// target statuses to help the user pick a valid one.
func (TransitionPresenter) PresentStatusNotFound(requested string, available []api.Transition) *present.OutputModel {
	sections := []present.Section{
		&present.MessageSection{
			Kind:    present.MessageError,
			Message: fmt.Sprintf("no transition to status '%s' is currently available on this issue", requested),
			Stream:  present.StreamStderr,
		},
	}

	seen := make(map[string]bool, len(available))
	var statusLines []present.Section
	for _, t := range available {
		if t.To.Name == "" || seen[strings.ToLower(t.To.Name)] {
			continue
		}
		seen[strings.ToLower(t.To.Name)] = true
		statusLines = append(statusLines, &present.MessageSection{
			Kind:    present.MessageInfo,
			Message: fmt.Sprintf("  %s", t.To.Name),
			Stream:  present.StreamStderr,
		})
	}

	if len(statusLines) > 0 {
		sections = append(sections, &present.MessageSection{
			Kind:    present.MessageInfo,
			Message: "Available target statuses:",
			Stream:  present.StreamStderr,
		})
		sections = append(sections, statusLines...)
	} else {
		sections = append(sections, &present.MessageSection{
			Kind:    present.MessageInfo,
			Message: "No transitions are available on this issue (terminal state).",
			Stream:  present.StreamStderr,
		})
	}

	return &present.OutputModel{Sections: sections}
}

// PresentStatusAmbiguous creates an error when multiple available transitions
// land on the same target status. It recommends `jtk transitions do <key> <id>`
// to pick the intended transition unambiguously.
func (TransitionPresenter) PresentStatusAmbiguous(issueKey, requested string, candidates []api.Transition) *present.OutputModel {
	sections := []present.Section{
		&present.MessageSection{
			Kind:    present.MessageError,
			Message: fmt.Sprintf("multiple transitions land on status '%s'", requested),
			Stream:  present.StreamStderr,
		},
		&present.MessageSection{
			Kind:    present.MessageInfo,
			Message: "Candidates:",
			Stream:  present.StreamStderr,
		},
	}

	for _, t := range candidates {
		sections = append(sections, &present.MessageSection{
			Kind:    present.MessageInfo,
			Message: fmt.Sprintf("  %s: %s -> %s", t.ID, t.Name, t.To.Name),
			Stream:  present.StreamStderr,
		})
	}

	sections = append(sections, &present.MessageSection{
		Kind:    present.MessageInfo,
		Message: fmt.Sprintf("Pick one with: jtk transitions do %s <id>", issueKey),
		Stream:  present.StreamStderr,
	})

	return &present.OutputModel{Sections: sections}
}

// PresentNotFound creates an error with available transitions as context.
func (TransitionPresenter) PresentNotFound(name string, available []api.Transition) *present.OutputModel {
	sections := []present.Section{
		&present.MessageSection{
			Kind:    present.MessageError,
			Message: fmt.Sprintf("Transition '%s' not found", name),
			Stream:  present.StreamStderr,
		},
		&present.MessageSection{
			Kind:    present.MessageInfo,
			Message: "Available transitions:",
			Stream:  present.StreamStderr,
		},
	}

	for _, t := range available {
		sections = append(sections, &present.MessageSection{
			Kind:    present.MessageInfo,
			Message: fmt.Sprintf("  %s: %s -> %s", t.ID, t.Name, t.To.Name),
			Stream:  present.StreamStderr,
		})
	}

	return &present.OutputModel{Sections: sections}
}
