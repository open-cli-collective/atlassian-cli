package artifact

import (
	"testing"

	"github.com/open-cli-collective/atlassian-go/artifact"
	"github.com/open-cli-collective/atlassian-go/testutil"

	"github.com/open-cli-collective/jira-ticket-cli/api"
)

func TestProjectBoard_AgentMode(t *testing.T) {
	t.Parallel()

	board := &api.Board{
		ID:   123,
		Name: "My Board",
		Type: "scrum",
		Location: api.BoardLocation{
			ProjectID:   456,
			ProjectKey:  "PROJ",
			ProjectName: "My Project",
		},
	}

	art := ProjectBoard(board, artifact.Agent)

	// Agent fields populated
	testutil.Equal(t, art.ID, 123)
	testutil.Equal(t, art.Name, "My Board")
	testutil.Equal(t, art.Type, "scrum")

	// Full-only fields empty
	testutil.Equal(t, art.ProjectKey, "")
	testutil.Equal(t, art.ProjectName, "")
}

func TestProjectBoard_FullMode(t *testing.T) {
	t.Parallel()

	board := &api.Board{
		ID:   123,
		Name: "My Board",
		Type: "kanban",
		Location: api.BoardLocation{
			ProjectID:   456,
			ProjectKey:  "PROJ",
			ProjectName: "My Project",
		},
	}

	art := ProjectBoard(board, artifact.Full)

	// Agent fields populated
	testutil.Equal(t, art.ID, 123)
	testutil.Equal(t, art.Name, "My Board")
	testutil.Equal(t, art.Type, "kanban")

	// Full-only fields populated
	testutil.Equal(t, art.ProjectKey, "PROJ")
	testutil.Equal(t, art.ProjectName, "My Project")
}

func TestProjectBoards(t *testing.T) {
	t.Parallel()

	boards := []api.Board{
		{ID: 1, Name: "Board 1", Type: "scrum"},
		{ID: 2, Name: "Board 2", Type: "kanban"},
	}

	arts := ProjectBoards(boards, artifact.Agent)

	testutil.Equal(t, len(arts), 2)
	testutil.Equal(t, arts[0].ID, 1)
	testutil.Equal(t, arts[0].Name, "Board 1")
	testutil.Equal(t, arts[1].ID, 2)
	testutil.Equal(t, arts[1].Name, "Board 2")
}

func TestProjectBoards_Empty(t *testing.T) {
	t.Parallel()

	var boards []api.Board
	arts := ProjectBoards(boards, artifact.Agent)

	testutil.Equal(t, len(arts), 0)
	testutil.NotNil(t, arts)
}
