package artifact

import (
	"time"

	"github.com/open-cli-collective/atlassian-go/artifact"

	"github.com/open-cli-collective/jira-ticket-cli/api"
)

// SprintArtifact is the projected output for a sprint.
type SprintArtifact struct {
	// Agent fields - essential for sprint selection
	ID    int    `json:"id"`
	Name  string `json:"name"`
	State string `json:"state"`

	// Full-only fields
	StartDate    string `json:"startDate,omitempty"`
	EndDate      string `json:"endDate,omitempty"`
	CompleteDate string `json:"completeDate,omitempty"`
	Goal         string `json:"goal,omitempty"`
	BoardID      *int   `json:"boardId,omitempty"` // Pointer so 0 is explicit in full mode
}

// ProjectSprint projects an api.Sprint to a SprintArtifact.
func ProjectSprint(s *api.Sprint, mode artifact.Type) *SprintArtifact {
	a := &SprintArtifact{
		ID:    s.ID,
		Name:  s.Name,
		State: s.State,
	}
	if mode.IsFull() {
		if s.StartDate != nil {
			a.StartDate = s.StartDate.Format(time.RFC3339)
		}
		if s.EndDate != nil {
			a.EndDate = s.EndDate.Format(time.RFC3339)
		}
		if s.CompleteDate != nil {
			a.CompleteDate = s.CompleteDate.Format(time.RFC3339)
		}
		a.Goal = s.Goal
		a.BoardID = &s.OriginBoardID
	}
	return a
}

// ProjectSprints projects a slice of api.Sprint to SprintArtifacts.
func ProjectSprints(sprints []api.Sprint, mode artifact.Type) []*SprintArtifact {
	result := make([]*SprintArtifact, len(sprints))
	for i := range sprints {
		result[i] = ProjectSprint(&sprints[i], mode)
	}
	return result
}
