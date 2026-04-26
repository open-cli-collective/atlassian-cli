package issues

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/atlassian-go/present"

	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
	jtkpresent "github.com/open-cli-collective/jira-ticket-cli/internal/present"
)

func newArchiveCmd(opts *root.Options) *cobra.Command {
	return &cobra.Command{
		Use:   "archive <issue-key> [<issue-key>...]",
		Short: "Archive one or more issues",
		Long:  "Archive one or more Jira issues. Archived issues can be restored later.",
		Example: `  jtk issues archive PROJ-123
  jtk issues archive PROJ-123 PROJ-124 PROJ-125`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runArchive(cmd.Context(), opts, args)
		},
	}
}

func runArchive(ctx context.Context, opts *root.Options, keys []string) error {
	client, err := opts.APIClient()
	if err != nil {
		return err
	}

	result, err := client.ArchiveIssues(ctx, keys)
	if err != nil {
		return err
	}

	hasErrors := len(result.Errors) > 0
	for _, ae := range result.Errors {
		fmt.Fprintf(opts.Stderr, "Archive error: %s (%s)\n", ae.Message, strings.Join(ae.IssueIdsOrKeys, ", "))
	}

	var successKeys []string
	if result.NumberUpdated > 0 && !hasErrors {
		successKeys = keys
	} else if result.NumberUpdated > 0 {
		failedIDs := make(map[string]bool)
		for _, ae := range result.Errors {
			for _, k := range ae.IssueIdsOrKeys {
				failedIDs[k] = true
			}
		}
		for _, key := range keys {
			if !failedIDs[key] {
				successKeys = append(successKeys, key)
			}
		}
	}

	if opts.EmitIDOnly() {
		if err := jtkpresent.EmitIDs(opts, successKeys); err != nil {
			return err
		}
	} else {
		for _, key := range successKeys {
			model := jtkpresent.IssuePresenter{}.PresentArchived(key)
			out := present.Render(model, opts.RenderStyle())
			_, _ = fmt.Fprint(opts.Stdout, out.Stdout)
		}
	}

	if hasErrors {
		failedCount := len(keys) - len(successKeys)
		return fmt.Errorf("%w (%d of %d issue(s) failed)", root.ErrAlreadyReported, failedCount, len(keys))
	}
	return nil
}
