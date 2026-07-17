// Package comments provides CLI commands for managing Jira issue comments.
package comments

import (
	"context"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/jira-ticket-cli/api"
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
	var nextPageToken string
	var fieldsFlag string

	cmd := &cobra.Command{
		Use:   "list <issue-key>",
		Short: "List comments on an issue",
		Long:  "List one page of comments on a specific issue.",
		Example: `  jtk comments list PROJ-123
  jtk comments list PROJ-123 --fulltext
  jtk comments list PROJ-123 --fields ID,AUTHOR
  jtk comments list PROJ-123 --fulltext --fields Body
  jtk comments list PROJ-123 --next-page-token 50`,
		Args: func(cmd *cobra.Command, args []string) error {
			if err := cobra.ExactArgs(1)(cmd, args); err != nil {
				return err
			}
			return jtkpresent.ValidateMax(maxResults)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runList(cmd.Context(), opts, args[0], maxResults, nextPageToken, opts.IsFullText(), fieldsFlag)
		},
	}

	cmd.Flags().IntVarP(&maxResults, "max", "m", 50, "Page size")
	cmd.Flags().StringVar(&nextPageToken, "next-page-token", "", "Decimal startAt for the next page")
	cmd.Flags().StringVar(&fieldsFlag, "fields", "", "Comma-separated display fields (labels)")

	return cmd
}

func runList(ctx context.Context, opts *root.Options, issueKey string, maxResults int, nextPageToken string, noTruncate bool, fieldsFlag string) error {
	startAt, err := jtkpresent.ParseStartAtToken(nextPageToken)
	if err != nil {
		return err
	}

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
	spec := jtkpresent.CommentListSpec
	if !idOnly {
		selected, projected, err = projection.Resolve(
			ctx,
			spec,
			fieldsFlag,
			noFieldFetch,
			"comments list",
		)
		if err != nil {
			return err
		}
	}

	result, err := client.GetComments(ctx, issueKey, startAt, maxResults)
	if err != nil {
		return err
	}

	hasMore := commentsHasMore(result.Total, result.StartAt, len(result.Comments), maxResults)
	nextToken := ""
	if hasMore {
		nextToken = strconv.Itoa(result.StartAt + len(result.Comments))
	}

	if idOnly {
		ids := make([]string, len(result.Comments))
		for i, c := range result.Comments {
			ids[i] = c.ID
		}
		return jtkpresent.EmitIDsWithPaginationToken(opts, ids, hasMore, nextToken)
	}

	if len(result.Comments) == 0 {
		model := jtkpresent.CommentPresenter{}.PresentEmpty(issueKey)
		model.Sections = jtkpresent.AppendPaginationHintWithToken(model.Sections, hasMore, nextToken)
		return jtkpresent.Emit(opts, model)
	}

	model := jtkpresent.CommentPresenter{}.PresentListWithPagination(
		result.Comments,
		projection.HasOptionalFields(selected, spec),
		noTruncate,
		hasMore,
		nextToken,
	)
	if projected {
		projection.ApplyToTableInModel(model, selected)
	}
	return jtkpresent.Emit(opts, model)
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
