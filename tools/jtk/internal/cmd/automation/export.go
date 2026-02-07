package automation

import (
	"bytes"
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

This outputs the exact JSON returned by the API, suitable for editing
and re-importing via 'jtk auto update'. The -o flag is ignored; output
is always JSON.

RECOMMENDED WORKFLOW:
  jtk auto export <rule-id> > rule.json
  # Edit rule.json — only change fields you understand
  jtk auto update <rule-id> --file rule.json`,
		Example: `  jtk automation export 12345
  jtk auto export 12345 > rule.json
  jtk auto export 12345 --compact`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runExport(opts, args[0], compact)
		},
	}

	cmd.Flags().BoolVar(&compact, "compact", false, "Output minified JSON")

	return cmd
}

func runExport(opts *root.Options, ruleID string, compact bool) error {
	client, err := opts.APIClient()
	if err != nil {
		return err
	}

	raw, err := client.GetAutomationRuleRaw(ruleID)
	if err != nil {
		return err
	}

	if compact {
		_, err = fmt.Fprintln(opts.Stdout, string(raw))
		return err
	}

	// Pretty-print the JSON
	var buf bytes.Buffer
	if err := json.Indent(&buf, raw, "", "  "); err != nil {
		// If indenting fails, output raw
		_, err = fmt.Fprintln(opts.Stdout, string(raw))
		return err
	}

	_, err = fmt.Fprintln(opts.Stdout, buf.String())
	return err
}
