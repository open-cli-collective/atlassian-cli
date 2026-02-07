package main

import (
	"fmt"
	"os"

	"github.com/open-cli-collective/atlassian-go/exitcode"

	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/attachments"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/automation"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/boards"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/comments"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/completion"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/configcmd"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/initcmd"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/issues"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/me"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/sprints"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/transitions"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/users"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(exitcode.GeneralError)
	}
}

func run() error {
	rootCmd, opts := root.NewCmd()

	// Register all commands
	initcmd.Register(rootCmd, opts)
	configcmd.Register(rootCmd, opts)
	issues.Register(rootCmd, opts)
	transitions.Register(rootCmd, opts)
	comments.Register(rootCmd, opts)
	attachments.Register(rootCmd, opts)
	automation.Register(rootCmd, opts)
	boards.Register(rootCmd, opts)
	sprints.Register(rootCmd, opts)
	users.Register(rootCmd, opts)
	me.Register(rootCmd, opts)
	completion.Register(rootCmd, opts)

	return rootCmd.Execute()
}
