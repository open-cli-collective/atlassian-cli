// Package development provides read-only access to Jira's Development panel.
package development

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
	jtkpresent "github.com/open-cli-collective/jira-ticket-cli/internal/present"
)

// Register registers the development command.
func Register(parent *cobra.Command, opts *root.Options) {
	cmd := &cobra.Command{
		Use:   "development",
		Short: "Read issue development data",
		Long:  "Read the pull requests shown in an issue's Jira Development panel. This uses a private Jira API that may change without notice.",
	}
	cmd.AddCommand(newGetCmd(opts))
	parent.AddCommand(cmd)
}

func newGetCmd(opts *root.Options) *cobra.Command {
	return &cobra.Command{
		Use:   "get <issue-key>",
		Short: "Get pull requests from an issue's Development panel",
		Long:  "Get the summary and deduplicated pull requests shown in an issue's Jira Development panel.",
		Example: `  jtk development get PROJ-123
  jtk development get PROJ-123 --id`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGet(cmd.Context(), opts, args[0])
		},
	}
}

func runGet(ctx context.Context, opts *root.Options, issueKey string) error {
	client, err := opts.APIClient()
	if err != nil {
		return err
	}

	development, err := client.GetDevelopment(ctx, issueKey)
	if err != nil {
		return err
	}

	if opts.EmitIDOnly() {
		urls := make([]string, len(development.PullRequests))
		for i, pr := range development.PullRequests {
			if pr.URL == "" {
				return fmt.Errorf("pull request %s has no URL", pr.ID)
			}
			urls[i] = pr.URL
		}
		return jtkpresent.EmitIDs(opts, urls)
	}

	return jtkpresent.Emit(opts, jtkpresent.DevelopmentPresenter{}.Present(development))
}
