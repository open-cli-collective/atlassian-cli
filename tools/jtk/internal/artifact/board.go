package artifact

import (
	"github.com/open-cli-collective/atlassian-go/artifact"

	"github.com/open-cli-collective/jira-ticket-cli/api"
)

// BoardArtifact is the projected output for an agile board.
type BoardArtifact struct {
	// Agent fields - essential for board selection
	ID   int    `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`

	// Full-only fields
	ProjectKey  string `json:"projectKey,omitempty"`
	ProjectName string `json:"projectName,omitempty"`
}

// ProjectBoard projects an api.Board to a BoardArtifact.
func ProjectBoard(b *api.Board, mode artifact.Type) *BoardArtifact {
	a := &BoardArtifact{
		ID:   b.ID,
		Name: b.Name,
		Type: b.Type,
	}
	if mode.IsFull() {
		a.ProjectKey = b.Location.ProjectKey
		a.ProjectName = b.Location.ProjectName
	}
	return a
}

// ProjectBoards projects a slice of api.Board to BoardArtifacts.
func ProjectBoards(boards []api.Board, mode artifact.Type) []*BoardArtifact {
	result := make([]*BoardArtifact, len(boards))
	for i := range boards {
		result[i] = ProjectBoard(&boards[i], mode)
	}
	return result
}
