package artifact

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/open-cli-collective/atlassian-go/artifact"
	"github.com/open-cli-collective/atlassian-go/testutil"

	"github.com/open-cli-collective/jira-ticket-cli/api"
)

func TestProjectComment_AgentMode(t *testing.T) {
	t.Parallel()

	comment := &api.Comment{
		ID:      "12345",
		Author:  api.User{DisplayName: "John Doe"},
		Body:    api.NewADFDocument("Short comment body"),
		Created: "2024-01-15T10:00:00.000Z",
		Updated: "2024-01-16T11:00:00.000Z",
	}

	art := ProjectComment(comment, artifact.Agent)

	// Agent fields populated
	testutil.Equal(t, art.ID, "12345")
	testutil.Equal(t, art.Author, "John Doe")
	testutil.Equal(t, art.Created, "2024-01-15")
	testutil.Equal(t, art.Body, "Short comment body")

	// Full-only fields empty
	testutil.Equal(t, art.Updated, "")
}

func TestProjectComment_AgentMode_TruncatesLongBody(t *testing.T) {
	t.Parallel()

	longBody := strings.Repeat("A", 300)
	comment := &api.Comment{
		ID:      "12345",
		Author:  api.User{DisplayName: "John Doe"},
		Body:    api.NewADFDocument(longBody),
		Created: "2024-01-15T10:00:00.000Z",
	}

	art := ProjectComment(comment, artifact.Agent)

	// Body should be truncated to 200 chars + "..."
	testutil.Equal(t, len(art.Body), 203)
	testutil.True(t, strings.HasSuffix(art.Body, "..."))
}

func TestProjectComment_FullMode(t *testing.T) {
	t.Parallel()

	longBody := strings.Repeat("A", 300)
	comment := &api.Comment{
		ID:      "12345",
		Author:  api.User{DisplayName: "John Doe"},
		Body:    api.NewADFDocument(longBody),
		Created: "2024-01-15T10:00:00.000Z",
		Updated: "2024-01-16T11:00:00.000Z",
	}

	art := ProjectComment(comment, artifact.Full)

	// Agent fields populated
	testutil.Equal(t, art.ID, "12345")
	testutil.Equal(t, art.Author, "John Doe")
	testutil.Equal(t, art.Created, "2024-01-15")

	// Full mode: body not truncated
	testutil.Equal(t, len(art.Body), 300)
	testutil.Equal(t, art.Body, longBody)

	// Full-only fields populated
	testutil.Equal(t, art.Updated, "2024-01-16")
}

func TestProjectComment_NilBody(t *testing.T) {
	t.Parallel()

	comment := &api.Comment{
		ID:      "12345",
		Author:  api.User{DisplayName: "John Doe"},
		Body:    nil,
		Created: "2024-01-15T10:00:00.000Z",
	}

	art := ProjectComment(comment, artifact.Agent)

	testutil.Equal(t, art.Body, "")
}

func TestProjectComment_JSONSerialization(t *testing.T) {
	t.Parallel()

	t.Run("agent mode omits updated", func(t *testing.T) {
		t.Parallel()
		comment := &api.Comment{
			ID:      "123",
			Author:  api.User{DisplayName: "Test"},
			Body:    api.NewADFDocument("test"),
			Created: "2024-01-15T10:00:00.000Z",
			Updated: "2024-01-16T10:00:00.000Z",
		}
		art := ProjectComment(comment, artifact.Agent)

		data, err := json.Marshal(art)
		testutil.RequireNoError(t, err)

		var parsed map[string]any
		err = json.Unmarshal(data, &parsed)
		testutil.RequireNoError(t, err)

		_, exists := parsed["updated"]
		testutil.False(t, exists) // Should not be present in agent mode
	})

	t.Run("full mode includes updated", func(t *testing.T) {
		t.Parallel()
		comment := &api.Comment{
			ID:      "123",
			Author:  api.User{DisplayName: "Test"},
			Body:    api.NewADFDocument("test"),
			Created: "2024-01-15T10:00:00.000Z",
			Updated: "2024-01-16T10:00:00.000Z",
		}
		art := ProjectComment(comment, artifact.Full)

		data, err := json.Marshal(art)
		testutil.RequireNoError(t, err)

		var parsed map[string]any
		err = json.Unmarshal(data, &parsed)
		testutil.RequireNoError(t, err)

		_, exists := parsed["updated"]
		testutil.True(t, exists)
	})
}

func TestProjectComments(t *testing.T) {
	t.Parallel()

	comments := []api.Comment{
		{ID: "1", Author: api.User{DisplayName: "User 1"}, Body: api.NewADFDocument("Comment 1"), Created: "2024-01-15T10:00:00.000Z"},
		{ID: "2", Author: api.User{DisplayName: "User 2"}, Body: api.NewADFDocument("Comment 2"), Created: "2024-01-16T10:00:00.000Z"},
	}

	arts := ProjectComments(comments, artifact.Agent)

	testutil.Equal(t, len(arts), 2)
	testutil.Equal(t, arts[0].ID, "1")
	testutil.Equal(t, arts[1].ID, "2")
}

func TestProjectComments_Empty(t *testing.T) {
	t.Parallel()

	var comments []api.Comment
	arts := ProjectComments(comments, artifact.Agent)

	testutil.Equal(t, len(arts), 0)
	testutil.NotNil(t, arts)
}

func TestFormatDate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input    string
		expected string
	}{
		{"2024-01-15T10:00:00.000Z", "2024-01-15"},
		{"2024-01-15", "2024-01-15"},
		{"short", "short"},
		{"", ""},
	}

	for _, tt := range tests {
		result := formatDate(tt.input)
		testutil.Equal(t, result, tt.expected)
	}
}
