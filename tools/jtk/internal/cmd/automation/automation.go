package automation

import (
	"github.com/spf13/cobra"

	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
)

// Register registers the automation commands
func Register(parent *cobra.Command, opts *root.Options) {
	cmd := &cobra.Command{
		Use:     "automation",
		Aliases: []string{"auto"},
		Short:   "Manage Jira automation rules",
		Long: `Commands for viewing and managing Jira automation rules.

Automation rules are managed via the Jira Cloud Automation REST API.
Rule components (triggers, conditions, actions) use undocumented schemas.

RECOMMENDED WORKFLOW for editing rules:
  1. jtk auto list                        # Find the rule
  2. jtk auto get <id>                    # Understand it
  3. jtk auto export <id> > rule.json     # Export for editing
  4. # Edit rule.json carefully
  5. jtk auto update <id> --file rule.json # Apply changes

The safest edits are to rule metadata (name, labels, description).
Component-level edits require understanding of the specific Jira instance.
Use enable/disable to toggle rules without touching the full definition.`,
	}

	cmd.AddCommand(newListCmd(opts))
	cmd.AddCommand(newGetCmd(opts))
	cmd.AddCommand(newExportCmd(opts))
	cmd.AddCommand(newUpdateCmd(opts))
	cmd.AddCommand(newEnableCmd(opts))
	cmd.AddCommand(newDisableCmd(opts))

	parent.AddCommand(cmd)
}
