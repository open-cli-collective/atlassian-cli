package issues

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/atlassian-go/present"

	"github.com/open-cli-collective/jira-ticket-cli/api"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
	jtkpresent "github.com/open-cli-collective/jira-ticket-cli/internal/present"
)

func newMoveCmd(opts *root.Options) *cobra.Command {
	var targetProject string
	var targetType string
	var notify bool
	var wait bool

	cmd := &cobra.Command{
		Use:   "move <issue-key>...",
		Short: "Move issues to another project (Cloud only)",
		Long: `Move one or more issues to a different project and/or issue type.

This command uses the Jira Cloud bulk move API and is not available
on Jira Server or Data Center.

The operation is asynchronous - by default it waits for completion.
Use --no-wait to return immediately with the task ID.

Limitations:
- Maximum 1000 issues per request
- Subtasks must be moved with their parent or separately
- Some field values may need to be remapped manually`,
		Example: `  # Move a single issue to another project
  jtk issues move PROJ-123 --to-project NEWPROJ

  # Move to specific issue type
  jtk issues move PROJ-123 --to-project NEWPROJ --to-type Task

  # Move multiple issues
  jtk issues move PROJ-123 PROJ-124 PROJ-125 --to-project NEWPROJ

  # Move without waiting for completion
  jtk issues move PROJ-123 --to-project NEWPROJ --no-wait

  # Move without notifications
  jtk issues move PROJ-123 --to-project NEWPROJ --no-notify`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMove(cmd.Context(), opts, args, targetProject, targetType, notify, wait)
		},
	}

	cmd.Flags().StringVar(&targetProject, "to-project", "", "Target project key (required)")
	cmd.Flags().StringVar(&targetType, "to-type", "", "Target issue type (default: same as source)")
	cmd.Flags().BoolVar(&notify, "notify", true, "Send notifications for the move")
	cmd.Flags().BoolVar(&wait, "wait", true, "Wait for the move to complete")

	_ = cmd.MarkFlagRequired("to-project")

	return cmd
}

