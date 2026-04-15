// Package fields provides CLI commands for managing Jira custom fields.
package fields

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/atlassian-go/present"
	"github.com/open-cli-collective/atlassian-go/prompt"
	"github.com/open-cli-collective/atlassian-go/view"

	"github.com/open-cli-collective/jira-ticket-cli/api"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
	jtkpresent "github.com/open-cli-collective/jira-ticket-cli/internal/present"
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
			return runContextsList(cmd.Context(), opts, args[0])
		},
	}
}

func runContextsList(ctx context.Context, opts *root.Options, fieldID string) error {
	v := opts.View()

	client, err := opts.APIClient()
	if err != nil {
		return err
	}

	result, err := client.GetFieldContexts(ctx, fieldID)
	if err != nil {
		return err
	}

	if len(result.Values) == 0 {
		model := jtkpresent.FieldPresenter{}.PresentNoContexts(fieldID)
		out := present.Render(model, opts.RenderStyle())
		fmt.Fprint(opts.Stdout, out.Stdout)
		fmt.Fprint(opts.Stderr, out.Stderr)
		return nil
	}

	if v.Format == view.FormatJSON {
		return v.JSON(result.Values)
	}

	contexts := make([]jtkpresent.FieldContext, len(result.Values))
	for i, ctx := range result.Values {
		contexts[i] = jtkpresent.FieldContext{
			ID:              ctx.ID,
			Name:            ctx.Name,
			IsGlobalContext: ctx.IsGlobalContext,
			IsAnyIssueType:  ctx.IsAnyIssueType,
		}
	}

	model := jtkpresent.FieldPresenter{}.PresentContexts(contexts)
	out := present.Render(model, opts.RenderStyle())
	fmt.Fprint(opts.Stdout, out.Stdout)
	fmt.Fprint(opts.Stderr, out.Stderr)
	return nil
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
			return runContextsCreate(cmd.Context(), opts, args[0], name, project)
		},
	}

	cmd.Flags().StringVarP(&name, "name", "n", "", "Context name (required)")
	cmd.Flags().StringVarP(&project, "project", "p", "", "Project ID to scope the context to")

	_ = cmd.MarkFlagRequired("name")

	return cmd
}

func runContextsCreate(ctx context.Context, opts *root.Options, fieldID, name, project string) error {
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

	fc, err := client.CreateFieldContext(ctx, fieldID, req)
	if err != nil {
		return err
	}

	if v.Format == view.FormatJSON {
		return v.JSON(fc)
	}

	model := jtkpresent.FieldPresenter{}.PresentContextCreated(fc.ID, fc.Name)
	out := present.Render(model, opts.RenderStyle())
	fmt.Fprint(opts.Stdout, out.Stdout)
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
			return runContextsDelete(cmd.Context(), opts, args[0], args[1], force)
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Skip confirmation prompt")

	return cmd
}

func runContextsDelete(ctx context.Context, opts *root.Options, fieldID, contextID string, force bool) error {
	if !force {
		fmt.Fprintf(opts.Stderr, "This will delete context %s from field %s.\n", contextID, fieldID)
		fmt.Fprint(opts.Stderr, "Are you sure? [y/N]: ")

		confirmed, err := prompt.Confirm(opts.Stdin)
		if err != nil {
			return fmt.Errorf("reading confirmation: %w", err)
		}
		if !confirmed {
			model := jtkpresent.FieldPresenter{}.PresentDeleteCancelled()
			out := present.Render(model, opts.RenderStyle())
			fmt.Fprint(opts.Stdout, out.Stdout)
			fmt.Fprint(opts.Stderr, out.Stderr)
			return nil
		}
	}

	client, err := opts.APIClient()
	if err != nil {
		return err
	}

	if err := client.DeleteFieldContext(ctx, fieldID, contextID); err != nil {
		return err
	}

	model := jtkpresent.FieldPresenter{}.PresentContextDeleted(contextID, fieldID)
	out := present.Render(model, opts.RenderStyle())
	fmt.Fprint(opts.Stdout, out.Stdout)
	return nil
}
