// Package sprints provides CLI commands for managing Jira sprints.
package sprints

import (
	"context"
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/atlassian-go/artifact"
	"github.com/open-cli-collective/atlassian-go/present"
	"github.com/open-cli-collective/atlassian-go/view"

	"github.com/open-cli-collective/jira-ticket-cli/api"
	jtkartifact "github.com/open-cli-collective/jira-ticket-cli/internal/artifact"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
	jtkpresent "github.com/open-cli-collective/jira-ticket-cli/internal/present"
	"github.com/open-cli-collective/jira-ticket-cli/internal/resolve"
)

// validateBoardRef rejects inputs that would parse as numeric but produce a
// synthetic Board{ID: n} with n <= 0, which the downstream Agile endpoints
// return confusing 404s for. Non-numeric names pass through unchanged —
// board-name resolution is handled by the resolver.
func validateBoardRef(board string) error {
	if board == "" {
		return fmt.Errorf("--board is required")
	}
	if n, err := strconv.Atoi(board); err == nil && n <= 0 {
		return fmt.Errorf("--board numeric ID must be positive (got %s)", board)
	}
	return nil
}

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
	var board string
	var state string
	var maxResults int

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List sprints for a board",
		Long:  "List sprints for a specific board. --board accepts a board ID or name.",
		Example: `  # List all sprints
  jtk sprints list --board 123
  jtk sprints list --board "MON board"

  # List only active sprints
  jtk sprints list --board 123 --state active`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := validateBoardRef(board); err != nil {
				return err
			}
			client, err := opts.APIClient()
			if err != nil {
				return err
			}
			resolvedBoard, err := resolve.New(client).Board(cmd.Context(), board)
			if err != nil {
				return err
			}
			return runList(cmd.Context(), opts, client, resolvedBoard.ID, state, maxResults)
		},
	}

	cmd.Flags().StringVarP(&board, "board", "b", "", "Board ID or name (required)")
	cmd.Flags().StringVarP(&state, "state", "s", "", "Filter by state (active, closed, future)")
	cmd.Flags().IntVarP(&maxResults, "max", "m", 50, "Maximum number of results")

	return cmd
}

func runList(ctx context.Context, opts *root.Options, client *api.Client, boardID int, state string, maxResults int) error {
	v := opts.View()

	result, err := client.ListSprints(ctx, boardID, state, 0, maxResults)
	if err != nil {
		return err
	}

	if len(result.Values) == 0 {
		model := jtkpresent.SprintPresenter{}.PresentEmpty()
		out := present.Render(model, opts.RenderStyle())
		_, _ = fmt.Fprint(opts.Stdout, out.Stdout)
		return nil
	}

	if v.Format == view.FormatJSON {
		arts := jtkartifact.ProjectSprints(result.Values, opts.ArtifactMode())
		hasMore := !result.IsLast
		return v.RenderArtifactList(artifact.NewListResult(arts, hasMore))
	}

	// Text path: presenter → render → write
	model := jtkpresent.SprintPresenter{}.PresentList(result.Values)
	out := present.Render(model, opts.RenderStyle())
	_, _ = fmt.Fprint(opts.Stdout, out.Stdout)
	_, _ = fmt.Fprint(opts.Stderr, out.Stderr)
	return nil
}

func newCurrentCmd(opts *root.Options) *cobra.Command {
	var board string

	cmd := &cobra.Command{
		Use:   "current",
		Short: "Show current sprint",
		Long:  "Show the current active sprint for a board. --board accepts a board ID or name.",
		Example: `  jtk sprints current --board 123
  jtk sprints current --board "MON board"`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := validateBoardRef(board); err != nil {
				return err
			}
			client, err := opts.APIClient()
			if err != nil {
				return err
			}
			resolvedBoard, err := resolve.New(client).Board(cmd.Context(), board)
			if err != nil {
				return err
			}
			return runCurrent(cmd.Context(), opts, client, resolvedBoard.ID)
		},
	}

	cmd.Flags().StringVarP(&board, "board", "b", "", "Board ID or name (required)")

	return cmd
}

func runCurrent(ctx context.Context, opts *root.Options, client *api.Client, boardID int) error {
	v := opts.View()

	sprint, err := client.GetCurrentSprint(ctx, boardID)
	if err != nil {
		return err
	}

	if v.Format == view.FormatJSON {
		return v.RenderArtifact(jtkartifact.ProjectSprint(sprint, opts.ArtifactMode()))
	}

	// Text path: presenter → render → write
	model := jtkpresent.SprintPresenter{}.PresentDetail(sprint)
	out := present.Render(model, opts.RenderStyle())
	_, _ = fmt.Fprint(opts.Stdout, out.Stdout)
	_, _ = fmt.Fprint(opts.Stderr, out.Stderr)
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
		model := jtkpresent.SprintPresenter{}.PresentNoIssues()
		out := present.Render(model, opts.RenderStyle())
		_, _ = fmt.Fprint(opts.Stdout, out.Stdout)
		return nil
	}

	if v.Format == view.FormatJSON {
		arts := jtkartifact.ProjectIssues(result.Issues, opts.ArtifactMode())
		// Guard against Total < 0 (Jira returns -1 when count is unknown).
		// When Total is unknown, assume more results if we got a full page.
		var hasMore bool
		if result.Total < 0 {
			hasMore = len(result.Issues) == maxResults
		} else {
			hasMore = result.StartAt+len(result.Issues) < result.Total
		}
		return v.RenderArtifactList(artifact.NewListResult(arts, hasMore))
	}

	// Text path: presenter → render → write
	model := jtkpresent.IssuePresenter{}.PresentList(result.Issues)
	out := present.Render(model, opts.RenderStyle())
	_, _ = fmt.Fprint(opts.Stdout, out.Stdout)
	_, _ = fmt.Fprint(opts.Stderr, out.Stderr)
	return nil
}

func newAddCmd(opts *root.Options) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add <sprint> <issue-key>...",
		Short: "Move issues to a sprint",
		Long:  "Move one or more issues to a specific sprint. <sprint> accepts a sprint ID or name.",
		Example: `  # Move a single issue by sprint ID
  jtk sprints add 123 PROJ-456

  # Move by sprint name (resolved via cache)
  jtk sprints add "MON Sprint 70" PROJ-456

  # Move multiple issues
  jtk sprints add 123 PROJ-456 PROJ-789 PROJ-101`,
		Args: cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := opts.APIClient()
			if err != nil {
				return err
			}
			resolvedSprint, err := resolve.New(client).Sprint(cmd.Context(), args[0], 0)
			if err != nil {
				return err
			}
			return runAdd(cmd.Context(), opts, client, resolvedSprint.ID, args[1:])
		},
	}

	return cmd
}

func runAdd(ctx context.Context, opts *root.Options, client *api.Client, sprintID int, issueKeys []string) error {
	if err := client.MoveIssuesToSprint(ctx, sprintID, issueKeys); err != nil {
		return err
	}

	model := jtkpresent.SprintPresenter{}.PresentMoved(issueKeys, sprintID)
	out := present.Render(model, opts.RenderStyle())
	_, _ = fmt.Fprint(opts.Stdout, out.Stdout)
	_, _ = fmt.Fprint(opts.Stderr, out.Stderr)
	return nil
}
