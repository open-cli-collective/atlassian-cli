package issues

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
	jtkpresent "github.com/open-cli-collective/jira-ticket-cli/internal/present"
)

func newRankCmd(opts *root.Options) *cobra.Command {
	var before, after string

	cmd := &cobra.Command{
		Use:   "rank <issue-key>... (--before <issue-key> | --after <issue-key>)",
		Short: "Rank issues before or after another issue",
		Long: `Move one or more issues to sit immediately before or after another issue in
Jira's rank - the order boards and backlogs show. The issues keep the order
given. Exactly one of --before and --after is required.

Ranking moves issues within the order and loses no data, so there is no prompt.`,
		Example: `  # Put PROJ-7 directly above PROJ-3
  jtk issues rank PROJ-7 --before PROJ-3

  # Put three issues, in this order, directly below PROJ-3
  jtk issues rank PROJ-7 PROJ-8 PROJ-9 --after PROJ-3`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRank(cmd.Context(), opts, args, before, after)
		},
	}

	cmd.Flags().StringVar(&before, "before", "", "Rank the issues immediately before this issue")
	cmd.Flags().StringVar(&after, "after", "", "Rank the issues immediately after this issue")
	cmd.MarkFlagsMutuallyExclusive("before", "after")
	cmd.MarkFlagsOneRequired("before", "after")

	return cmd
}

func runRank(ctx context.Context, opts *root.Options, issueKeys []string, before, after string) error {
	anchor, position := before, "before"
	if after != "" {
		anchor, position = after, "after"
	}
	for _, k := range issueKeys {
		if strings.EqualFold(k, anchor) {
			return fmt.Errorf("cannot rank %s %s itself", k, position)
		}
	}

	client, err := opts.APIClient()
	if err != nil {
		return err
	}

	failures, err := client.RankIssues(ctx, issueKeys, before, after)
	if err != nil {
		return err
	}
	if len(failures) > 0 {
		lines := make([]string, len(failures))
		for i, f := range failures {
			lines[i] = fmt.Sprintf("%s (%d): %s", f.Key, f.Status, strings.Join(f.Errors, "; "))
		}
		return fmt.Errorf("jira did not rank %d of %d issue(s):\n  %s",
			len(failures), len(issueKeys), strings.Join(lines, "\n  "))
	}

	if opts.EmitIDOnly() {
		return jtkpresent.EmitIDs(opts, issueKeys)
	}
	return jtkpresent.Emit(opts, jtkpresent.IssuePresenter{}.PresentRanked(issueKeys, position, anchor))
}
