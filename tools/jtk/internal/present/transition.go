// Package present provides presenters that map domain types to presentation models.
package present

import (
	"fmt"
	"strings"

	"github.com/open-cli-collective/atlassian-go/present"

	"github.com/open-cli-collective/jira-ticket-cli/api"
)

// TransitionPresenter creates presentation models for transition data.
type TransitionPresenter struct{}

// PresentList creates a table view for a list of transitions.
func (TransitionPresenter) PresentList(transitions []api.Transition) *present.OutputModel {
	rows := make([]present.Row, len(transitions))
	for i, t := range transitions {
		toStatus := ""
		if t.To.Name != "" {
			toStatus = t.To.Name
		}
		rows[i] = present.Row{
			Cells: []string{t.ID, t.Name, toStatus},
		}
	}
	return &present.OutputModel{
		Sections: []present.Section{
			&present.TableSection{
				Headers: []string{"ID", "NAME", "TO STATUS"},
				Rows:    rows,
			},
		},
	}
}

// PresentListWithFields creates a table view for transitions with required fields.
func (TransitionPresenter) PresentListWithFields(transitions []api.Transition) *present.OutputModel {
	rows := make([]present.Row, len(transitions))
	for i, t := range transitions {
		toStatus := ""
		if t.To.Name != "" {
			toStatus = t.To.Name
		}
		required := getRequiredFields(t)
		rows[i] = present.Row{
			Cells: []string{t.ID, t.Name, toStatus, required},
		}
	}
	return &present.OutputModel{
		Sections: []present.Section{
			&present.TableSection{
				Headers: []string{"ID", "NAME", "TO STATUS", "REQUIRED FIELDS"},
				Rows:    rows,
			},
		},
	}
}

// GetRequiredFieldsForTransition returns a comma-separated list of required field names
// This is exported for use in transitions command tests
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
	return strings.Join(required, ", ")
}

// getRequiredFields returns a comma-separated list of required field names (internal use)
func getRequiredFields(t api.Transition) string {
	return GetRequiredFieldsForTransition(t)
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

// PresentNotFound creates an error with available transitions as context.
// Both the error and the available options route to stderr.
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
