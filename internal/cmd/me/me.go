package me

import (
	"github.com/spf13/cobra"

	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
)

// Register registers the me command
func Register(parent *cobra.Command, opts *root.Options) {
	cmd := &cobra.Command{
		Use:   "me",
		Short: "Show current user",
		Long:  "Show information about the currently authenticated Jira user.",
		Example: `  # Show current user info
  jira-ticket-cli me

  # Show just the account ID (for scripting)
  jira-ticket-cli me -o plain`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(opts)
		},
	}

	parent.AddCommand(cmd)
}

func run(opts *root.Options) error {
	v := opts.View()

	client, err := opts.APIClient()
	if err != nil {
		return err
	}

	user, err := client.GetCurrentUser()
	if err != nil {
		return err
	}

	if opts.Output == "json" {
		return v.JSON(user)
	}

	if opts.Output == "plain" {
		v.Println("%s", user.AccountID)
		return nil
	}

	v.Println("Account ID:   %s", user.AccountID)
	v.Println("Display Name: %s", user.DisplayName)
	if user.EmailAddress != "" {
		v.Println("Email:        %s", user.EmailAddress)
	}
	v.Println("Active:       %t", user.Active)

	return nil
}
