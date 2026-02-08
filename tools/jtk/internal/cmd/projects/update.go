package projects

import (
	"github.com/spf13/cobra"

	"github.com/open-cli-collective/jira-ticket-cli/api"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
)

func newUpdateCmd(opts *root.Options) *cobra.Command {
	var name string
	var description string
	var lead string

	cmd := &cobra.Command{
		Use:   "update <project-key>",
		Short: "Update a project",
		Long:  "Update a Jira project's metadata. Only specified fields are changed.",
		Example: `  # Rename a project
  jtk projects update MYPROJ --name "New Name"

  # Update description
  jtk projects update MYPROJ --description "Updated description"

  # Change project lead
  jtk projects update MYPROJ --lead <account-id>`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUpdate(opts, args[0], name, description, lead)
		},
	}

	cmd.Flags().StringVarP(&name, "name", "n", "", "New project name")
	cmd.Flags().StringVarP(&description, "description", "d", "", "New project description")
	cmd.Flags().StringVarP(&lead, "lead", "l", "", "New lead account ID")

	return cmd
}

func runUpdate(opts *root.Options, keyOrID, name, description, lead string) error {
	v := opts.View()

	client, err := opts.APIClient()
	if err != nil {
		return err
	}

	req := &api.UpdateProjectRequest{
		Name:          name,
		Description:   description,
		LeadAccountID: lead,
	}

	project, err := client.UpdateProject(keyOrID, req)
	if err != nil {
		return err
	}

	if opts.Output == "json" {
		return v.JSON(project)
	}

	v.Success("Updated project %s", project.Key)

	return nil
}
