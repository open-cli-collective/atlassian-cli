package automation

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/atlassian-go/present"

	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
	jtkpresent "github.com/open-cli-collective/jira-ticket-cli/internal/present"
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
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runCreate(cmd.Context(), opts, filePath)
		},
	}

	cmd.Flags().StringVarP(&filePath, "file", "f", "", "Path to JSON file containing the rule definition (required)")
	_ = cmd.MarkFlagRequired("file")

	return cmd
}

func runCreate(ctx context.Context, opts *root.Options, filePath string) error {

	// Read and validate file before creating the API client so we fail
	// fast on bad input without needing network access.
	data, err := os.ReadFile(filePath) //nolint:gosec // CLI tool reads user-provided file paths
	if err != nil {
		return fmt.Errorf("reading file %s: %w", filePath, err)
	}

	if !json.Valid(data) {
		return fmt.Errorf("file %s does not contain valid JSON", filePath)
	}

	// Strip server-assigned fields that would cause conflicts on create.
	// The API rejects requests containing a UUID that already exists.
	var ruleMap map[string]any
	if err := json.Unmarshal(data, &ruleMap); err != nil {
		return fmt.Errorf("parsing rule JSON: %w", err)
	}
	for _, key := range []string{"uuid", "id", "ruleKey", "created", "updated"} {
		delete(ruleMap, key)
	}
	data, err = json.Marshal(ruleMap)
	if err != nil {
		return fmt.Errorf("re-encoding rule JSON: %w", err)
	}

	client, err := opts.APIClient()
	if err != nil {
		return err
	}

	respBody, err := client.CreateAutomationRule(ctx, json.RawMessage(data))
	if err != nil {
		return err
	}

	// Parse the response to extract the new rule's identifier.
	var created struct {
		// Legacy/documented response fields.
		ID      json.Number `json:"id"`
		RuleKey string      `json:"ruleKey"`
		// Observed Cloud response fields.
		UUID     string `json:"uuid"`
		RuleUUID string `json:"ruleUuid"`
		Name     string `json:"name"`
	}
	if err := json.Unmarshal(respBody, &created); err != nil {
		// Even if we can't parse the response, the rule was created.
		model := jtkpresent.AutomationPresenter{}.PresentCreatedUnparsed()
		out := present.Render(model, opts.RenderStyle())
		fmt.Fprint(opts.Stdout, out.Stdout)
		return nil
	}

	identifier := created.UUID
	if identifier == "" {
		identifier = created.RuleUUID
	}
	if identifier == "" {
		identifier = created.RuleKey
	}
	if identifier == "" {
		identifier = created.ID.String()
	}

	var model *present.OutputModel
	if created.Name != "" {
		model = jtkpresent.AutomationPresenter{}.PresentCreated(created.Name, identifier)
	} else {
		model = jtkpresent.AutomationPresenter{}.PresentCreatedMinimal(identifier)
	}
	out := present.Render(model, opts.RenderStyle())
	fmt.Fprint(opts.Stdout, out.Stdout)

	return nil
}
