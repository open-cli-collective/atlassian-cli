package automation

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
)

func newCreateCmd(opts *root.Options) *cobra.Command {
	var filePath string

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create an automation rule from a JSON file",
		Long: `Create a new automation rule from a JSON file.

The recommended workflow is to export an existing rule, modify it,
and create a new rule from the modified JSON:

  jtk auto export <source-id> > new-rule.json
  # Edit new-rule.json (change name, adjust components)
  jtk auto create --file new-rule.json

The API auto-generates new IDs. Fields like 'id' and 'ruleKey' from
the exported JSON are ignored — the new rule gets its own identifiers.

New rules are created in DISABLED state by default.`,
		Example: `  jtk automation create --file rule.json
  jtk auto create -f new-rule.json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCreate(opts, filePath)
		},
	}

	cmd.Flags().StringVarP(&filePath, "file", "f", "", "Path to JSON file containing the rule definition (required)")
	_ = cmd.MarkFlagRequired("file")

	return cmd
}

func runCreate(opts *root.Options, filePath string) error {
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

	respBody, err := client.CreateAutomationRule(json.RawMessage(data))
	if err != nil {
		return err
	}

	// Parse the response to extract the new rule's ID
	var created struct {
		ID      json.Number `json:"id"`
		RuleKey string      `json:"ruleKey"`
		Name    string      `json:"name"`
	}
	if err := json.Unmarshal(respBody, &created); err != nil {
		// Even if we can't parse the response, the rule was created
		v.Success("Created automation rule (could not parse response for details)")
		return nil
	}

	if created.Name != "" {
		v.Success("Created automation rule: %s (ID: %s)", created.Name, created.ID.String())
	} else {
		v.Success("Created automation rule (ID: %s)", created.ID.String())
	}

	return nil
}
