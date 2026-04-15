// Package present provides presenters that map domain types to presentation models.
package present

import (
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
