// Package comments provides CLI commands for managing Jira issue comments.
package comments

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/atlassian-go/artifact"
	"github.com/open-cli-collective/atlassian-go/present"
	"github.com/open-cli-collective/atlassian-go/view"

	"github.com/open-cli-collective/jira-ticket-cli/api"
	jtkartifact "github.com/open-cli-collective/jira-ticket-cli/internal/artifact"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
	jtkpresent "github.com/open-cli-collective/jira-ticket-cli/internal/present"
	"github.com/open-cli-collective/jira-ticket-cli/internal/present/projection"
	"github.com/open-cli-collective/jira-ticket-cli/internal/text"
)

// noFieldFetch is the projection.Resolve fetcher for comments. Comment
// fields are not Jira issue fields, so there is no metadata to fetch;
// returning nil routes deferred tokens cleanly to UnknownFieldError
// rather than UnrenderedFieldError or a real network call against
// /rest/api/3/field.
func noFieldFetch(_ context.Context) ([]api.Field, error) { return nil, nil }

// Register registers the comments commands
func Register(parent *cobra.Command, opts *root.Options) {
	cmd := &cobra.Command{
		Use:     "comments",
		Aliases: []string{"comment", "c"},
		Short:   "Manage issue comments",
		Long:    "Commands for viewing and adding comments on issues.",
	}

	cmd.AddCommand(newListCmd(opts))
	cmd.AddCommand(newAddCmd(opts))
	cmd.AddCommand(newDeleteCmd(opts))

	parent.AddCommand(cmd)
}

func newListCmd(opts *root.Options) *cobra.Command {
	var maxResults int
	var noTruncate bool
	var fieldsFlag string

	cmd := &cobra.Command{
		Use:   "list <issue-key>",
		Short: "List comments on an issue",
		Long:  "List all comments on a specific issue.",
		Example: `  jtk comments list PROJ-123
  jtk comments list PROJ-123 --fulltext
  jtk comments list PROJ-123 --fields ID,AUTHOR
  jtk comments list PROJ-123 --fulltext --fields Body`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runList(cmd.Context(), opts, args[0], maxResults, noTruncate || opts.IsFullText(), fieldsFlag)
		},
	}

	cmd.Flags().IntVarP(&maxResults, "max", "m", 50, "Maximum number of comments")
	cmd.Flags().BoolVar(&noTruncate, "no-truncate", false, "Show full comment bodies without truncation")
	_ = cmd.Flags().MarkDeprecated("no-truncate", "use --fulltext instead")
	cmd.Flags().StringVar(&fieldsFlag, "fields", "", "Comma-separated display fields (labels)")

	return cmd
}

func runList(ctx context.Context, opts *root.Options, issueKey string, maxResults int, noTruncate bool, fieldsFlag string) error {
	v := opts.View()

	client, err := opts.APIClient()
	if err != nil {
		return err
	}

	// Validate flag combinations and resolve --fields before any network call,
	// mirroring runGet ordering. --id suppresses both gates (projection and
	// JSON-vs-fields) because it collapses output to bare IDs regardless.
	idOnly := opts.EmitIDOnly()
	var selected []projection.ColumnSpec
	var projected bool
	if !idOnly {
		if fieldsFlag != "" && v.Format == view.FormatJSON {
			return jtkpresent.ErrFieldsWithJSON
		}
		spec := jtkpresent.CommentListSpec
		if noTruncate {
			spec = jtkpresent.CommentDetailSpec
		}
		selected, projected, err = projection.Resolve(
			ctx,
			spec,
			opts.IsExtended(),
			fieldsFlag,
			noFieldFetch,
			"comments list",
		)
		if err != nil {
			return err
		}
	}

	result, err := client.GetComments(ctx, issueKey, 0, maxResults)
	if err != nil {
		return err
	}

	hasMore := commentsHasMore(result.Total, result.StartAt, len(result.Comments), maxResults)

	if idOnly {
		ids := make([]string, len(result.Comments))
		for i, c := range result.Comments {
			ids[i] = c.ID
		}
		return jtkpresent.EmitIDsWithPagination(opts, ids, hasMore)
	}

	if len(result.Comments) == 0 {
		model := jtkpresent.CommentPresenter{}.PresentEmpty(issueKey)
		model.Sections = jtkpresent.AppendPaginationHint(model.Sections, hasMore)
		return jtkpresent.Emit(opts, model)
	}

	if v.Format == view.FormatJSON {
		arts := jtkartifact.ProjectComments(result.Comments, opts.ArtifactMode())
		return v.RenderArtifactList(artifact.NewListResult(arts, hasMore))
	}

	extended := opts.IsExtended()
	var model *present.OutputModel
	if noTruncate {
		model = jtkpresent.CommentPresenter{}.PresentListFullWithPagination(result.Comments, extended, hasMore)
		if projected {
			projectAllDetailSectionsInModel(model, selected)
		}
	} else {
		model = jtkpresent.CommentPresenter{}.PresentListWithPagination(result.Comments, extended, hasMore)
		if projected {
			projection.ApplyToTableInModel(model, selected)
		}
	}
	return jtkpresent.Emit(opts, model)
}

