// Package main is the entry point for the jtk CLI.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/open-cli-collective/atlassian-go/exitcode"

	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/attachments"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/automation"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/boards"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/comments"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/completion"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/configcmd"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/dashboards"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/fields"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/initcmd"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/issues"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/links"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/me"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/projects"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/refresh"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/sprints"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/transitions"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/users"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := run(ctx); err != nil {
		if !errors.Is(err, refresh.ErrAlreadyReported) {
			fmt.Fprintln(os.Stderr, err)
		}
		os.Exit(exitcode.GeneralError)
	}
}

func run(ctx context.Context) error {
	rootCmd, opts := root.NewCmd()

	// Register all commands
	initcmd.Register(rootCmd, opts)
	configcmd.Register(rootCmd, opts)
	fields.Register(rootCmd, opts)
	issues.Register(rootCmd, opts)
	transitions.Register(rootCmd, opts)
	comments.Register(rootCmd, opts)
	links.Register(rootCmd, opts)
	attachments.Register(rootCmd, opts)
	automation.Register(rootCmd, opts)
	boards.Register(rootCmd, opts)
	dashboards.Register(rootCmd, opts)
	projects.Register(rootCmd, opts)
	sprints.Register(rootCmd, opts)
	users.Register(rootCmd, opts)
	me.Register(rootCmd, opts)
	refresh.Register(rootCmd, opts)
	completion.Register(rootCmd, opts)

	return rootCmd.ExecuteContext(ctx)
}
