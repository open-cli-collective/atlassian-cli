// Package projects provides CLI commands for managing Jira projects.
package projects

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/atlassian-go/present"
	"github.com/open-cli-collective/atlassian-go/view"

	"github.com/open-cli-collective/jira-ticket-cli/api"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cache"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
	jtkpresent "github.com/open-cli-collective/jira-ticket-cli/internal/present"
	"github.com/open-cli-collective/jira-ticket-cli/internal/resolve"
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

The --lead flag accepts a cached user reference: accountId, email,
display name, or "me". Use 'jtk users search' for candidates or
'jtk me' to get your own accountId.`,
		Example: `  # Create a software project (--lead resolves via the users cache)
  jtk projects create --key MYPROJ --name "My Project" --lead me
  jtk projects create --key MYPROJ --name "My Project" --lead "Aaron Wong"
  jtk projects create --key MYPROJ --name "My Project" --lead aaron@example.com

  # Create a business project with description
  jtk projects create --key BIZ --name "Business" --type business --lead me --description "Business project"

  # Project types: software (default), service_desk, business
  jtk projects types`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runCreate(cmd.Context(), opts, key, name, projectType, lead, description)
		},
	}

	cmd.Flags().StringVarP(&key, "key", "k", "", "Project key (required)")
	cmd.Flags().StringVarP(&name, "name", "n", "", "Project name (required)")
	cmd.Flags().StringVarP(&projectType, "type", "t", "software", "Project type (software, service_desk, business)")
	cmd.Flags().StringVarP(&lead, "lead", "l", "", "Lead: accountId, email, display name, or \"me\" (required)")
	cmd.Flags().StringVarP(&description, "description", "d", "", "Project description")

	_ = cmd.MarkFlagRequired("key")
	_ = cmd.MarkFlagRequired("name")
	_ = cmd.MarkFlagRequired("lead")

	return cmd
}

func runCreate(ctx context.Context, opts *root.Options, key, name, projectType, lead, description string) error {
	v := opts.View()

	client, err := opts.APIClient()
	if err != nil {
		return err
	}

	resolvedLead, err := resolve.New(client).User(ctx, lead)
	if err != nil {
		return err
	}

	req := &api.CreateProjectRequest{
		Key:            key,
		Name:           name,
		ProjectTypeKey: projectType,
		LeadAccountID:  resolvedLead.AccountID,
		Description:    description,
	}

	project, err := client.CreateProject(ctx, req)
	if err != nil {
		return err
	}

	_ = cache.Touch(cache.ProjectDependents()...)

	if v.Format == view.FormatJSON {
		return v.JSON(project)
	}

	model := jtkpresent.ProjectPresenter{}.PresentCreated(project.Key, name)
	out := present.Render(model, opts.RenderStyle())
	_, _ = fmt.Fprint(opts.Stdout, out.Stdout)
	return nil
}
