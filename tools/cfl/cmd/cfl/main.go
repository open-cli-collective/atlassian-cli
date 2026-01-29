// Package main is the entry point for the cfl CLI.
package main

import (
	"fmt"
	"os"

	"github.com/open-cli-collective/atlassian-go/exitcode"

	"github.com/open-cli-collective/confluence-cli/internal/cmd/attachment"
	"github.com/open-cli-collective/confluence-cli/internal/cmd/completion"
	initcmd "github.com/open-cli-collective/confluence-cli/internal/cmd/init"
	"github.com/open-cli-collective/confluence-cli/internal/cmd/page"
	"github.com/open-cli-collective/confluence-cli/internal/cmd/root"
	"github.com/open-cli-collective/confluence-cli/internal/cmd/search"
	"github.com/open-cli-collective/confluence-cli/internal/cmd/space"
)

func main() {
	cmd, opts := root.NewCmd()

	root.RegisterCommands(cmd, opts,
		initcmd.Register,
		page.Register,
		space.Register,
		attachment.Register,
		search.Register,
		completion.Register,
	)

	if err := cmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(exitcode.GeneralError)
	}
}
