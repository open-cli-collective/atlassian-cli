package fields

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/atlassian-go/prompt"

	"github.com/open-cli-collective/jira-ticket-cli/api"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
)

// Register registers the fields commands
func Register(parent *cobra.Command, opts *root.Options) {
	cmd := &cobra.Command{
		Use:     "fields",
		Aliases: []string{"field", "f"},
		Short:   "Manage Jira custom fields",
		Long:    "Commands for managing custom field definitions, contexts, and options.",
	}

	cmd.AddCommand(newListCmd(opts))
	cmd.AddCommand(newCreateCmd(opts))
	cmd.AddCommand(newDeleteCmd(opts))
	cmd.AddCommand(newRestoreCmd(opts))
	cmd.AddCommand(newContextsCmd(opts))
	cmd.AddCommand(newOptionsCmd(opts))

	parent.AddCommand(cmd)
}

func newListCmd(opts *root.Options) *cobra.Command {
	var customOnly bool

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List field definitions",
		Long:  "List all fields or only custom fields. Shows field ID, name, type, and whether it is custom.",
		Example: `  # List all fields
  jtk fields list

  # List only custom fields
  jtk fields list --custom

  # List fields as JSON
  jtk fields list -o json`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runList(cmd.Context(), opts, customOnly)
		},
	}

	cmd.Flags().BoolVar(&customOnly, "custom", false, "Show only custom fields")

	return cmd
}

func runList(ctx context.Context, opts *root.Options, customOnly bool) error {
	v := opts.View()

	client, err := opts.APIClient()
	if err != nil {
		return err
	}

	var fields []api.Field
	if customOnly {
		fields, err = client.GetCustomFields(ctx)
	} else {
		fields, err = client.GetFields(ctx)
	}
	if err != nil {
		return err
	}

	if len(fields) == 0 {
		v.Info("No fields found")
		return nil
	}

	if opts.Output == "json" {
		return v.JSON(fields)
	}

	headers := []string{"ID", "NAME", "TYPE", "CUSTOM"}
	var rows [][]string

	for _, f := range fields {
		custom := "no"
		if f.Custom {
			custom = "yes"
		}
		rows = append(rows, []string{f.ID, f.Name, f.Schema.Type, custom})
	}

	return v.Table(headers, rows)
}

func newCreateCmd(opts *root.Options) *cobra.Command {
	var name, fieldType, description string

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a custom field",
		Long: `Create a new custom field in Jira.

Common field types:
  com.atlassian.jira.plugin.system.customfieldtypes:textfield     (single-line text)
  com.atlassian.jira.plugin.system.customfieldtypes:textarea      (multi-line text)
  com.atlassian.jira.plugin.system.customfieldtypes:select        (single select)
  com.atlassian.jira.plugin.system.customfieldtypes:multiselect   (multi select)
  com.atlassian.jira.plugin.system.customfieldtypes:float         (number)`,
		Example: `  # Create a single-select field
  jtk fields create --name "Environment" --type com.atlassian.jira.plugin.system.customfieldtypes:select

  # Create a text field with description
  jtk fields create --name "Release Notes" --type com.atlassian.jira.plugin.system.customfieldtypes:textarea --description "Notes for the release"`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runCreate(cmd.Context(), opts, name, fieldType, description)
		},
	}

	cmd.Flags().StringVarP(&name, "name", "n", "", "Field name (required)")
	cmd.Flags().StringVarP(&fieldType, "type", "t", "", "Field type (required)")
	cmd.Flags().StringVarP(&description, "description", "d", "", "Field description")

	_ = cmd.MarkFlagRequired("name")
	_ = cmd.MarkFlagRequired("type")

	return cmd
}

func runCreate(ctx context.Context, opts *root.Options, name, fieldType, description string) error {
	v := opts.View()

	client, err := opts.APIClient()
	if err != nil {
		return err
	}

	field, err := client.CreateField(ctx, &api.CreateFieldRequest{
		Name:        name,
		Type:        fieldType,
		Description: description,
	})
	if err != nil {
		return err
	}

	if opts.Output == "json" {
		return v.JSON(field)
	}

	v.Success("Created field %s (%s)", field.ID, field.Name)
	return nil
}

func newDeleteCmd(opts *root.Options) *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "delete <field-id>",
		Short: "Trash a custom field",
		Long: `Move a custom field to the trash (soft delete).

The field can be restored using 'jtk fields restore'.
Trashed fields are permanently deleted after 60 days.`,
		Example: `  # Trash a field (will prompt for confirmation)
  jtk fields delete customfield_10100

  # Trash without confirmation
  jtk fields delete customfield_10100 --force`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDelete(cmd.Context(), opts, args[0], force)
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Skip confirmation prompt")

	return cmd
}

func runDelete(ctx context.Context, opts *root.Options, fieldID string, force bool) error {
	v := opts.View()

	if !force {
		fmt.Fprintf(opts.Stderr, "This will trash field %s. It can be restored later.\n", fieldID)
		fmt.Fprint(opts.Stderr, "Are you sure? [y/N]: ")

		confirmed, err := prompt.Confirm(opts.Stdin)
		if err != nil {
			return fmt.Errorf("reading confirmation: %w", err)
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

	if err := client.TrashField(ctx, fieldID); err != nil {
		return err
	}

	v.Success("Trashed field %s", fieldID)
	return nil
}

func newRestoreCmd(opts *root.Options) *cobra.Command {
	return &cobra.Command{
		Use:   "restore <field-id>",
		Short: "Restore a trashed field",
		Long:  "Restore a custom field from the trash.",
		Example: `  # Restore a trashed field
  jtk fields restore customfield_10100`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRestore(cmd.Context(), opts, args[0])
		},
	}
}

func runRestore(ctx context.Context, opts *root.Options, fieldID string) error {
	v := opts.View()

	client, err := opts.APIClient()
	if err != nil {
		return err
	}

	if err := client.RestoreField(ctx, fieldID); err != nil {
		return err
	}

	v.Success("Restored field %s", fieldID)
	return nil
}
