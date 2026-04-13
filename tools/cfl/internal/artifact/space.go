package artifact

import (
	"github.com/open-cli-collective/atlassian-go/artifact"

	"github.com/open-cli-collective/confluence-cli/api"
)

// SpaceArtifact represents a space artifact.
type SpaceArtifact struct {
	ID          string `json:"id"`
	Key         string `json:"key"`
	Name        string `json:"name"`
	Type        string `json:"type,omitempty"`
	Status      string `json:"status,omitempty"`
	Description string `json:"description,omitempty"`
}

// ProjectSpace projects an api.Space to a SpaceArtifact.
func ProjectSpace(s *api.Space, mode artifact.Type) *SpaceArtifact {
	art := &SpaceArtifact{
		ID:   s.ID,
		Key:  s.Key,
		Name: s.Name,
	}

	if mode == artifact.Full {
		art.Type = s.Type
		art.Status = s.Status
		if s.Description != nil && s.Description.Plain != nil {
			art.Description = s.Description.Plain.Value
		}
	}

	return art
}

// ProjectSpaces projects a slice of api.Space to SpaceArtifacts.
func ProjectSpaces(spaces []api.Space, mode artifact.Type) []*SpaceArtifact {
	arts := make([]*SpaceArtifact, len(spaces))
	for i := range spaces {
		arts[i] = ProjectSpace(&spaces[i], mode)
	}
	return arts
}
