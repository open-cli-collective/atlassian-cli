package fields

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/atlassian-go/prompt"

	"github.com/open-cli-collective/jira-ticket-cli/api"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
)

func newContextsCmd(opts *root.Options) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "contexts",
		Aliases: []string{"context", "ctx"},
		Short:   "Manage field contexts",
		Long:    "Commands for listing, creating, and deleting custom field contexts.",
	}

	cmd.AddCommand(newContextsListCmd(opts))
	cmd.AddCommand(newContextsCreateCmd(opts))
	cmd.AddCommand(newContextsDeleteCmd(opts))

	return cmd
}

func newContextsListCmd(opts *root.Options) *cobra.Command {
	return &cobra.Command{
		Use:   "list <field-id>",
		Short: "List contexts for a field",
		Example: `  # List contexts for a custom field
  jtk fields contexts list customfield_10100`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runContextsList(opts, args[0])
		},
	}
}

func runContextsList(opts *root.Options, fieldID string) error {
	v := opts.View()

	client, err := opts.APIClient()
	if err != nil {
		return err
	}

	result, err := client.GetFieldContexts(fieldID)
	if err != nil {
		return err
	}

	if len(result.Values) == 0 {
		v.Info("No contexts found for field %s", fieldID)
		return nil
	}

	if opts.Output == "json" {
		return v.JSON(result.Values)
	}

	headers := []string{"ID", "NAME", "GLOBAL", "ANY_ISSUE_TYPE"}
	var rows [][]string

	for _, ctx := range result.Values {
		global := "no"
		if ctx.IsGlobalContext {
			global = "yes"
		}
		anyIssueType := "no"
		if ctx.IsAnyIssueType {
			anyIssueType = "yes"
		}
		rows = append(rows, []string{ctx.ID, ctx.Name, global, anyIssueType})
	}

	return v.Table(headers, rows)
}

func newContextsCreateCmd(opts *root.Options) *cobra.Command {
	var name, project string

	cmd := &cobra.Command{
		Use:   "create <field-id>",
		Short: "Create a field context",
		Example: `  # Create a context for a field
  jtk fields contexts create customfield_10100 --name "Bug Context"

  # Create a context scoped to a project
  jtk fields contexts create customfield_10100 --name "Project Context" --project 10001`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runContextsCreate(opts, args[0], name, project)
		},
	}

	cmd.Flags().StringVarP(&name, "name", "n", "", "Context name (required)")
	cmd.Flags().StringVarP(&project, "project", "p", "", "Project ID to scope the context to")

	_ = cmd.MarkFlagRequired("name")

	return cmd
}

func runContextsCreate(opts *root.Options, fieldID, name, project string) error {
	v := opts.View()

	client, err := opts.APIClient()
	if err != nil {
		return err
	}

	req := &api.CreateFieldContextRequest{
		Name: name,
	}
	if project != "" {
		req.ProjectIDs = []string{project}
	}

	ctx, err := client.CreateFieldContext(fieldID, req)
	if err != nil {
		return err
	}

	if opts.Output == "json" {
		return v.JSON(ctx)
	}

	v.Success("Created context %s (%s)", ctx.ID, ctx.Name)
	return nil
}

func newContextsDeleteCmd(opts *root.Options) *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "delete <field-id> <context-id>",
		Short: "Delete a field context",
		Example: `  # Delete a context (will prompt for confirmation)
  jtk fields contexts delete customfield_10100 10003

  # Delete without confirmation
  jtk fields contexts delete customfield_10100 10003 --force`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runContextsDelete(opts, args[0], args[1], force)
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Skip confirmation prompt")

	return cmd
}

func runContextsDelete(opts *root.Options, fieldID, contextID string, force bool) error {
	v := opts.View()

	if !force {
		fmt.Printf("This will delete context %s from field %s.\n", contextID, fieldID)
		fmt.Print("Are you sure? [y/N]: ")

		confirmed, err := prompt.Confirm(opts.Stdin)
		if err != nil {
			return fmt.Errorf("failed to read confirmation: %w", err)
		}
		if !confirmed {
			v.Info("Deletion cancelled.")
			return nil
		}
	}

	client, err := opts.APIClient()
	if err != nil {
		return err
	}

	if err := client.DeleteFieldContext(fieldID, contextID); err != nil {
		return err
	}

	v.Success("Deleted context %s from field %s", contextID, fieldID)
	return nil
}
