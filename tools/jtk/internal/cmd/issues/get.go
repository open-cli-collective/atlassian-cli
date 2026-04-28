package issues

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/atlassian-go/present"
	"github.com/open-cli-collective/atlassian-go/view"

	"github.com/open-cli-collective/jira-ticket-cli/api"
	jtkartifact "github.com/open-cli-collective/jira-ticket-cli/internal/artifact"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cache"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
	jtkpresent "github.com/open-cli-collective/jira-ticket-cli/internal/present"
	"github.com/open-cli-collective/jira-ticket-cli/internal/present/projection"
)

func newGetCmd(opts *root.Options) *cobra.Command {
	var noTruncate bool
	var fieldsFlag string
	var customFields bool

	cmd := &cobra.Command{
		Use:   "get <issue-key>",
		Short: "Get issue details",
		Long:  "Retrieve and display details for a specific issue.",
		Example: `  jtk issues get PROJ-123
  jtk issues get PROJ-123 --fulltext
  jtk issues get PROJ-123 --id
  jtk issues get PROJ-123 --fields Status,Assignee
  jtk issues get PROJ-123 --fields "Issue Type"
  jtk issues get PROJ-123 --custom-fields`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGet(cmd.Context(), opts, args[0], noTruncate || opts.IsFullText(), fieldsFlag, customFields)
		},
	}

	cmd.Flags().BoolVar(&noTruncate, "no-truncate", false, "Show full description without truncation")
	_ = cmd.Flags().MarkDeprecated("no-truncate", "use --fulltext instead")
	cmd.Flags().StringVar(&fieldsFlag, "fields", "", "Comma-separated display fields (labels, Jira field IDs, or human names)")
	cmd.Flags().BoolVar(&customFields, "custom-fields", false, "Append custom fields section to output")

	return cmd
}

func runGet(ctx context.Context, opts *root.Options, issueKey string, noTruncate bool, fieldsFlag string, customFields bool) error {
	v := opts.View()

	client, err := opts.APIClient()
	if err != nil {
		return err
	}

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

	if customFields && v.Format == view.FormatJSON {
		return jtkpresent.ErrFieldsWithJSON
	}

	selected, projected, err := projection.Resolve(
		ctx,
		jtkpresent.IssueDetailSpec,
		opts.IsExtended(),
		fieldsFlag,
		fieldsFetcher(client),
		"issues get",
	)
	if err != nil {
		return err
	}

	issue, err := client.GetIssue(ctx, issueKey)
	if err != nil {
		return err
	}

	if v.Format == view.FormatJSON {
		return v.RenderArtifact(jtkartifact.ProjectIssue(issue, opts.ArtifactMode()))
	}

	presenter := jtkpresent.IssuePresenter{}

	if opts.IsExtended() {
		noTruncate = true
	}

	if projected {
		model := presenter.PresentDetailProjection(issue, client.IssueURL(issue.Key), noTruncate)
		jtkpresent.AppendDynamicDetailFields(model, issue, projection.DynamicSpecs(selected))
		projection.ApplyToDetailInModel(model, selected)
		if customFields {
			appendCustomFields(ctx, client, issue, model)
		}
		return jtkpresent.Emit(opts, model)
	}
	model := presenter.PresentDetail(issue, client.IssueURL(issue.Key), opts.IsExtended(), noTruncate)
	if customFields {
		appendCustomFields(ctx, client, issue, model)
	}
	return jtkpresent.Emit(opts, model)
}

func appendCustomFields(ctx context.Context, client *api.Client, issue *api.Issue, model *present.OutputModel) {
	fields, err := cache.GetFieldsCacheFirst(ctx, client)
	if err != nil {
		return
	}
	entries := api.ExtractIssueFieldValues(issue, fields)
	jtkpresent.AppendCustomFieldsSection(model, entries)
}
