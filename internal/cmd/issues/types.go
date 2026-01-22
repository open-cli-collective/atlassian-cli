package issues

import (
	"github.com/spf13/cobra"

	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
)

func newTypesCmd(opts *root.Options) *cobra.Command {
	var project string

	cmd := &cobra.Command{
		Use:   "types",
		Short: "List valid issue types for a project",
		Long:  "List all valid issue types that can be used when creating issues in a specific project.",
		Example: `  # List issue types for a project
  jira-ticket-cli issues types --project MYPROJ

  # Using short flag
  jira-ticket-cli issues types -p MYPROJ`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTypes(opts, project)
		},
	}

	cmd.Flags().StringVarP(&project, "project", "p", "", "Project key (required)")
	_ = cmd.MarkFlagRequired("project")

	return cmd
}

func runTypes(opts *root.Options, project string) error {
	v := opts.View()

	client, err := opts.APIClient()
	if err != nil {
		return err
	}

	projectDetail, err := client.GetProject(project)
	if err != nil {
		return err
	}

	if opts.Output == "json" {
		return v.JSON(projectDetail.IssueTypes)
	}

	if len(projectDetail.IssueTypes) == 0 {
		v.Info("No issue types found for project %s", project)
		return nil
	}

	headers := []string{"ID", "NAME", "SUBTASK", "DESCRIPTION"}
	var rows [][]string

	for _, t := range projectDetail.IssueTypes {
		subtask := "no"
		if t.Subtask {
			subtask = "yes"
		}
		rows = append(rows, []string{t.ID, t.Name, subtask, truncate(t.Description, 60)})
	}

	return v.Table(headers, rows)
}
