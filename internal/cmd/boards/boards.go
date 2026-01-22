package boards

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
)

// Register registers the boards commands
func Register(parent *cobra.Command, opts *root.Options) {
	cmd := &cobra.Command{
		Use:     "boards",
		Aliases: []string{"board", "b"},
		Short:   "Manage agile boards",
		Long:    "Commands for viewing agile boards.",
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
  jira-ticket-cli boards list

  # List boards for a project
  jira-ticket-cli boards list --project MYPROJECT`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runList(opts, project, maxResults)
		},
	}

	cmd.Flags().StringVarP(&project, "project", "p", "", "Filter by project key")
	cmd.Flags().IntVarP(&maxResults, "max", "m", 50, "Maximum number of results")

	return cmd
}

func runList(opts *root.Options, project string, maxResults int) error {
	v := opts.View()

	client, err := opts.APIClient()
	if err != nil {
		return err
	}

	result, err := client.ListBoards(project, 0, maxResults)
	if err != nil {
		return err
	}

	if len(result.Values) == 0 {
		v.Info("No boards found")
		return nil
	}

	if opts.Output == "json" {
		return v.JSON(result.Values)
	}

	headers := []string{"ID", "NAME", "TYPE", "PROJECT"}
	var rows [][]string

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
		Example: `  jira-ticket-cli boards get 123`,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var boardID int
			if _, err := fmt.Sscanf(args[0], "%d", &boardID); err != nil {
				return fmt.Errorf("invalid board ID: %s", args[0])
			}
			return runGet(opts, boardID)
		},
	}
}

func runGet(opts *root.Options, boardID int) error {
	v := opts.View()

	client, err := opts.APIClient()
	if err != nil {
		return err
	}

	board, err := client.GetBoard(boardID)
	if err != nil {
		return err
	}

	if opts.Output == "json" {
		return v.JSON(board)
	}

	v.Println("ID:      %d", board.ID)
	v.Println("Name:    %s", board.Name)
	v.Println("Type:    %s", board.Type)
	v.Println("Project: %s", board.Location.ProjectKey)

	return nil
}