func runMove(ctx context.Context, opts *root.Options, issueKeys []string, targetProject, targetType string, notify, wait bool) error {
	ip := jtkpresent.IssuePresenter{}

	if len(issueKeys) > 1000 {
		return fmt.Errorf("cannot move more than 1000 issues at once (got %d)", len(issueKeys))
	}

	client, err := opts.APIClient()
	if err != nil {
		return err
	}

	// Get target project's issue types to validate or default the type
	issueTypes, err := client.GetProjectIssueTypes(ctx, targetProject)
	if err != nil {
		return fmt.Errorf("getting target project issue types: %w", err)
	}

	if len(issueTypes) == 0 {
		return fmt.Errorf("target project %s has no issue types", targetProject)
	}

	// Find target issue type
	var targetIssueType *api.IssueType
	if targetType == "" {
		// Get the source issue's type to use as default
		issue, err := client.GetIssue(ctx, issueKeys[0])
		if err != nil {
			return fmt.Errorf("getting source issue: %w", err)
		}

		sourceTypeName := issue.Fields.IssueType.Name
		for i := range issueTypes {
			if strings.EqualFold(issueTypes[i].Name, sourceTypeName) {
				targetIssueType = &issueTypes[i]
				break
			}
		}

		if targetIssueType == nil {
			// Fall back to first non-subtask type
			for i := range issueTypes {
				if !issueTypes[i].Subtask {
					targetIssueType = &issueTypes[i]
					break
				}
			}
		}
	} else {
		// Find by name
		for i := range issueTypes {
			if strings.EqualFold(issueTypes[i].Name, targetType) {
				targetIssueType = &issueTypes[i]
				break
			}
		}
	}

	if targetIssueType == nil {
		var availableTypes []string
		for _, t := range issueTypes {
			if !t.Subtask {
				availableTypes = append(availableTypes, t.Name)
			}
		}
		model := ip.PresentTypeNotFound(targetType, targetProject, availableTypes)
		out := present.Render(model, opts.RenderStyle())
		_, _ = fmt.Fprint(opts.Stdout, out.Stdout)
		_, _ = fmt.Fprint(opts.Stderr, out.Stderr)
		return fmt.Errorf("issue type not found: %s", targetType)
	}

	// Progress message to stderr
	progressModel := ip.PresentMoveProgress(len(issueKeys), targetProject, targetIssueType.Name)
	progressOut := present.Render(progressModel, opts.RenderStyle())
	_, _ = fmt.Fprint(opts.Stderr, progressOut.Stderr)

	// Build and execute the move request
	req := api.BuildMoveRequest(issueKeys, targetProject, targetIssueType.ID, notify)

	resp, err := client.MoveIssues(ctx, req)
	if err != nil {
		// Check if this is a Server/DC instance
		if strings.Contains(err.Error(), "404") || strings.Contains(err.Error(), "not found") {
			return fmt.Errorf("move operation failed - this feature is only available on Jira Cloud")
		}
		return fmt.Errorf("initiating move: %w", err)
	}

	if !wait {
		model := ip.PresentMoveInitiated(resp.TaskID)
		out := present.Render(model, opts.RenderStyle())
		_, _ = fmt.Fprint(opts.Stdout, out.Stdout)
		_, _ = fmt.Fprint(opts.Stderr, out.Stderr)
		return nil
	}

	// Wait for completion - progress to stderr
	waitModel := ip.PresentMoveWaiting()
	waitOut := present.Render(waitModel, opts.RenderStyle())
	_, _ = fmt.Fprint(opts.Stderr, waitOut.Stderr)

	for {
		status, err := client.GetMoveTaskStatus(ctx, resp.TaskID)
		if err != nil {
			return fmt.Errorf("getting task status: %w", err)
		}

		switch status.Status {
		case "COMPLETE":
			if status.Result != nil && len(status.Result.Failed) > 0 {
				model := ip.PresentMovePartialFailure(status.Result.Successful, status.Result.Failed)
				out := present.Render(model, opts.RenderStyle())
				_, _ = fmt.Fprint(opts.Stdout, out.Stdout)
				_, _ = fmt.Fprint(opts.Stderr, out.Stderr)
				return fmt.Errorf("some issues failed to move")
			}
			model := ip.PresentMoved(len(issueKeys), targetProject)
			out := present.Render(model, opts.RenderStyle())
			_, _ = fmt.Fprint(opts.Stdout, out.Stdout)
			return nil

		case "FAILED":
			return fmt.Errorf("move failed")

		case "CANCELLED":
			return fmt.Errorf("move was cancelled")

		case "ENQUEUED", "RUNNING":
			// Still in progress
			time.Sleep(1 * time.Second)

		default:
			return fmt.Errorf("unknown task status: %s", status.Status)
		}
	}
}

func newMoveStatusCmd(opts *root.Options) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "move-status <task-id>",
		Short: "Check status of a move operation",
		Long:  "Check the status of an asynchronous move operation by task ID.",
		Example: `  # Check move task status
  jtk issues move-status abc123`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMoveStatus(cmd.Context(), opts, args[0])
		},
	}

	return cmd
}

func runMoveStatus(ctx context.Context, opts *root.Options, taskID string) error {
	v := opts.View()

	client, err := opts.APIClient()
	if err != nil {
		return err
	}

	status, err := client.GetMoveTaskStatus(ctx, taskID)
	if err != nil {
		return err
	}

	if opts.Output == "json" {
		return v.JSON(status)
	}

	model := jtkpresent.IssuePresenter{}.PresentMoveStatus(status)
	out := present.Render(model, opts.RenderStyle())
	_, _ = fmt.Fprint(opts.Stdout, out.Stdout)
	_, _ = fmt.Fprint(opts.Stderr, out.Stderr)
	return nil
}
