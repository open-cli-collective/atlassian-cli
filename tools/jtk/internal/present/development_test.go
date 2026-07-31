package present

import (
	"testing"

	"github.com/open-cli-collective/atlassian-go/present"
	"github.com/open-cli-collective/atlassian-go/testutil"

	"github.com/open-cli-collective/jira-ticket-cli/api"
)

func TestDevelopmentPresenter(t *testing.T) {
	t.Parallel()

	model := DevelopmentPresenter{}.Present(&api.Development{
		IssueKey:         "PROJ-123",
		PullRequestState: "MERGED",
		Commits:          5,
		Builds:           3,
		SuccessfulBuilds: 3,
		Providers:        []string{"GitHub"},
		PullRequests: []api.DevelopmentPullRequest{{
			ID:         "42",
			Title:      "Add feature",
			URL:        "https://github.com/owner/repo/pull/42",
			Status:     "MERGED",
			LastUpdate: "2026-07-31T12:00:00Z",
			Repository: "owner/repo",
		}},
	})

	testutil.Len(t, model.Sections, 5)
	summary := model.Sections[1].(*present.MessageSection)
	testutil.Equal(t, summary.Message, "Pull Requests: 1 (MERGED)   Commits: 5   Builds: 3 (3 successful)")
	table := model.Sections[4].(*present.TableSection)
	testutil.Equal(t, table.Headers[0], "PR")
	testutil.Equal(t, table.Headers[5], "URL")
	testutil.Equal(t, table.Rows[0].Cells[0], "#42")
	testutil.Equal(t, table.Rows[0].Cells[3], "2026-07-31")
	testutil.Equal(t, table.Rows[0].Cells[5], "https://github.com/owner/repo/pull/42")
}

func TestDevelopmentPresenter_NoPullRequests(t *testing.T) {
	t.Parallel()

	model := DevelopmentPresenter{}.Present(&api.Development{IssueKey: "PROJ-123"})
	testutil.Len(t, model.Sections, 2)
	summary := model.Sections[1].(*present.MessageSection)
	testutil.Equal(t, summary.Message, "Pull Requests: 0   Commits: 0   Builds: 0")
}
