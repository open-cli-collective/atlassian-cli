package projects

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/atlassian-go/present"
	"github.com/open-cli-collective/atlassian-go/view"

	"github.com/open-cli-collective/jira-ticket-cli/api"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cache"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
	"github.com/open-cli-collective/jira-ticket-cli/internal/mutation"
	jtkpresent "github.com/open-cli-collective/jira-ticket-cli/internal/present"
	"github.com/open-cli-collective/jira-ticket-cli/internal/resolve"
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

  # Change project lead (accepts accountId, email, display name, or "me")
  jtk projects update MYPROJ --lead "Aaron Wong"
  jtk projects update MYPROJ --lead aaron@example.com`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUpdate(cmd.Context(), opts, args[0], name, description, lead)
		},
	}

	cmd.Flags().StringVarP(&name, "name", "n", "", "New project name")
	cmd.Flags().StringVarP(&description, "description", "d", "", "New project description")
	cmd.Flags().StringVarP(&lead, "lead", "l", "", "New lead: accountId, email, display name, or \"me\"")

	return cmd
}

func runUpdate(ctx context.Context, opts *root.Options, keyOrID, name, description, lead string) error {
	v := opts.View()

	client, err := opts.APIClient()
	if err != nil {
		return err
	}

	req := &api.UpdateProjectRequest{
		Name:        name,
		Description: description,
	}

	if lead != "" {
		resolvedLead, err := resolve.New(client).User(ctx, lead)
		if err != nil {
			return err
		}
		req.LeadAccountID = resolvedLead.AccountID
	}

	project, err := client.UpdateProject(ctx, keyOrID, req)
	if err != nil {
		return err
	}

	_ = cache.Touch(cache.ProjectDependents()...)

	if opts.EmitIDOnly() {
		return jtkpresent.EmitIDs(opts, []string{project.Key})
	}

	if v.Format == view.FormatJSON {
		return v.JSON(project)
	}

	var lastFetched *api.ProjectDetail
	return mutation.WriteAndPresent(ctx, opts, mutation.Config{
		Write: func(_ context.Context) (string, error) {
			return project.Key, nil
		},
		Fetch: func(ctx context.Context, id string) (*present.OutputModel, error) {
			fetched, err := client.GetProject(ctx, id, api.ProjectGetExpand)
			if err != nil {
				return nil, err
			}
			lastFetched = fetched
			return jtkpresent.ProjectPresenter{}.PresentProjectDetail(fetched, opts.IsExtended()), nil
		},
		IsFresh: func(_ *present.OutputModel) bool {
			if lastFetched == nil {
				return false
			}
			if name != "" && lastFetched.Name != name {
				return false
			}
			if description != "" && lastFetched.Description != description {
				return false
			}
			if req.LeadAccountID != "" && lastFetched.Lead != nil && lastFetched.Lead.AccountID != req.LeadAccountID {
				return false
			}
			return true
		},
		Fallback: func(id string) *present.OutputModel {
			return jtkpresent.ProjectPresenter{}.PresentUpdated(id)
		},
	})
}
