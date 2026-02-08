package projects

import (
	"github.com/spf13/cobra"

	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
)

func newListCmd(opts *root.Options) *cobra.Command {
	var query string
	var maxResults int

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List projects",
		Long:  "List Jira projects, optionally filtered by a search query.",
		Example: `  # List all projects
  jtk projects list

  # Search projects by name
  jtk projects list --query "my project"

  # Limit results
  jtk projects list --max 10`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runList(opts, query, maxResults)
		},
	}

	cmd.Flags().StringVarP(&query, "query", "q", "", "Filter projects by name")
	cmd.Flags().IntVarP(&maxResults, "max", "m", 50, "Maximum number of results")

	return cmd
}

func runList(opts *root.Options, query string, maxResults int) error {
	v := opts.View()

	client, err := opts.APIClient()
	if err != nil {
		return err
	}

	result, err := client.SearchProjects(query, 0, maxResults)
	if err != nil {
		return err
	}

	if len(result.Values) == 0 {
		v.Info("No projects found")
		return nil
	}

	if opts.Output == "json" {
		return v.JSON(result.Values)
	}

	headers := []string{"KEY", "NAME", "TYPE", "LEAD"}
	var rows [][]string

	for _, p := range result.Values {
		lead := ""
		if p.Lead != nil {
			lead = p.Lead.DisplayName
		}
		rows = append(rows, []string{
			p.Key,
			p.Name,
			p.ProjectTypeKey,
			lead,
		})
	}

	return v.Table(headers, rows)
}
