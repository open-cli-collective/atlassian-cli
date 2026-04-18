package issues

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/atlassian-go/view"

	jtkartifact "github.com/open-cli-collective/jira-ticket-cli/internal/artifact"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
	jtkpresent "github.com/open-cli-collective/jira-ticket-cli/internal/present"
	"github.com/open-cli-collective/jira-ticket-cli/internal/present/projection"
)

func newGetCmd(opts *root.Options) *cobra.Command {
	var noTruncate bool
	var fieldsFlag string

	cmd := &cobra.Command{
		Use:   "get <issue-key>",
		Short: "Get issue details",
		Long:  "Retrieve and display details for a specific issue.",
		Example: `  jtk issues get PROJ-123
  jtk issues get PROJ-123 --fulltext
  jtk issues get PROJ-123 --id
  jtk issues get PROJ-123 --fields Status,Assignee
  jtk issues get PROJ-123 --fields "Issue Type"`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGet(cmd.Context(), opts, args[0], noTruncate || opts.IsFullText(), fieldsFlag)
		},
	}

	cmd.Flags().BoolVar(&noTruncate, "no-truncate", false, "Show full description without truncation")
	_ = cmd.Flags().MarkDeprecated("no-truncate", "use --fulltext instead")
	cmd.Flags().StringVar(&fieldsFlag, "fields", "", "Comma-separated display fields (labels, Jira field IDs, or human names)")

	return cmd
}

func runGet(ctx context.Context, opts *root.Options, issueKey string, noTruncate bool, fieldsFlag string) error {
	v := opts.View()

	client, err := opts.APIClient()
	if err != nil {
		return err
	}

	// --id wins over --fields: skip projection entirely when --id is set so
	// we don't waste a GetFields() call on a token whose result will be
	// thrown away. Also defensively skip JSON + --fields error in this case
	// — --id also overrides --output json semantics.
	if opts.EmitIDOnly() {
		issue, err := client.GetIssue(ctx, issueKey)
		if err != nil {
			return err
		}
		return jtkpresent.EmitIDs(opts, []string{issue.Key})
	}

	if fieldsFlag != "" && v.Format == view.FormatJSON {
		return jtkpresent.ErrFieldsWithJSON
	}

	selected, projected, err := projection.Resolve(
		ctx,
		jtkpresent.IssueDetailSpec,
		opts.IsExtended(),
		fieldsFlag,
		client.GetFields,
		"issues get",
	)
	if err != nil {
		return err
	}

	// issues get does not minimize fetch — api.GetIssue has no field-selection
	// parameter, and adding one is out of scope for #233. Projection is
	// purely a display-time operation here.
	issue, err := client.GetIssue(ctx, issueKey)
	if err != nil {
		return err
	}

	// For JSON output, return the projected artifact
	if v.Format == view.FormatJSON {
		return v.RenderArtifact(jtkartifact.ProjectIssue(issue, opts.ArtifactMode()))
	}

	model := jtkpresent.IssuePresenter{}.PresentDetail(issue, client.IssueURL(issue.Key), noTruncate)
	if projected {
		projection.ApplyToDetailInModel(model, selected)
	}
	return jtkpresent.Emit(opts, model)
}
