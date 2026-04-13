// Package boards provides CLI commands for managing Jira agile boards.
package boards

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/atlassian-go/artifact"
	"github.com/open-cli-collective/atlassian-go/view"

	"github.com/open-cli-collective/jira-ticket-cli/api"
	jtkartifact "github.com/open-cli-collective/jira-ticket-cli/internal/artifact"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
)

// Register registers the boards commands
func Register(parent *cobra.Command, opts *root.Options) {
	cmd := &cobra.Command{
		Use:     "boards",
		Aliases: []string{"board", "b"},
		Short:   "Manage agile boards",
		Long:    "Commands for viewing agile boards.",
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
	cmd.AddCommand(newGetCmd(opts))

	parent.AddCommand(cmd)
}

func newListCmd(opts *root.Options) *cobra.Command {
	var project string
	var maxResults int

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List boards",
		Long:  "List agile boards, optionally filtered by project.",
		Example: `  # List all boards
  jtk boards list

  # List boards for a project
  jtk boards list --project MYPROJECT`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runList(cmd.Context(), opts, project, maxResults)
		},
	}

	cmd.Flags().StringVarP(&project, "project", "p", "", "Filter by project key")
	cmd.Flags().IntVarP(&maxResults, "max", "m", 50, "Maximum number of results")

	return cmd
}

func runList(ctx context.Context, opts *root.Options, project string, maxResults int) error {
	v := opts.View()

	client, err := opts.APIClient()
	if err != nil {
		return err
	}

	result, err := client.ListBoards(ctx, project, 0, maxResults)
	if err != nil {
		return err
	}

	if len(result.Values) == 0 {
		v.Info("No boards found")
		return nil
	}

	if v.Format == view.FormatJSON {
		arts := jtkartifact.ProjectBoards(result.Values, opts.ArtifactMode())
		hasMore := !result.IsLast
		return v.RenderArtifactList(artifact.NewListResult(arts, hasMore))
	}

	headers := []string{"ID", "NAME", "TYPE", "PROJECT"}
	rows := make([][]string, 0, len(result.Values))

	for _, b := range result.Values {
		rows = append(rows, []string{
			fmt.Sprintf("%d", b.ID),
			b.Name,
			b.Type,
			b.Location.ProjectKey,
		})
	}

	return v.Table(headers, rows)
}

func newGetCmd(opts *root.Options) *cobra.Command {
	return &cobra.Command{
		Use:     "get <board-id>",
		Short:   "Get board details",
		Long:    "Get details for a specific board.",
		Example: `  jtk boards get 123`,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var boardID int
			if _, err := fmt.Sscanf(args[0], "%d", &boardID); err != nil {
				return fmt.Errorf("invalid board ID: %s", args[0])
			}
			return runGet(cmd.Context(), opts, boardID)
		},
	}
}

func runGet(ctx context.Context, opts *root.Options, boardID int) error {
	v := opts.View()

	client, err := opts.APIClient()
	if err != nil {
		return err
	}

	board, err := client.GetBoard(ctx, boardID)
	if err != nil {
		return err
	}

	if v.Format == view.FormatJSON {
		return v.RenderArtifact(jtkartifact.ProjectBoard(board, opts.ArtifactMode()))
	}

	v.Println("ID:      %d", board.ID)
	v.Println("Name:    %s", board.Name)
	v.Println("Type:    %s", board.Type)
	v.Println("Project: %s", board.Location.ProjectKey)

	return nil
}
