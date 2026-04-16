// Package comments provides CLI commands for managing Jira issue comments.
package comments

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/atlassian-go/artifact"
	"github.com/open-cli-collective/atlassian-go/present"
	"github.com/open-cli-collective/atlassian-go/view"

	jtkartifact "github.com/open-cli-collective/jira-ticket-cli/internal/artifact"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
	jtkpresent "github.com/open-cli-collective/jira-ticket-cli/internal/present"
	"github.com/open-cli-collective/jira-ticket-cli/internal/text"
)

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

	cmd := &cobra.Command{
		Use:   "list <issue-key>",
		Short: "List comments on an issue",
		Long:  "List all comments on a specific issue.",
		Example: `  jtk comments list PROJ-123
  jtk comments list PROJ-123 --fulltext`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runList(cmd.Context(), opts, args[0], maxResults, noTruncate || opts.IsFullText())
		},
	}

	cmd.Flags().IntVarP(&maxResults, "max", "m", 50, "Maximum number of comments")
	cmd.Flags().BoolVar(&noTruncate, "no-truncate", false, "Show full comment bodies without truncation")
	_ = cmd.Flags().MarkDeprecated("no-truncate", "use --fulltext instead")

	return cmd
}

func runList(ctx context.Context, opts *root.Options, issueKey string, maxResults int, noTruncate bool) error {
	v := opts.View()

	client, err := opts.APIClient()
	if err != nil {
		return err
	}

	result, err := client.GetComments(ctx, issueKey, 0, maxResults)
	if err != nil {
		return err
	}

	if len(result.Comments) == 0 {
		model := jtkpresent.CommentPresenter{}.PresentEmpty(issueKey)
		out := present.Render(model, opts.RenderStyle())
		fmt.Fprint(opts.Stdout, out.Stdout)
		fmt.Fprint(opts.Stderr, out.Stderr)
		return nil
	}

	if v.Format == view.FormatJSON {
		arts := jtkartifact.ProjectComments(result.Comments, opts.ArtifactMode())
		// Use authoritative pagination metadata from API response.
		// Guard against Total==0 edge case in Jira Cloud by also checking
		// if we received a full page of results.
		hasMore := false
		if result.Total > 0 {
			hasMore = result.StartAt+len(result.Comments) < result.Total
		} else if len(result.Comments) == maxResults {
			// Total is 0 but we got a full page - likely more results exist
			hasMore = true
		}
		return v.RenderArtifactList(artifact.NewListResult(arts, hasMore))
	}

	var model *present.OutputModel
	if noTruncate {
		model = jtkpresent.CommentPresenter{}.PresentListFull(result.Comments)
	} else {
		model = jtkpresent.CommentPresenter{}.PresentList(result.Comments)
	}
	out := present.Render(model, opts.RenderStyle())
	fmt.Fprint(opts.Stdout, out.Stdout)
	fmt.Fprint(opts.Stderr, out.Stderr)
	return nil
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

	if opts.Output == "json" {
		return v.JSON(comment)
	}

	model := jtkpresent.CommentPresenter{}.PresentAdded(comment.ID, issueKey)
	out := present.Render(model, opts.RenderStyle())
	fmt.Fprint(opts.Stdout, out.Stdout)
	fmt.Fprint(opts.Stderr, out.Stderr)
	return nil
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
	v := opts.View()

	client, err := opts.APIClient()
	if err != nil {
		return err
	}

	if err := client.DeleteComment(ctx, issueKey, commentID); err != nil {
		return err
	}

	if opts.Output == "json" {
		return v.JSON(map[string]string{"status": "deleted", "commentId": commentID})
	}

	model := jtkpresent.CommentPresenter{}.PresentDeleted(commentID, issueKey)
	out := present.Render(model, opts.RenderStyle())
	fmt.Fprint(opts.Stdout, out.Stdout)
	fmt.Fprint(opts.Stderr, out.Stderr)
	return nil
}
