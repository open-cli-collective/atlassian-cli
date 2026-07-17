package automation

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
)

func newExportCmd(opts *root.Options) *cobra.Command {
	var compact bool

	cmd := &cobra.Command{
		Use:   "export <rule-id>",
		Short: "Export automation rule as JSON",
		Long: `Export the full automation rule definition as JSON.

This validates and formats the JSON returned by the API, suitable for
editing and re-importing via 'jtk auto update'. Output is always JSON.

RECOMMENDED WORKFLOW:
  jtk auto export <rule-id> > rule.json
  # Edit rule.json — only change fields you understand
  jtk auto update <rule-id> --file rule.json`,
		Example: `  jtk automation export 12345
  jtk auto export 12345 > rule.json
  jtk auto export 12345 --compact`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runExport(cmd.Context(), opts, args[0], compact)
		},
	}

	cmd.Flags().BoolVar(&compact, "compact", false, "Output minified JSON")

	return cmd
}

func runExport(ctx context.Context, opts *root.Options, ruleID string, compact bool) error {
	client, err := opts.APIClient()
	if err != nil {
		return err
	}

	raw, err := client.GetAutomationRuleRaw(ctx, ruleID)
	if err != nil {
		return err
	}

	var buf bytes.Buffer
	if compact {
		err = json.Compact(&buf, raw)
	} else {
		err = json.Indent(&buf, raw, "", "  ")
	}
	if err != nil {
		return fmt.Errorf("formatting automation rule JSON: %w", err)
	}

	_, err = fmt.Fprintln(opts.Stdout, buf.String())
	return err
}
