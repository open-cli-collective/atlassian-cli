package projects

import (
	"context"
	"strings"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
)

func newGetCmd(opts *root.Options) *cobra.Command {
	return &cobra.Command{
		Use:   "get <project-key>",
		Short: "Get project details",
		Long:  "Get details for a specific project by key or ID.",
		Example: `  jtk projects get MYPROJECT
  jtk projects get 10001`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGet(cmd.Context(), opts, args[0])
		},
	}
}

func runGet(ctx context.Context, opts *root.Options, keyOrID string) error {
	v := opts.View()

	client, err := opts.APIClient()
	if err != nil {
		return err
	}

	project, err := client.GetProject(ctx, keyOrID)
	if err != nil {
		return err
	}

	if opts.Output == "json" {
		return v.JSON(project)
	}

	v.Println("Key:         %s", project.Key)
	v.Println("Name:        %s", project.Name)
	v.Println("ID:          %s", project.ID.String())
	v.Println("Type:        %s", project.ProjectTypeKey)

	if project.Lead != nil {
		v.Println("Lead:        %s", project.Lead.DisplayName)
	}

	if project.Description != "" {
		v.Println("Description: %s", project.Description)
	}

	if len(project.IssueTypes) > 0 {
		var names []string
		for _, it := range project.IssueTypes {
			names = append(names, it.Name)
		}
		v.Println("Issue Types: %s", strings.Join(names, ", "))
	}

	if project.URL != "" {
		v.Println("URL:         %s", project.URL)
	}

	return nil
}
