package artifact

import (
	"github.com/open-cli-collective/atlassian-go/artifact"

	"github.com/open-cli-collective/confluence-cli/api"
)

// SearchResultArtifact represents a search result artifact.
type SearchResultArtifact struct {
	ID         string `json:"id"`
	Title      string `json:"title"`
	Type       string `json:"type"`
	SpaceName  string `json:"spaceName"`
	Excerpt    string `json:"excerpt,omitempty"`
	URL        string `json:"url,omitempty"`
	ModifiedAt string `json:"modifiedAt,omitempty"`
}

// maxExcerptRunes is the maximum number of runes for excerpt in agent mode.
const maxExcerptRunes = 200

// ProjectSearchResult projects an api.SearchResult to a SearchResultArtifact.
func ProjectSearchResult(r *api.SearchResult, mode artifact.Type) *SearchResultArtifact {
	art := &SearchResultArtifact{
		ID:        r.Content.ID,
		Title:     r.Title,
		Type:      r.Content.Type,
		SpaceName: r.ResultGlobalContainer.Title,
	}

	// Truncate excerpt in agent mode
	excerpt := r.Excerpt
	if mode == artifact.Agent {
		runes := []rune(excerpt)
		if len(runes) > maxExcerptRunes {
			excerpt = string(runes[:maxExcerptRunes]) + "..."
		}
	}
	if excerpt != "" {
		art.Excerpt = excerpt
	}

	if mode == artifact.Full {
		art.URL = r.URL
		art.ModifiedAt = r.LastModified
	}

	return art
}

// ProjectSearchResults projects a slice of api.SearchResult to SearchResultArtifacts.
func ProjectSearchResults(results []api.SearchResult, mode artifact.Type) []*SearchResultArtifact {
	arts := make([]*SearchResultArtifact, len(results))
	for i := range results {
		arts[i] = ProjectSearchResult(&results[i], mode)
	}
	return arts
}
