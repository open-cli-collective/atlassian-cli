// Package main is the entry point for the cfl (Confluence) CLI.
//
// Distribution is fully automated: merges to main with feat:/fix: prefixes
// trigger auto-release, which runs GoReleaser (Homebrew + binary artifacts)
// and dispatches the chocolatey and winget publish workflows.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/open-cli-collective/atlassian-go/exitcode"
	"github.com/open-cli-collective/atlassian-go/keyring"

	"github.com/open-cli-collective/confluence-cli/internal/cmd/attachment"
	"github.com/open-cli-collective/confluence-cli/internal/cmd/completion"
	"github.com/open-cli-collective/confluence-cli/internal/cmd/configcmd"
	initcmd "github.com/open-cli-collective/confluence-cli/internal/cmd/init"
	"github.com/open-cli-collective/confluence-cli/internal/cmd/me"
	"github.com/open-cli-collective/confluence-cli/internal/cmd/page"
	"github.com/open-cli-collective/confluence-cli/internal/cmd/root"
	"github.com/open-cli-collective/confluence-cli/internal/cmd/search"
	"github.com/open-cli-collective/confluence-cli/internal/cmd/setcredential"
	"github.com/open-cli-collective/confluence-cli/internal/cmd/space"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cmd, opts := root.NewCmd()

	root.RegisterCommands(cmd, opts,
		initcmd.Register,
		configcmd.Register,
		me.Register,
		page.Register,
		space.Register,
		attachment.Register,
		search.Register,
		setcredential.Register,
		completion.Register,
	)

	err := cmd.ExecuteContext(ctx)
	// Emit the one-time §1.8 migration notice (if migration ran this
	// invocation) before exiting — flushed here, not in a defer, so it
	// still prints when a later command error triggers os.Exit.
	keyring.FlushMigrationNotice(os.Stderr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(exitcode.GeneralError)
	}
}
