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

func newOptionsCmd(opts *root.Options) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "options",
		Aliases: []string{"option", "opt"},
		Short:   "Manage field option values",
		Long: `Commands for managing option values of select and multiselect custom fields.

When --context is omitted, the default (first) context is used automatically.`,
	}

	cmd.AddCommand(newOptionsListCmd(opts))
	cmd.AddCommand(newOptionsAddCmd(opts))
	cmd.AddCommand(newOptionsUpdateCmd(opts))
	cmd.AddCommand(newOptionsDeleteCmd(opts))

	return cmd
}

// resolveContextID returns the provided context ID, or auto-detects the default context.
func resolveContextID(ctx context.Context, client *api.Client, fieldID, contextFlag string) (string, error) {
	if contextFlag != "" {
		return contextFlag, nil
	}
	fc, err := client.GetDefaultFieldContext(ctx, fieldID)
	if err != nil {
		return "", fmt.Errorf("could not auto-detect context (use --context to specify): %w", err)
	}
	return fc.ID, nil
}

func newOptionsListCmd(opts *root.Options) *cobra.Command {
	var contextID string

	cmd := &cobra.Command{
		Use:   "list <field-id>",
		Short: "List options for a field",
		Long:  "List option values for a select or multiselect custom field. Auto-detects the default context if --context is not specified.",
		Example: `  # List options (auto-detects context)
  jtk fields options list customfield_10100

  # List options for a specific context
  jtk fields options list customfield_10100 --context 10001`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runOptionsList(cmd.Context(), opts, args[0], contextID)
		},
	}

	cmd.Flags().StringVarP(&contextID, "context", "c", "", "Context ID (auto-detected if omitted)")

	return cmd
}

func runOptionsList(ctx context.Context, opts *root.Options, fieldID, contextFlag string) error {
	v := opts.View()

	client, err := opts.APIClient()
	if err != nil {
		return err
	}

	ctxID, err := resolveContextID(ctx, client, fieldID, contextFlag)
	if err != nil {
		return err
	}

	result, err := client.GetFieldContextOptions(ctx, fieldID, ctxID)
	if err != nil {
		return err
	}

	if len(result.Values) == 0 {
		model := jtkpresent.FieldPresenter{}.PresentNoOptions(fieldID)
		out := present.Render(model, opts.RenderStyle())
		fmt.Fprint(opts.Stdout, out.Stdout)
		fmt.Fprint(opts.Stderr, out.Stderr)
		return nil
	}

	if v.Format == view.FormatJSON {
		return v.JSON(result.Values)
	}

	options := make([]jtkpresent.FieldContextOption, len(result.Values))
	for i, opt := range result.Values {
		options[i] = jtkpresent.FieldContextOption{
			ID:       opt.ID,
			Value:    opt.Value,
			Disabled: opt.Disabled,
		}
	}

	model := jtkpresent.FieldPresenter{}.PresentContextOptions(options)
	out := present.Render(model, opts.RenderStyle())
	fmt.Fprint(opts.Stdout, out.Stdout)
	fmt.Fprint(opts.Stderr, out.Stderr)
	return nil
}

func newOptionsAddCmd(opts *root.Options) *cobra.Command {
	var value, contextID string

	cmd := &cobra.Command{
		Use:   "add <field-id>",
		Short: "Add an option to a field",
		Long:  "Add a new option value to a select or multiselect custom field. Auto-detects the default context if --context is not specified.",
		Example: `  # Add an option (auto-detects context)
  jtk fields options add customfield_10100 --value "Production"

  # Add to a specific context
  jtk fields options add customfield_10100 --value "Staging" --context 10001`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runOptionsAdd(cmd.Context(), opts, args[0], value, contextID)
		},
	}

	cmd.Flags().StringVarP(&value, "value", "V", "", "Option value (required)")
	cmd.Flags().StringVarP(&contextID, "context", "c", "", "Context ID (auto-detected if omitted)")

	_ = cmd.MarkFlagRequired("value")

	return cmd
}

