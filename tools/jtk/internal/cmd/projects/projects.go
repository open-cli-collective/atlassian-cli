package projects

import (
	"github.com/spf13/cobra"

	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
)

// Register registers the projects commands
func Register(parent *cobra.Command, opts *root.Options) {
	cmd := &cobra.Command{
		Use:     "projects",
		Aliases: []string{"project", "proj", "p"},
		Short:   "Manage Jira projects",
		Long:    "Commands for creating, viewing, updating, and deleting Jira projects.",
	}

	cmd.AddCommand(newListCmd(opts))
	cmd.AddCommand(newGetCmd(opts))
	cmd.AddCommand(newCreateCmd(opts))
	cmd.AddCommand(newUpdateCmd(opts))
	cmd.AddCommand(newDeleteCmd(opts))
	cmd.AddCommand(newRestoreCmd(opts))
	cmd.AddCommand(newTypesCmd(opts))

	parent.AddCommand(cmd)
}
