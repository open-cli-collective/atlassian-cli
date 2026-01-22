package comments

import (
	"github.com/spf13/cobra"

	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
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

	cmd := &cobra.Command{
		Use:     "list <issue-key>",
		Short:   "List comments on an issue",
		Long:    "List all comments on a specific issue.",
		Example: `  jira-ticket-cli comments list PROJ-123`,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runList(opts, args[0], maxResults)
		},
	}

	cmd.Flags().IntVarP(&maxResults, "max", "m", 50, "Maximum number of comments")

	return cmd
}

func runList(opts *root.Options, issueKey string, maxResults int) error {
	v := opts.View()

	client, err := opts.APIClient()
	if err != nil {
		return err
	}

	result, err := client.GetComments(issueKey, 0, maxResults)
	if err != nil {
		return err
	}

	if len(result.Comments) == 0 {
		v.Info("No comments on %s", issueKey)
		return nil
	}

	if opts.Output == "json" {
		return v.JSON(result.Comments)
	}

	headers := []string{"ID", "AUTHOR", "CREATED", "BODY"}
	var rows [][]string

	for _, c := range result.Comments {
		body := ""
		if c.Body != nil {
			body = c.Body.ToPlainText()
			if len(body) > 50 {
				body = body[:50] + "..."
			}
		}

		rows = append(rows, []string{
			c.ID,
			c.Author.DisplayName,
			formatTime(c.Created),
			body,
		})
	}

	return v.Table(headers, rows)
}

func newAddCmd(opts *root.Options) *cobra.Command {
	var body string

	cmd := &cobra.Command{
		Use:     "add <issue-key>",
		Short:   "Add a comment to an issue",
		Long:    "Add a new comment to an issue.",
		Example: `  jira-ticket-cli comments add PROJ-123 --body "This is my comment"`,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAdd(opts, args[0], body)
		},
	}

	cmd.Flags().StringVarP(&body, "body", "b", "", "Comment text (required)")
	_ = cmd.MarkFlagRequired("body")

	return cmd
}

func runAdd(opts *root.Options, issueKey, body string) error {
	v := opts.View()

	client, err := opts.APIClient()
	if err != nil {
		return err
	}

	comment, err := client.AddComment(issueKey, body)
	if err != nil {
		return err
	}

	if opts.Output == "json" {
		return v.JSON(comment)
	}

	v.Success("Added comment %s to %s", comment.ID, issueKey)
	return nil
}

func newDeleteCmd(opts *root.Options) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "delete <issue-key> <comment-id>",
		Short:   "Delete a comment from an issue",
		Long:    "Delete an existing comment from an issue.",
		Example: `  jira-ticket-cli comments delete PROJ-123 12345`,
		Args:    cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDelete(opts, args[0], args[1])
		},
	}

	return cmd
}

func runDelete(opts *root.Options, issueKey, commentID string) error {
	v := opts.View()

	client, err := opts.APIClient()
	if err != nil {
		return err
	}

	if err := client.DeleteComment(issueKey, commentID); err != nil {
		return err
	}

	if opts.Output == "json" {
		return v.JSON(map[string]string{"status": "deleted", "commentId": commentID})
	}

	v.Success("Deleted comment %s from %s", commentID, issueKey)
	return nil
}

func formatTime(t string) string {
	// Jira returns ISO 8601 format, just show date
	if len(t) >= 10 {
		return t[:10]
	}
	return t
}
