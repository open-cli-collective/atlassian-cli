// Package sprints provides CLI commands for managing Jira sprints.
package sprints

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/jira-ticket-cli/api"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
)

// Register registers the sprints commands
func Register(parent *cobra.Command, opts *root.Options) {
	cmd := &cobra.Command{
		Use:     "sprints",
		Aliases: []string{"sprint", "sp"},
		Short:   "Manage sprints",
		Long:    "Commands for viewing sprints and sprint issues.",
		// SupportsAgile checks AgileURL — the correct guard for Agile API commands.
		// Non-Agile scope-restricted commands (automation, dashboards) use IsBearerAuth() instead.
		PersistentPreRunE: func(_ *cobra.Command, _ []string) error {
			client, err := opts.APIClient()
			if err != nil {
				return err
			}
			if !client.SupportsAgile() {
				return api.ErrAgileUnavailable
			}
			return nil
		},
	}

	cmd.AddCommand(newListCmd(opts))
	cmd.AddCommand(newCurrentCmd(opts))
	cmd.AddCommand(newIssuesCmd(opts))
	cmd.AddCommand(newAddCmd(opts))

	parent.AddCommand(cmd)
}

func newListCmd(opts *root.Options) *cobra.Command {
	var boardID int
	var state string
	var maxResults int

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List sprints for a board",
		Long:  "List sprints for a specific board.",
		Example: `  # List all sprints
  jtk sprints list --board 123

  # List only active sprints
  jtk sprints list --board 123 --state active`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if boardID == 0 {
				return fmt.Errorf("--board is required")
			}
			return runList(cmd.Context(), opts, boardID, state, maxResults)
		},
	}

	cmd.Flags().IntVarP(&boardID, "board", "b", 0, "Board ID (required)")
	cmd.Flags().StringVarP(&state, "state", "s", "", "Filter by state (active, closed, future)")
	cmd.Flags().IntVarP(&maxResults, "max", "m", 50, "Maximum number of results")

	return cmd
}

func runList(ctx context.Context, opts *root.Options, boardID int, state string, maxResults int) error {
	v := opts.View()

	client, err := opts.APIClient()
	if err != nil {
		return err
	}

	result, err := client.ListSprints(ctx, boardID, state, 0, maxResults)
	if err != nil {
		return err
	}

	if len(result.Values) == 0 {
		v.Info("No sprints found")
		return nil
	}

	if opts.Output == "json" {
		return v.JSON(result.Values)
	}

	headers := []string{"ID", "NAME", "STATE", "START", "END"}
	rows := make([][]string, 0, len(result.Values))

	for _, s := range result.Values {
		startDate := ""
		if s.StartDate != nil {
			startDate = s.StartDate.Format("2006-01-02")
		}
		endDate := ""
		if s.EndDate != nil {
			endDate = s.EndDate.Format("2006-01-02")
		}

		rows = append(rows, []string{
			fmt.Sprintf("%d", s.ID),
			s.Name,
			s.State,
			startDate,
			endDate,
		})
	}

	return v.Table(headers, rows)
}

func newCurrentCmd(opts *root.Options) *cobra.Command {
	var boardID int

	cmd := &cobra.Command{
		Use:     "current",
		Short:   "Show current sprint",
		Long:    "Show the current active sprint for a board.",
		Example: `  jtk sprints current --board 123`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if boardID == 0 {
				return fmt.Errorf("--board is required")
			}
			return runCurrent(cmd.Context(), opts, boardID)
		},
	}

	cmd.Flags().IntVarP(&boardID, "board", "b", 0, "Board ID (required)")

	return cmd
}

