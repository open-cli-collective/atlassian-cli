package configcmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/confluence-cli/internal/cmd/root"
)

func newTestCmd(opts *root.Options) *cobra.Command {
	return &cobra.Command{
		Use:   "test",
		Short: "Test connectivity with current configuration",
		Long: `Test the connection to Confluence using the current configuration.

This verifies that:
- The URL is reachable
- The credentials are valid
- You have permission to access the API`,
		Example: `  # Test current configuration
  cfl config test`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runTest(cmd.Context(), opts)
		},
	}
}

func runTest(ctx context.Context, opts *root.Options) error {
	// Try to get the API client - this validates config
	client, err := opts.APIClient()
	if err != nil {
		return fmt.Errorf("configuration error: %w", err)
	}

	_, _ = fmt.Fprint(opts.Stderr, "Testing connection... ")

	// Try to list spaces (limit 1) to verify connectivity
	_, err = client.ListSpaces(ctx, nil)
	if err != nil {
		_, _ = fmt.Fprintln(opts.Stderr, "failed!")
		_, _ = fmt.Fprintln(opts.Stderr)
		_, _ = fmt.Fprintln(opts.Stderr, "Troubleshooting:")
		_, _ = fmt.Fprintln(opts.Stderr, "  - Verify your URL is correct (should include https://)")
		_, _ = fmt.Fprintln(opts.Stderr, "  - Check your email and API token")
		_, _ = fmt.Fprintln(opts.Stderr, "  - Ensure your API token hasn't expired")
		_, _ = fmt.Fprintln(opts.Stderr, "  - Verify you have permission to access Confluence")
		_, _ = fmt.Fprintln(opts.Stderr)
		_, _ = fmt.Fprintln(opts.Stderr, "To regenerate an API token:")
		_, _ = fmt.Fprintln(opts.Stderr, "  https://id.atlassian.com/manage-profile/security/api-tokens")
		return fmt.Errorf("connection test failed: %w", err)
	}

	_, _ = fmt.Fprintln(opts.Stderr, "success!")
	_, _ = fmt.Fprintln(opts.Stderr)

	// Get current user details
	user, err := client.GetCurrentUser(ctx)
	if err != nil {
		// User details failed but connection worked - show basic success
		_, _ = fmt.Fprintln(opts.Stderr, "Your cfl configuration is working correctly.")
		return nil
	}

	_, _ = fmt.Fprintln(opts.Stderr, "Authentication successful")
	_, _ = fmt.Fprintln(opts.Stderr, "API access verified")
	_, _ = fmt.Fprintln(opts.Stderr)

	// Display user info - try DisplayName first, fall back to PublicName
	displayName := user.DisplayName
	if displayName == "" {
		displayName = user.PublicName
	}

	if displayName != "" {
		if user.Email != "" {
			_, _ = fmt.Fprintf(opts.Stderr, "Authenticated as: %s (%s)\n", displayName, user.Email)
		} else {
			_, _ = fmt.Fprintf(opts.Stderr, "Authenticated as: %s\n", displayName)
		}
	}
	if user.AccountID != "" {
		_, _ = fmt.Fprintf(opts.Stderr, "Account ID: %s\n", user.AccountID)
	}

	return nil
}
