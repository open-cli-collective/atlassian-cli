package configcmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/confluence-cli/internal/cmd/root"
	cflpresent "github.com/open-cli-collective/confluence-cli/internal/present"
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

	_ = cflpresent.Emit(opts, cflpresent.ConfigPresenter{}.PresentTestProgress())

	// Try to list spaces (limit 1) to verify connectivity
	_, err = client.ListSpaces(ctx, nil)
	if err != nil {
		_ = cflpresent.Emit(opts, cflpresent.ConfigPresenter{}.PresentTestFailure())
		return fmt.Errorf("connection test failed: %w", err)
	}
	if err := cflpresent.Emit(opts, cflpresent.ConfigPresenter{}.PresentTestConnectionSuccess()); err != nil {
		return err
	}

	// Get current user details
	user, err := client.GetCurrentUser(ctx)
	if err != nil {
		// User details failed but connection worked - show basic success
		return cflpresent.Emit(opts, cflpresent.ConfigPresenter{}.PresentTestSuccess(nil))
	}

	return cflpresent.Emit(opts, cflpresent.ConfigPresenter{}.PresentTestSuccess(user))
}