func runCurrent(ctx context.Context, opts *root.Options, boardID int) error {
	v := opts.View()

	client, err := opts.APIClient()
	if err != nil {
		return err
	}

	sprint, err := client.GetCurrentSprint(ctx, boardID)
	if err != nil {
		return err
	}

	if opts.Output == "json" {
		return v.JSON(sprint)
	}

	v.Println("ID:    %d", sprint.ID)
	v.Println("Name:  %s", sprint.Name)
	v.Println("State: %s", sprint.State)
	if sprint.StartDate != nil {
		v.Println("Start: %s", sprint.StartDate.Format("2006-01-02"))
	}
	if sprint.EndDate != nil {
		v.Println("End:   %s", sprint.EndDate.Format("2006-01-02"))
	}
	if sprint.Goal != "" {
		v.Println("Goal:  %s", sprint.Goal)
	}

	return nil
}

func newIssuesCmd(opts *root.Options) *cobra.Command {
	var maxResults int

	cmd := &cobra.Command{
		Use:     "issues <sprint-id>",
		Short:   "List issues in a sprint",
		Long:    "List all issues in a specific sprint.",
		Example: `  jtk sprints issues 456`,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var sprintID int
			if _, err := fmt.Sscanf(args[0], "%d", &sprintID); err != nil {
				return fmt.Errorf("invalid sprint ID: %s", args[0])
			}
			return runIssues(cmd.Context(), opts, sprintID, maxResults)
		},
	}

	cmd.Flags().IntVarP(&maxResults, "max", "m", 50, "Maximum number of results")

	return cmd
}

func runIssues(ctx context.Context, opts *root.Options, sprintID int, maxResults int) error {
	v := opts.View()

	client, err := opts.APIClient()
	if err != nil {
		return err
	}

	result, err := client.GetSprintIssues(ctx, sprintID, 0, maxResults)
	if err != nil {
		return err
	}

	if len(result.Issues) == 0 {
		v.Info("No issues in sprint")
		return nil
	}

	if opts.Output == "json" {
		return v.JSON(result.Issues)
	}

	headers := []string{"KEY", "SUMMARY", "STATUS", "ASSIGNEE", "TYPE"}
	rows := make([][]string, 0, len(result.Issues))

	for _, issue := range result.Issues {
		status := ""
		if issue.Fields.Status != nil {
			status = issue.Fields.Status.Name
		}

		assignee := ""
		if issue.Fields.Assignee != nil {
			assignee = issue.Fields.Assignee.DisplayName
		}

		issueType := ""
		if issue.Fields.IssueType != nil {
			issueType = issue.Fields.IssueType.Name
		}

		summary := issue.Fields.Summary
		if len(summary) > 50 {
			summary = summary[:50] + "..."
		}

		rows = append(rows, []string{
			issue.Key,
			summary,
			status,
			assignee,
			issueType,
		})
	}

	return v.Table(headers, rows)
}

func newAddCmd(opts *root.Options) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add <sprint-id> <issue-key>...",
		Short: "Move issues to a sprint",
		Long:  "Move one or more issues to a specific sprint.",
		Example: `  # Move a single issue
  jtk sprints add 123 PROJ-456

  # Move multiple issues
  jtk sprints add 123 PROJ-456 PROJ-789 PROJ-101`,
		Args: cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			var sprintID int
			if _, err := fmt.Sscanf(args[0], "%d", &sprintID); err != nil {
				return fmt.Errorf("invalid sprint ID: %s", args[0])
			}
			return runAdd(cmd.Context(), opts, sprintID, args[1:])
		},
	}

	return cmd
}

func runAdd(ctx context.Context, opts *root.Options, sprintID int, issueKeys []string) error {
	v := opts.View()

	client, err := opts.APIClient()
	if err != nil {
		return err
	}

	if err := client.MoveIssuesToSprint(ctx, sprintID, issueKeys); err != nil {
		return err
	}

	if len(issueKeys) == 1 {
		v.Success("Moved %s to sprint %d", issueKeys[0], sprintID)
	} else {
		v.Success("Moved %d issues to sprint %d", len(issueKeys), sprintID)
	}

	return nil
}
