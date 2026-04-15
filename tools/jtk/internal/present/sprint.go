package present

import (
	"fmt"

	"github.com/open-cli-collective/atlassian-go/present"

	"github.com/open-cli-collective/jira-ticket-cli/api"
)

// SprintPresenter creates presentation models for sprint data.
type SprintPresenter struct{}

// PresentDetail creates a detail view for a single sprint.
func (SprintPresenter) PresentDetail(sprint *api.Sprint) *present.OutputModel {
	fields := []present.Field{
		{Label: "ID", Value: FormatInt(sprint.ID)},
		{Label: "Name", Value: sprint.Name},
		{Label: "State", Value: sprint.State},
	}

	if sprint.Goal != "" {
		fields = append(fields, present.Field{Label: "Goal", Value: sprint.Goal})
	}
	if sprint.StartDate != nil {
		fields = append(fields, present.Field{Label: "Start Date", Value: FormatDate(sprint.StartDate)})
	}
	if sprint.EndDate != nil {
		fields = append(fields, present.Field{Label: "End Date", Value: FormatDate(sprint.EndDate)})
	}
	if sprint.CompleteDate != nil {
		fields = append(fields, present.Field{Label: "Complete Date", Value: FormatDate(sprint.CompleteDate)})
	}

	return &present.OutputModel{
		Sections: []present.Section{&present.DetailSection{Fields: fields}},
	}
}

// PresentList creates a table view for a list of sprints.
func (SprintPresenter) PresentList(sprints []api.Sprint) *present.OutputModel {
	rows := make([]present.Row, len(sprints))
	for i, sprint := range sprints {
		rows[i] = present.Row{
			Cells: []string{
				FormatInt(sprint.ID),
				sprint.Name,
				sprint.State,
				FormatDate(sprint.StartDate),
				FormatDate(sprint.EndDate),
			},
		}
	}

	return &present.OutputModel{
		Sections: []present.Section{
			&present.TableSection{
				Headers: []string{"ID", "NAME", "STATE", "START", "END"},
				Rows:    rows,
			},
		},
	}
}

// PresentMoved creates a success message for moving issues to a sprint.
func (SprintPresenter) PresentMoved(issueKeys []string, sprintID int) *present.OutputModel {
	var msg string
	if len(issueKeys) == 1 {
		msg = fmt.Sprintf("Moved %s to sprint %d", issueKeys[0], sprintID)
	} else {
		msg = fmt.Sprintf("Moved %d issues to sprint %d", len(issueKeys), sprintID)
	}
	return &present.OutputModel{
		Sections: []present.Section{
			&present.MessageSection{
				Kind:    present.MessageSuccess,
				Message: msg,
				Stream:  present.StreamStdout,
			},
		},
	}
}

// PresentEmpty creates an info message when no sprints are found.
func (SprintPresenter) PresentEmpty() *present.OutputModel {
	return &present.OutputModel{
		Sections: []present.Section{
			&present.MessageSection{
				Kind:    present.MessageInfo,
				Message: "No sprints found",
				Stream:  present.StreamStdout,
			},
		},
	}
}

// PresentNoIssues creates an info message when no issues are in a sprint.
func (SprintPresenter) PresentNoIssues() *present.OutputModel {
	return &present.OutputModel{
		Sections: []present.Section{
			&present.MessageSection{
				Kind:    present.MessageInfo,
				Message: "No issues in sprint",
				Stream:  present.StreamStdout,
			},
		},
	}
}
