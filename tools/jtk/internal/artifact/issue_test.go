package artifact

import (
	"testing"

	"github.com/open-cli-collective/atlassian-go/artifact"
	"github.com/open-cli-collective/atlassian-go/testutil"

	"github.com/open-cli-collective/jira-ticket-cli/api"
)

func TestProjectIssue_AgentMode(t *testing.T) {
	t.Parallel()

	issue := &api.Issue{
		Key: "PROJ-123",
		Fields: api.IssueFields{
			Summary:   "Fix the bug",
			Status:    &api.Status{Name: "In Progress"},
			IssueType: &api.IssueType{Name: "Bug"},
			Assignee:  &api.User{DisplayName: "John Doe"},
			Priority:  &api.Priority{Name: "High"},
			Project:   &api.Project{Key: "PROJ"},
		},
	}

	art := ProjectIssue(issue, artifact.Agent)

	// Agent fields populated
	testutil.Equal(t, art.Key, "PROJ-123")
	testutil.Equal(t, art.Summary, "Fix the bug")
	testutil.Equal(t, art.Status, "In Progress")
	testutil.Equal(t, art.Type, "Bug")
	testutil.Equal(t, art.Assignee, "John Doe")

	// Full-only fields empty
	testutil.Equal(t, art.Priority, "")
	testutil.Equal(t, art.Project, "")
}

func TestProjectIssue_FullMode(t *testing.T) {
	t.Parallel()

	issue := &api.Issue{
		Key: "PROJ-123",
		Fields: api.IssueFields{
			Summary:   "Fix the bug",
			Status:    &api.Status{Name: "Done"},
			IssueType: &api.IssueType{Name: "Task"},
			Assignee:  &api.User{DisplayName: "Jane Doe"},
			Priority:  &api.Priority{Name: "Critical"},
			Project:   &api.Project{Key: "PROJ"},
		},
	}

	art := ProjectIssue(issue, artifact.Full)

	// Agent fields populated
	testutil.Equal(t, art.Key, "PROJ-123")
	testutil.Equal(t, art.Summary, "Fix the bug")
	testutil.Equal(t, art.Status, "Done")
	testutil.Equal(t, art.Type, "Task")
	testutil.Equal(t, art.Assignee, "Jane Doe")

	// Full-only fields populated
	testutil.Equal(t, art.Priority, "Critical")
	testutil.Equal(t, art.Project, "PROJ")
}

func TestProjectIssue_NilFields(t *testing.T) {
	t.Parallel()

	issue := &api.Issue{
		Key: "PROJ-456",
		Fields: api.IssueFields{
			Summary: "Minimal issue",
			// All pointer fields nil
		},
	}

	art := ProjectIssue(issue, artifact.Full)

	testutil.Equal(t, art.Key, "PROJ-456")
	testutil.Equal(t, art.Summary, "Minimal issue")
	testutil.Equal(t, art.Status, "")
	testutil.Equal(t, art.Type, "")
	testutil.Equal(t, art.Assignee, "")
	testutil.Equal(t, art.Priority, "")
	testutil.Equal(t, art.Project, "")
}

func TestProjectIssues(t *testing.T) {
	t.Parallel()

	issues := []api.Issue{
		{Key: "PROJ-1", Fields: api.IssueFields{Summary: "Issue 1"}},
		{Key: "PROJ-2", Fields: api.IssueFields{Summary: "Issue 2"}},
		{Key: "PROJ-3", Fields: api.IssueFields{Summary: "Issue 3"}},
	}

	arts := ProjectIssues(issues, artifact.Agent)

	testutil.Equal(t, len(arts), 3)
	testutil.Equal(t, arts[0].Key, "PROJ-1")
	testutil.Equal(t, arts[1].Key, "PROJ-2")
	testutil.Equal(t, arts[2].Key, "PROJ-3")
}

func TestProjectIssues_Empty(t *testing.T) {
	t.Parallel()

	var issues []api.Issue
	arts := ProjectIssues(issues, artifact.Agent)

	testutil.Equal(t, len(arts), 0)
	testutil.NotNil(t, arts)
}
