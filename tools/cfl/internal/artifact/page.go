// Package artifact provides artifact types for projecting API responses to
// structured output for CLI commands.
package artifact

import (
	"github.com/open-cli-collective/atlassian-go/artifact"

	"github.com/open-cli-collective/confluence-cli/api"
)

// PageArtifact represents a content-bearing page artifact for page view.
type PageArtifact struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	SpaceID   string `json:"spaceId"`
	SpaceKey  string `json:"spaceKey,omitempty"`
	ParentID  string `json:"parentId,omitempty"`
	Content   string `json:"content"`
	Version   int    `json:"version,omitempty"`
	CreatedAt string `json:"createdAt,omitempty"`
	AuthorID  string `json:"authorId,omitempty"`
}

// ProjectPage projects an api.Page to a PageArtifact.
// The spaceKey is passed separately since api.Page only has SpaceID.
// Content should be pre-transformed (XHTML or markdown) by the caller.
func ProjectPage(p *api.Page, spaceKey string, content string, mode artifact.Type) *PageArtifact {
	art := &PageArtifact{
		ID:       p.ID,
		Title:    p.Title,
		SpaceID:  p.SpaceID,
		SpaceKey: spaceKey,
		ParentID: p.ParentID,
		Content:  content,
	}

	if mode == artifact.Full {
		if p.Version != nil {
			art.Version = p.Version.Number
			if p.Version.CreatedAt != nil && !p.Version.CreatedAt.IsZero() {
				art.CreatedAt = p.Version.CreatedAt.UTC().Format("2006-01-02T15:04:05Z07:00")
			}
			art.AuthorID = p.Version.AuthorID
		}
	}

	return art
}

// PageListArtifact represents a metadata-only page artifact for page list.
type PageListArtifact struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	SpaceID  string `json:"spaceId"`
	Status   string `json:"status"`
	Version  int    `json:"version,omitempty"`
	ParentID string `json:"parentId,omitempty"`
}

// ProjectPageListItem projects an api.Page to a PageListArtifact.
func ProjectPageListItem(p *api.Page, mode artifact.Type) *PageListArtifact {
	art := &PageListArtifact{
		ID:      p.ID,
		Title:   p.Title,
		SpaceID: p.SpaceID,
		Status:  p.Status,
	}

	if mode == artifact.Full {
		if p.Version != nil {
			art.Version = p.Version.Number
		}
		art.ParentID = p.ParentID
	}

	return art
}

// ProjectPageListItems projects a slice of api.Page to PageListArtifacts.
func ProjectPageListItems(pages []api.Page, mode artifact.Type) []*PageListArtifact {
	arts := make([]*PageListArtifact, len(pages))
	for i := range pages {
		arts[i] = ProjectPageListItem(&pages[i], mode)
	}
	return arts
}
