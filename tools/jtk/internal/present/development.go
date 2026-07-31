package present

import (
	"fmt"
	"strings"

	"github.com/open-cli-collective/atlassian-go/present"

	"github.com/open-cli-collective/jira-ticket-cli/api"
)

// DevelopmentPresenter renders Jira Development-panel pull requests.
type DevelopmentPresenter struct{}

// Present renders the issue summary followed by one row per pull request.
func (DevelopmentPresenter) Present(development *api.Development) *present.OutputModel {
	state := ""
	if len(development.PullRequests) > 0 && development.PullRequestState != "" {
		state = " (" + development.PullRequestState + ")"
	}
	builds := fmt.Sprintf("%d", development.Builds)
	if development.Builds > 0 {
		builds += fmt.Sprintf(" (%d successful)", development.SuccessfulBuilds)
	}

	sections := []present.Section{
		msg(fmt.Sprintf("%s  Development", development.IssueKey)),
		msg(fmt.Sprintf(
			"Pull Requests: %d%s   Commits: %d   Builds: %s",
			len(development.PullRequests), state, development.Commits, builds,
		)),
	}
	if len(development.Providers) > 0 {
		sections = append(sections, msg("Providers: "+strings.Join(development.Providers, ", ")))
	}
	if len(development.PullRequests) == 0 {
		return &present.OutputModel{Sections: sections}
	}

	rows := make([]present.Row, len(development.PullRequests))
	for i, pr := range development.PullRequests {
		id := pr.ID
		if id != "" && !strings.HasPrefix(id, "#") {
			id = "#" + id
		}
		updated := pr.LastUpdate
		if len(updated) >= 10 {
			updated = updated[:10]
		}
		rows[i] = present.Row{Cells: []string{
			OrDash(id),
			OrDash(pr.Repository),
			OrDash(pr.Status),
			OrDash(updated),
			OrDash(pr.Title),
			OrDash(pr.URL),
		}}
	}
	sections = append(sections,
		msg(""),
		&present.TableSection{
			Headers: []string{"PR", "REPOSITORY", "STATUS", "UPDATED", "TITLE", "URL"},
			Rows:    rows,
		},
	)
	return &present.OutputModel{Sections: sections}
}