func runOptionsAdd(ctx context.Context, opts *root.Options, fieldID, value, contextFlag string) error {
	v := opts.View()

	client, err := opts.APIClient()
	if err != nil {
		return err
	}

	ctxID, err := resolveContextID(ctx, client, fieldID, contextFlag)
	if err != nil {
		return err
	}

	options, err := client.CreateFieldContextOptions(ctx, fieldID, ctxID, &api.CreateFieldContextOptionsRequest{
		Options: []api.CreateFieldContextOptionEntry{
			{Value: value},
		},
	})
	if err != nil {
		return err
	}

	if v.Format == view.FormatJSON {
		return v.JSON(options)
	}

	optionID := ""
	optionValue := value
	if len(options) > 0 {
		optionID = options[0].ID
		optionValue = options[0].Value
	}
	model := jtkpresent.FieldPresenter{}.PresentOptionAdded(optionID, optionValue)
	out := present.Render(model, opts.RenderStyle())
	fmt.Fprint(opts.Stdout, out.Stdout)
	return nil
}

func newOptionsUpdateCmd(opts *root.Options) *cobra.Command {
	var optionID, value, contextID string

	cmd := &cobra.Command{
		Use:   "update <field-id>",
		Short: "Update a field option",
		Long:  "Update an existing option value in a select or multiselect custom field. Auto-detects the default context if --context is not specified.",
		Example: `  # Update an option value
  jtk fields options update customfield_10100 --option 10001 --value "Production (updated)"`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runOptionsUpdate(cmd.Context(), opts, args[0], optionID, value, contextID)
		},
	}

	cmd.Flags().StringVar(&optionID, "option", "", "Option ID to update (required)")
	cmd.Flags().StringVarP(&value, "value", "V", "", "New option value (required)")
	cmd.Flags().StringVarP(&contextID, "context", "c", "", "Context ID (auto-detected if omitted)")

	_ = cmd.MarkFlagRequired("option")
	_ = cmd.MarkFlagRequired("value")

	return cmd
}

func runOptionsUpdate(ctx context.Context, opts *root.Options, fieldID, optionID, value, contextFlag string) error {
	v := opts.View()

	client, err := opts.APIClient()
	if err != nil {
		return err
	}

	ctxID, err := resolveContextID(ctx, client, fieldID, contextFlag)
	if err != nil {
		return err
	}

	options, err := client.UpdateFieldContextOptions(ctx, fieldID, ctxID, &api.UpdateFieldContextOptionsRequest{
		Options: []api.UpdateFieldContextOptionEntry{
			{ID: optionID, Value: value},
		},
	})
	if err != nil {
		return err
	}

	if v.Format == view.FormatJSON {
		return v.JSON(options)
	}

	model := jtkpresent.FieldPresenter{}.PresentOptionUpdated(optionID)
	out := present.Render(model, opts.RenderStyle())
	fmt.Fprint(opts.Stdout, out.Stdout)
	return nil
}

func newOptionsDeleteCmd(opts *root.Options) *cobra.Command {
	var optionID, contextID string
	var force bool

	cmd := &cobra.Command{
		Use:   "delete <field-id>",
		Short: "Delete a field option",
		Long:  "Delete an option value from a select or multiselect custom field. Auto-detects the default context if --context is not specified.",
		Example: `  # Delete an option (will prompt for confirmation)
  jtk fields options delete customfield_10100 --option 10001

  # Delete without confirmation
  jtk fields options delete customfield_10100 --option 10001 --force`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runOptionsDelete(cmd.Context(), opts, args[0], optionID, contextID, force)
		},
	}

	cmd.Flags().StringVar(&optionID, "option", "", "Option ID to delete (required)")
	cmd.Flags().StringVarP(&contextID, "context", "c", "", "Context ID (auto-detected if omitted)")
	cmd.Flags().BoolVar(&force, "force", false, "Skip confirmation prompt")

	_ = cmd.MarkFlagRequired("option")

	return cmd
}

func runOptionsDelete(ctx context.Context, opts *root.Options, fieldID, optionID, contextFlag string, force bool) error {
	if !force {
		fmt.Fprintf(opts.Stderr, "This will delete option %s from field %s.\n", optionID, fieldID)
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

	ctxID, err := resolveContextID(ctx, client, fieldID, contextFlag)
	if err != nil {
		return err
	}

	if err := client.DeleteFieldContextOption(ctx, fieldID, ctxID, optionID); err != nil {
		return err
	}

	model := jtkpresent.FieldPresenter{}.PresentOptionDeleted(optionID, fieldID)
	out := present.Render(model, opts.RenderStyle())
	fmt.Fprint(opts.Stdout, out.Stdout)
	return nil
}
