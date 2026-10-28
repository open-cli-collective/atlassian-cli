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

	pullRequests := fmt.Sprintf("%d", len(development.PullRequests))
	if development.PullRequestsTruncated {
		pullRequests += "+"
	}

	sections := []present.Section{
		msg(fmt.Sprintf("%s  Development", development.IssueKey)),
		msg(fmt.Sprintf(
			"Pull Requests: %s%s   Commits: %d   Builds: %s",
			pullRequests, state, development.Commits, builds,
		)),
	}
	if len(development.Providers) > 0 {
		sections = append(sections, msg("Providers: "+strings.Join(development.Providers, ", ")))
	}
	if development.PullRequestsTruncated {
		sections = append(sections, DevelopmentTruncationWarning())
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

// DevelopmentTruncationWarning is the stderr advisory for a pull request list
// cut off at Jira's dev-status maximum.
func DevelopmentTruncationWarning() *present.MessageSection {
	return &present.MessageSection{
		Kind:    present.MessageWarning,
		Message: "Pull request list may be incomplete: Jira's dev-status API returns at most 100 per provider, and the paginated Graph Store lookup was unavailable.",
		Stream:  present.StreamStderr,
	}
}
