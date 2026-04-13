// Package users provides CLI commands for searching Jira users.
package users

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/atlassian-go/artifact"
	"github.com/open-cli-collective/atlassian-go/view"

	jtkartifact "github.com/open-cli-collective/jira-ticket-cli/internal/artifact"
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

	cmd.AddCommand(newGetCmd(opts))
	cmd.AddCommand(newSearchCmd(opts))

	parent.AddCommand(cmd)
}

func newGetCmd(opts *root.Options) *cobra.Command {
	return &cobra.Command{
		Use:   "get <account-id>",
		Short: "Get user details by account ID",
		Long:  "Retrieve and display details for a specific user by their Jira account ID.",
		Example: `  # Get user details
  jtk users get 61292e4c4f29230069621c5f

  # Get as JSON
  jtk users get 61292e4c4f29230069621c5f -o json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGet(cmd.Context(), opts, args[0])
		},
	}
}

func runGet(ctx context.Context, opts *root.Options, accountID string) error {
	v := opts.View()

	client, err := opts.APIClient()
	if err != nil {
		return err
	}

	user, err := client.GetUser(ctx, accountID)
	if err != nil {
		return err
	}

	if v.Format == view.FormatJSON {
		return v.RenderArtifact(jtkartifact.ProjectUser(user, opts.ArtifactMode()))
	}

	active := "yes"
	if !user.Active {
		active = "no"
	}

	v.Println("Account ID:  %s", user.AccountID)
	v.Println("Name:        %s", user.DisplayName)
	if user.EmailAddress != "" {
		v.Println("Email:       %s", user.EmailAddress)
	}
	v.Println("Active:      %s", active)

	return nil
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
  jtk users search john

  # Get results as JSON
  jtk users search john -o json

  # Limit results
  jtk users search john --max 5`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSearch(cmd.Context(), opts, args[0], maxResults)
		},
	}

	cmd.Flags().IntVar(&maxResults, "max", 10, "Maximum number of results")

	return cmd
}

func runSearch(ctx context.Context, opts *root.Options, query string, maxResults int) error {
	v := opts.View()

	client, err := opts.APIClient()
	if err != nil {
		return err
	}

	users, err := client.SearchUsers(ctx, query, maxResults)
	if err != nil {
		return err
	}

	if len(users) == 0 {
		v.Info("No users found matching '%s'", query)
		return nil
	}

	if v.Format == view.FormatJSON {
		arts := jtkartifact.ProjectUsers(users, opts.ArtifactMode())
		// API returns bare []User with no pagination metadata.
		// Infer hasMore when result count equals requested max.
		hasMore := maxResults > 0 && len(users) == maxResults
		return v.RenderArtifactList(artifact.NewListResult(arts, hasMore))
	}

	headers := []string{"ACCOUNT_ID", "NAME", "EMAIL", "ACTIVE"}
	rows := make([][]string, 0, len(users))

	for _, u := range users {
		active := "yes"
		if !u.Active {
			active = "no"
		}
		rows = append(rows, []string{u.AccountID, u.DisplayName, u.EmailAddress, active})
	}

	return v.Table(headers, rows)
}
