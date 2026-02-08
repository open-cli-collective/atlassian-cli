package projects

import (
	"github.com/spf13/cobra"

	"github.com/open-cli-collective/jira-ticket-cli/api"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
)

func newCreateCmd(opts *root.Options) *cobra.Command {
	var key string
	var name string
	var projectType string
	var lead string
	var description string

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new project",
		Long: `Create a new Jira project.

The lead account ID is required and must be a valid Jira account ID.
Use 'jtk users search' to find account IDs, or 'jtk me' to get your own.`,
		Example: `  # Create a software project
  jtk projects create --key MYPROJ --name "My Project" --lead <account-id>

  # Create a business project with description
  jtk projects create --key BIZ --name "Business" --type business --lead <account-id> --description "Business project"

  # Project types: software (default), service_desk, business
  jtk projects types`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCreate(opts, key, name, projectType, lead, description)
		},
	}

	cmd.Flags().StringVarP(&key, "key", "k", "", "Project key (required)")
	cmd.Flags().StringVarP(&name, "name", "n", "", "Project name (required)")
	cmd.Flags().StringVarP(&projectType, "type", "t", "software", "Project type (software, service_desk, business)")
	cmd.Flags().StringVarP(&lead, "lead", "l", "", "Lead account ID (required)")
	cmd.Flags().StringVarP(&description, "description", "d", "", "Project description")

	_ = cmd.MarkFlagRequired("key")
	_ = cmd.MarkFlagRequired("name")
	_ = cmd.MarkFlagRequired("lead")

	return cmd
}

func runCreate(opts *root.Options, key, name, projectType, lead, description string) error {
	v := opts.View()

	client, err := opts.APIClient()
	if err != nil {
		return err
	}

	req := &api.CreateProjectRequest{
		Key:            key,
		Name:           name,
		ProjectTypeKey: projectType,
		LeadAccountID:  lead,
		Description:    description,
	}

	project, err := client.CreateProject(req)
	if err != nil {
		return err
	}

	if opts.Output == "json" {
		return v.JSON(project)
	}

	v.Success("Created project %s (%s)", project.Key, project.Name)

	return nil
}
