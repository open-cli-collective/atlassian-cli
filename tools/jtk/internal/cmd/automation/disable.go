package automation

import (
	"github.com/spf13/cobra"

	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
)

func newDisableCmd(opts *root.Options) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "disable <rule-id>",
		Short: "Disable an automation rule",
		Long:  "Disable an enabled automation rule. This is a safe operation that does not modify the rule definition.",
		Example: `  jtk automation disable 12345
  jtk auto disable 12345`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSetState(opts, args[0], false)
		},
	}

	return cmd
}
