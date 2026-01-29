package issues

import (
	"github.com/spf13/cobra"

	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
)

// Register registers the issues commands
func Register(parent *cobra.Command, opts *root.Options) {
	cmd := &cobra.Command{
		Use:     "issues",
		Aliases: []string{"issue", "i"},
		Short:   "Manage Jira issues",
		Long:    "Commands for creating, viewing, updating, and searching Jira issues.",
	}

	cmd.AddCommand(newGetCmd(opts))
	cmd.AddCommand(newListCmd(opts))
	cmd.AddCommand(newSearchCmd(opts))
	cmd.AddCommand(newCreateCmd(opts))
	cmd.AddCommand(newUpdateCmd(opts))
	cmd.AddCommand(newDeleteCmd(opts))
	cmd.AddCommand(newAssignCmd(opts))
	cmd.AddCommand(newFieldsCmd(opts))
	cmd.AddCommand(newFieldOptionsCmd(opts))
	cmd.AddCommand(newTypesCmd(opts))

	parent.AddCommand(cmd)
}
