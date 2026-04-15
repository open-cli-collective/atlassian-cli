package issues

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/atlassian-go/present"
	"github.com/open-cli-collective/atlassian-go/view"

	jtkartifact "github.com/open-cli-collective/jira-ticket-cli/internal/artifact"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
	jtkpresent "github.com/open-cli-collective/jira-ticket-cli/internal/present"
)

func newGetCmd(opts *root.Options) *cobra.Command {
	var noTruncate bool

	cmd := &cobra.Command{
		Use:   "get <issue-key>",
		Short: "Get issue details",
		Long:  "Retrieve and display details for a specific issue.",
		Example: `  jtk issues get PROJ-123
  jtk issues get PROJ-123 --no-truncate
  jtk issues get PROJ-123 -o json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGet(cmd.Context(), opts, args[0], noTruncate)
		},
	}

	cmd.Flags().BoolVar(&noTruncate, "no-truncate", false, "Show full description without truncation")

	return cmd
}

func runGet(ctx context.Context, opts *root.Options, issueKey string, noTruncate bool) error {
	v := opts.View()

	client, err := opts.APIClient()
	if err != nil {
		return err
	}

	issue, err := client.GetIssue(ctx, issueKey)
	if err != nil {
		return err
	}

	// For JSON output, return the projected artifact
	if v.Format == view.FormatJSON {
		return v.RenderArtifact(jtkartifact.ProjectIssue(issue, opts.ArtifactMode()))
	}

	// Text path: presenter → render → write
	model := jtkpresent.IssuePresenter{}.PresentDetail(issue, client.IssueURL(issue.Key), noTruncate)
	out := present.Render(model, opts.RenderStyle())
	_, _ = fmt.Fprint(opts.Stdout, out.Stdout)
	_, _ = fmt.Fprint(opts.Stderr, out.Stderr)
	return nil
}
