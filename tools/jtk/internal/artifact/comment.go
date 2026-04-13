package artifact

import (
	"strings"

	"github.com/open-cli-collective/atlassian-go/artifact"

	"github.com/open-cli-collective/jira-ticket-cli/api"
)

// CommentArtifact is the projected output for a comment.
// Body is included in agent mode (truncated) because it's the primary semantic content.
type CommentArtifact struct {
	// Agent fields - body is primary content, truncated for triage
	ID      string `json:"id"`
	Author  string `json:"author"`
	Created string `json:"created"`
	Body    string `json:"body"` // Truncated in agent mode, full in full mode

	// Full-only fields
	Updated string `json:"updated,omitempty"`
}

// ProjectComment projects an api.Comment to a CommentArtifact.
func ProjectComment(c *api.Comment, mode artifact.Type) *CommentArtifact {
	body := ""
	if c.Body != nil {
		body = strings.TrimSpace(c.Body.ToPlainText())
	}

	// In agent mode, truncate body for triage readability.
	// Use rune-based truncation to avoid cutting multi-byte UTF-8 characters.
	if !mode.IsFull() {
		runes := []rune(body)
		if len(runes) > 200 {
			body = string(runes[:200]) + "..."
		}
	}

	a := &CommentArtifact{
		ID:      c.ID,
		Author:  c.Author.DisplayName,
		Created: formatDate(c.Created),
		Body:    body,
	}

	if mode.IsFull() {
		a.Updated = formatDate(c.Updated)
	}

	return a
}

// ProjectComments projects a slice of api.Comment to CommentArtifacts.
func ProjectComments(comments []api.Comment, mode artifact.Type) []*CommentArtifact {
	result := make([]*CommentArtifact, len(comments))
	for i := range comments {
		result[i] = ProjectComment(&comments[i], mode)
	}
	return result
}

// formatDate extracts just the date portion from an ISO timestamp.
func formatDate(timestamp string) string {
	if len(timestamp) >= 10 {
		return timestamp[:10]
	}
	return timestamp
}
