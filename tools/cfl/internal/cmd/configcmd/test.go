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
		RunE: func(_ *cobra.Command, _ []string) error {
			return runTest(opts)
		},
	}
}

func runTest(opts *root.Options) error {
	// Try to get the API client - this validates config
	client, err := opts.APIClient()
	if err != nil {
		return fmt.Errorf("configuration error: %w", err)
	}

	fmt.Print("Testing connection... ")

	// Try to list spaces (limit 1) to verify connectivity
	_, err = client.ListSpaces(context.Background(), nil)
	if err != nil {
		fmt.Println("failed!")
		fmt.Println()
		fmt.Println("Troubleshooting:")
		fmt.Println("  - Verify your URL is correct (should include https://)")
		fmt.Println("  - Check your email and API token")
		fmt.Println("  - Ensure your API token hasn't expired")
		fmt.Println("  - Verify you have permission to access Confluence")
		fmt.Println()
		fmt.Println("To regenerate an API token:")
		fmt.Println("  https://id.atlassian.com/manage-profile/security/api-tokens")
		return fmt.Errorf("connection test failed: %w", err)
	}

	fmt.Println("success!")
	fmt.Println()

	// Get current user details
	user, err := client.GetCurrentUser(context.Background())
	if err != nil {
		// User details failed but connection worked - show basic success
		fmt.Println("Your cfl configuration is working correctly.")
		return nil
	}

	fmt.Println("Authentication successful")
	fmt.Println("API access verified")
	fmt.Println()

	// Display user info - try DisplayName first, fall back to PublicName
	displayName := user.DisplayName
	if displayName == "" {
		displayName = user.PublicName
	}

	if displayName != "" {
		if user.Email != "" {
			fmt.Printf("Authenticated as: %s (%s)\n", displayName, user.Email)
		} else {
			fmt.Printf("Authenticated as: %s\n", displayName)
		}
	}
	if user.AccountID != "" {
		fmt.Printf("Account ID: %s\n", user.AccountID)
	}

	return nil
}