// projectAllDetailSectionsInModel rewrites every DetailSection of model
// to the selected fields, leaving non-Detail sections (e.g. the
// pagination MessageSection) untouched.
func projectAllDetailSectionsInModel(model *present.OutputModel, selected []projection.ColumnSpec) {
	for i, s := range model.Sections {
		if ds, ok := s.(*present.DetailSection); ok {
			model.Sections[i] = projection.ProjectDetail(ds, selected)
		}
	}
}

// commentsHasMore computes pagination using the authoritative API metadata,
// falling back to a full-page heuristic when Total is unavailable (Jira Cloud
// occasionally returns Total=0).
//
// When got==0 there are definitionally no more pages, even with the
// heuristic — without this guard, degenerate inputs like (0,0,0,0) would
// falsely report hasMore=true.
func commentsHasMore(total, startAt, got, maxResults int) bool {
	if got == 0 {
		return false
	}
	if total > 0 {
		return startAt+got < total
	}
	return got == maxResults
}

func newAddCmd(opts *root.Options) *cobra.Command {
	var body string

	cmd := &cobra.Command{
		Use:     "add <issue-key>",
		Short:   "Add a comment to an issue",
		Long:    "Add a new comment to an issue.",
		Example: `  jtk comments add PROJ-123 --body "This is my comment"`,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAdd(cmd.Context(), opts, args[0], body)
		},
	}

	cmd.Flags().StringVarP(&body, "body", "b", "", "Comment text (required)")
	_ = cmd.MarkFlagRequired("body")

	return cmd
}

func runAdd(ctx context.Context, opts *root.Options, issueKey, body string) error {
	v := opts.View()

	client, err := opts.APIClient()
	if err != nil {
		return err
	}

	comment, err := client.AddComment(ctx, issueKey, text.InterpretEscapes(body))
	if err != nil {
		return err
	}

	if opts.EmitIDOnly() {
		return jtkpresent.EmitIDs(opts, []string{comment.ID})
	}

	if opts.Output == "json" {
		return v.JSON(comment)
	}

	return jtkpresent.Emit(opts, jtkpresent.CommentPresenter{}.PresentAddedDetail(issueKey, comment))
}

func newDeleteCmd(opts *root.Options) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "delete <issue-key> <comment-id>",
		Short:   "Delete a comment from an issue",
		Long:    "Delete an existing comment from an issue.",
		Example: `  jtk comments delete PROJ-123 12345`,
		Args:    cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDelete(cmd.Context(), opts, args[0], args[1])
		},
	}

	return cmd
}

func runDelete(ctx context.Context, opts *root.Options, issueKey, commentID string) error {
	client, err := opts.APIClient()
	if err != nil {
		return err
	}

	if err := client.DeleteComment(ctx, issueKey, commentID); err != nil {
		return err
	}

	return jtkpresent.Emit(opts, jtkpresent.CommentPresenter{}.PresentDeleted(commentID, issueKey))
}
