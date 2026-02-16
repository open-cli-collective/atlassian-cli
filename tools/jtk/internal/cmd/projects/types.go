package projects

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
)

func newTypesCmd(opts *root.Options) *cobra.Command {
	return &cobra.Command{
		Use:     "types",
		Short:   "List project types",
		Long:    "List available project types for creating new projects.",
		Example: `  jtk projects types`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runTypes(cmd.Context(), opts)
		},
	}
}

func runTypes(ctx context.Context, opts *root.Options) error {
	v := opts.View()

	client, err := opts.APIClient()
	if err != nil {
		return err
	}

	types, err := client.ListProjectTypes(ctx)
	if err != nil {
		return err
	}

	if len(types) == 0 {
		v.Info("No project types found")
		return nil
	}

	if opts.Output == "json" {
		return v.JSON(types)
	}

	headers := []string{"KEY", "FORMATTED"}
	var rows [][]string

	for _, t := range types {
		rows = append(rows, []string{
			t.Key,
			t.FormattedKey,
		})
	}

	return v.Table(headers, rows)
}
