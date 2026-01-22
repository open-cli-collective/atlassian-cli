package users

import (
	"github.com/spf13/cobra"

	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
)

// Register registers the users commands
func Register(parent *cobra.Command, opts *root.Options) {
	cmd := &cobra.Command{
		Use:     "users",
		Aliases: []string{"user", "u"},
		Short:   "Search and lookup users",
		Long:    "Commands for searching and looking up Jira users.",
	}

	cmd.AddCommand(newSearchCmd(opts))

	parent.AddCommand(cmd)
}

func newSearchCmd(opts *root.Options) *cobra.Command {
	var maxResults int

	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search for users",
		Long: `Search for users by name, email, or username.

The search is case-insensitive and matches against display name, email address,
and other user attributes. Use this to find account IDs for issue assignment.`,
		Example: `  # Search for users named "john"
  jira-ticket-cli users search john

  # Get results as JSON
  jira-ticket-cli users search john -o json

  # Limit results
  jira-ticket-cli users search john --max 5`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSearch(opts, args[0], maxResults)
		},
	}

	cmd.Flags().IntVar(&maxResults, "max", 10, "Maximum number of results")

	return cmd
}

func runSearch(opts *root.Options, query string, maxResults int) error {
	v := opts.View()

	client, err := opts.APIClient()
	if err != nil {
		return err
	}

	users, err := client.SearchUsers(query, maxResults)
	if err != nil {
		return err
	}

	if len(users) == 0 {
		v.Info("No users found matching '%s'", query)
		return nil
	}

	if opts.Output == "json" {
		return v.JSON(users)
	}

	headers := []string{"ACCOUNT_ID", "NAME", "EMAIL", "ACTIVE"}
	var rows [][]string

	for _, u := range users {
		active := "yes"
		if !u.Active {
			active = "no"
		}
		rows = append(rows, []string{u.AccountID, u.DisplayName, u.EmailAddress, active})
	}

	return v.Table(headers, rows)
}
