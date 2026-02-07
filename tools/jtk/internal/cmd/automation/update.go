package automation

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
)

func newUpdateCmd(opts *root.Options) *cobra.Command {
	var filePath string

	cmd := &cobra.Command{
		Use:   "update <rule-id>",
		Short: "Update an automation rule from a JSON file",
		Long: `Update an automation rule by replacing it with a JSON file.

IMPORTANT: Always export the current rule first before editing:

  jtk auto export <rule-id> > rule.json
  # Edit rule.json — only change fields you understand
  jtk auto update <rule-id> --file rule.json

Automation rule components (triggers, conditions, actions) use undocumented
schemas. Only modify fields you understand. If you are unsure what a field
does, do not change it.

The safest edits are to rule metadata: name, description, labels, and
enabled/disabled state (prefer 'jtk auto enable/disable' for state changes).
Component-level edits require understanding of the specific Jira instance's
field mappings and workflow configuration.`,
		Example: `  jtk automation update 12345 --file rule.json
  jtk auto update 12345 --file updated-rule.json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUpdate(opts, args[0], filePath)
		},
	}

	cmd.Flags().StringVarP(&filePath, "file", "f", "", "Path to JSON file containing the rule definition (required)")
	_ = cmd.MarkFlagRequired("file")

	return cmd
}

func runUpdate(opts *root.Options, ruleID, filePath string) error {
	v := opts.View()

	// Read and validate file before creating the API client so we fail
	// fast on bad input without needing network access.
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file %s: %w", filePath, err)
	}

	if !json.Valid(data) {
		return fmt.Errorf("file %s does not contain valid JSON", filePath)
	}

	client, err := opts.APIClient()
	if err != nil {
		return err
	}

	// Fetch current rule to show what we're updating
	current, err := client.GetAutomationRule(ruleID)
	if err != nil {
		return fmt.Errorf("failed to fetch current rule: %w", err)
	}

	v.Info("Updating rule: %s (UUID: %s, State: %s)", current.Name, current.Identifier(), current.State)

	if err := client.UpdateAutomationRule(ruleID, json.RawMessage(data)); err != nil {
		return err
	}

	v.Success("Updated automation rule %s", ruleID)
	return nil
}
