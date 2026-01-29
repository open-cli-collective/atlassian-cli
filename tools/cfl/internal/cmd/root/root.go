// Package root provides the root command for the cfl CLI.
package root

import (
	"github.com/spf13/cobra"

	"github.com/open-cli-collective/confluence-cli/internal/cmd/attachment"
	"github.com/open-cli-collective/confluence-cli/internal/cmd/completion"
	initcmd "github.com/open-cli-collective/confluence-cli/internal/cmd/init"
	"github.com/open-cli-collective/confluence-cli/internal/cmd/page"
	"github.com/open-cli-collective/confluence-cli/internal/cmd/search"
	"github.com/open-cli-collective/confluence-cli/internal/cmd/space"
	"github.com/open-cli-collective/confluence-cli/internal/version"
)

// NewCmdRoot creates the root command for cfl.
func NewCmdRoot() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cfl",
		Short: "A command-line interface for Atlassian Confluence",
		Long: `cfl is a CLI tool for interacting with Atlassian Confluence Cloud.

It provides commands for managing pages, spaces, and attachments
with a markdown-first approach for content editing.

Get started by running: cfl init`,
		SilenceUsage:  true,
		SilenceErrors: true,
		Version:       version.Version,
	}

	// Global flags
	cmd.PersistentFlags().StringP("config", "c", "", "config file (default: ~/.config/cfl/config.yml)")
	cmd.PersistentFlags().StringP("output", "o", "table", "output format: table, json, plain")
	cmd.PersistentFlags().Bool("no-color", false, "disable colored output")

	// Set version template
	cmd.SetVersionTemplate("cfl version {{.Version}} (commit: " + version.Commit + ", built: " + version.Date + ")\n")

	// Subcommands
	cmd.AddCommand(initcmd.NewCmdInit())
	cmd.AddCommand(page.NewCmdPage())
	cmd.AddCommand(space.NewCmdSpace())
	cmd.AddCommand(attachment.NewCmdAttachment())
	cmd.AddCommand(search.NewCmdSearch())
	cmd.AddCommand(completion.NewCmdCompletion())

	return cmd
}
