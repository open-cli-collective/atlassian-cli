package issues

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/atlassian-go/view"

	jtkartifact "github.com/open-cli-collective/jira-ticket-cli/internal/artifact"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cache"
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
		transitions, transErr := client.GetTransitions(ctx, issueKey)
		watchers, _ := client.GetWatchers(ctx, issueKey)
		fields, _ := cache.GetFieldsCacheFirst(ctx, client)
		dctx := &jtkpresent.DetailContext{
			Transitions:       transitions,
			TransitionsFailed: transErr != nil,
			Watchers:          watchers,
			Fields:            fields,
		}

		if projected {
			model := presenter.PresentDetailProjection(issue, client.IssueURL(issue.Key), noTruncate, dctx)
			jtkpresent.AppendDynamicDetailFields(model, issue, projection.DynamicSpecs(selected))
			projection.ApplyToDetailInModel(model, selected)
			return jtkpresent.Emit(opts, model)
		}
		model := presenter.PresentDetailExtended(issue, client.IssueURL(issue.Key), dctx)
		return jtkpresent.Emit(opts, model)
	}

	if projected {
		model := presenter.PresentDetailProjection(issue, client.IssueURL(issue.Key), noTruncate, nil)
		jtkpresent.AppendDynamicDetailFields(model, issue, projection.DynamicSpecs(selected))
		projection.ApplyToDetailInModel(model, selected)
		return jtkpresent.Emit(opts, model)
	}

	model := presenter.PresentDetail(issue, client.IssueURL(issue.Key), false, noTruncate)
	return jtkpresent.Emit(opts, model)
}
