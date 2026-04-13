package artifact

import (
	"testing"
	"time"

	"github.com/open-cli-collective/atlassian-go/artifact"
	"github.com/open-cli-collective/atlassian-go/testutil"

	"github.com/open-cli-collective/jira-ticket-cli/api"
)

func TestProjectSprint_AgentMode(t *testing.T) {
	t.Parallel()

	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 1, 14, 0, 0, 0, 0, time.UTC)

	sprint := &api.Sprint{
		ID:            123,
		Name:          "Sprint 1",
		State:         "active",
		StartDate:     &start,
		EndDate:       &end,
		Goal:          "Complete feature X",
		OriginBoardID: 456,
	}

	art := ProjectSprint(sprint, artifact.Agent)

	// Agent fields populated
	testutil.Equal(t, art.ID, 123)
	testutil.Equal(t, art.Name, "Sprint 1")
	testutil.Equal(t, art.State, "active")

	// Full-only fields empty/nil
	testutil.Equal(t, art.StartDate, "")
	testutil.Equal(t, art.EndDate, "")
	testutil.Equal(t, art.CompleteDate, "")
	testutil.Equal(t, art.Goal, "")
	testutil.Nil(t, art.BoardID)
}

func TestProjectSprint_FullMode(t *testing.T) {
	t.Parallel()

	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 1, 14, 0, 0, 0, 0, time.UTC)
	complete := time.Date(2024, 1, 13, 0, 0, 0, 0, time.UTC)

	sprint := &api.Sprint{
		ID:            123,
		Name:          "Sprint 1",
		State:         "closed",
		StartDate:     &start,
		EndDate:       &end,
		CompleteDate:  &complete,
		Goal:          "Complete feature X",
		OriginBoardID: 456,
	}

	art := ProjectSprint(sprint, artifact.Full)

	// Agent fields populated
	testutil.Equal(t, art.ID, 123)
	testutil.Equal(t, art.Name, "Sprint 1")
	testutil.Equal(t, art.State, "closed")

	// Full-only fields populated
	testutil.Equal(t, art.StartDate, "2024-01-01T00:00:00Z")
	testutil.Equal(t, art.EndDate, "2024-01-14T00:00:00Z")
	testutil.Equal(t, art.CompleteDate, "2024-01-13T00:00:00Z")
	testutil.Equal(t, art.Goal, "Complete feature X")
	testutil.NotNil(t, art.BoardID)
	testutil.Equal(t, *art.BoardID, 456)
}

func TestProjectSprint_NilDates(t *testing.T) {
	t.Parallel()

	sprint := &api.Sprint{
		ID:    123,
		Name:  "Sprint 1",
		State: "future",
	}

	art := ProjectSprint(sprint, artifact.Full)

	// Nil dates should result in empty strings
	testutil.Equal(t, art.StartDate, "")
	testutil.Equal(t, art.EndDate, "")
	testutil.Equal(t, art.CompleteDate, "")
}

func TestProjectSprints(t *testing.T) {
	t.Parallel()

	sprints := []api.Sprint{
		{ID: 1, Name: "Sprint 1", State: "closed"},
		{ID: 2, Name: "Sprint 2", State: "active"},
		{ID: 3, Name: "Sprint 3", State: "future"},
	}

	arts := ProjectSprints(sprints, artifact.Agent)

	testutil.Equal(t, len(arts), 3)
	testutil.Equal(t, arts[0].ID, 1)
	testutil.Equal(t, arts[1].ID, 2)
	testutil.Equal(t, arts[2].ID, 3)
}

func TestProjectSprints_Empty(t *testing.T) {
	t.Parallel()

	var sprints []api.Sprint
	arts := ProjectSprints(sprints, artifact.Agent)

	testutil.Equal(t, len(arts), 0)
	testutil.NotNil(t, arts) // Should be empty slice, not nil
}
