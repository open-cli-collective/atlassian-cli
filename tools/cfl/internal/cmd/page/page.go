// Package page provides page-related commands.
package page

import (
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/confluence-cli/internal/cmd/root"
)

// stdinReader returns the reader to use for `--file -` content. Tests inject
// a reader via root.Options.Stdin; in production it defaults to os.Stdin.
func stdinReader(o *root.Options) io.Reader {
	if o != nil && o.Stdin != nil {
		return o.Stdin
	}
	return os.Stdin
}

// Register adds page commands to the root command.
func Register(rootCmd *cobra.Command, opts *root.Options) {
	cmd := &cobra.Command{
		Use:     "page",
		Aliases: []string{"pages"},
		Short:   "Manage Confluence pages",
		Long:    `Commands for creating, viewing, editing, and listing Confluence pages.`,
	}

	cmd.AddCommand(newListCmd(opts))
	cmd.AddCommand(newViewCmd(opts))
	cmd.AddCommand(newHistoryCmd(opts))
	cmd.AddCommand(newCreateCmd(opts))
	cmd.AddCommand(newEditCmd(opts))
	cmd.AddCommand(newDeleteCmd(opts))
	cmd.AddCommand(newCopyCmd(opts))

	rootCmd.AddCommand(cmd)
}
