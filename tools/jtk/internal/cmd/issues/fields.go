package issues

import (
	"context"
	"strings"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/atlassian-go/view"

	"github.com/open-cli-collective/jira-ticket-cli/api"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cache"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
	jtkpresent "github.com/open-cli-collective/jira-ticket-cli/internal/present"
)

func newFieldsCmd(opts *root.Options) *cobra.Command {
	var customOnly bool

	cmd := &cobra.Command{
		Use:   "fields [issue-key]",
		Short: "List available fields",
		Long:  "List fields and their metadata. If an issue key is provided, shows all fields with their current values for that issue.",
		Example: `  # List all fields
  jtk issues fields

  # List only custom fields
  jtk issues fields --custom-fields

  # Show field values for a specific issue
  jtk issues fields PROJ-123

  # Show only custom field values for an issue
  jtk issues fields PROJ-123 --custom-fields

  # Extended output with searchable/navigable/orderable/clause names
  jtk issues fields --extended

  # Emit only field IDs
  jtk issues fields --id`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			issueKey := ""
			if len(args) > 0 {
				issueKey = args[0]
			}
			return runFields(cmd.Context(), opts, issueKey, customOnly)
		},
	}

	cmd.Flags().BoolVar(&customOnly, "custom-fields", false, "Show only custom fields")
	cmd.Flags().BoolVar(&customOnly, "custom", false, "Show only custom fields")
	_ = cmd.Flags().MarkHidden("custom")

	return cmd
}

func runFields(ctx context.Context, opts *root.Options, issueKey string, customOnly bool) error {
	client, err := opts.APIClient()
	if err != nil {
		return err
	}

	if issueKey != "" {
		return runIssueFields(ctx, opts, client, issueKey, customOnly)
	}
	return runGlobalFields(ctx, opts, client, customOnly)
}

func runGlobalFields(ctx context.Context, opts *root.Options, client *api.Client, customOnly bool) error {
	fields, err := cache.GetFieldsCacheFirst(ctx, client)
	if err != nil {
		return err
	}
	if customOnly {
		var custom []api.Field
		for _, f := range fields {
			if f.Custom {
				custom = append(custom, f)
			}
		}
		fields = custom
	}

	if opts.EmitIDOnly() {
		ids := make([]string, len(fields))
		for i, f := range fields {
			ids[i] = f.ID
		}
		return jtkpresent.EmitIDs(opts, ids)
	}

	if len(fields) == 0 {
		return jtkpresent.Emit(opts, jtkpresent.FieldPresenter{}.PresentEmpty())
	}

	v := opts.View()
	if v.Format == view.FormatJSON {
		return v.JSON(fields)
	}

	model := jtkpresent.FieldPresenter{}.PresentList(fields, opts.IsExtended())
	return jtkpresent.Emit(opts, model)
}

func runIssueFields(ctx context.Context, opts *root.Options, client *api.Client, issueKey string, customOnly bool) error {
	issue, err := client.GetIssue(ctx, issueKey)
	if err != nil {
		return err
	}

	fields, err := cache.GetFieldsCacheFirst(ctx, client)
	if err != nil {
		return err
	}

	entries := api.ExtractIssueFieldValues(issue, fields)

	if customOnly {
		var filtered []api.IssueFieldEntry
		for _, e := range entries {
			if strings.HasPrefix(e.ID, "customfield_") {
				filtered = append(filtered, e)
			}
		}
		entries = filtered
	}

	if len(entries) == 0 {
		return jtkpresent.Emit(opts, jtkpresent.FieldPresenter{}.PresentEmpty())
	}

	if opts.EmitIDOnly() {
		ids := make([]string, len(entries))
		for i, e := range entries {
			ids[i] = e.ID
		}
		return jtkpresent.EmitIDs(opts, ids)
	}

	v := opts.View()
	if v.Format == view.FormatJSON {
		return v.JSON(entries)
	}

	model := jtkpresent.FieldPresenter{}.PresentIssueFields(entries)
	return jtkpresent.Emit(opts, model)
}
