package automation

import (
	"strings"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/atlassian-go/view"

	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
)

func newListCmd(opts *root.Options) *cobra.Command {
	var state string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List automation rules",
		Long:  "List all automation rules with optional state filtering.",
		Example: `  jtk automation list
  jtk automation list --state ENABLED
  jtk auto list -o json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runList(opts, strings.ToUpper(state))
		},
	}

	cmd.Flags().StringVar(&state, "state", "", "Filter by state (ENABLED or DISABLED)")

	return cmd
}

func runList(opts *root.Options, state string) error {
	v := opts.View()

	client, err := opts.APIClient()
	if err != nil {
		return err
	}

	rules, err := client.ListAutomationRulesFiltered(state)
	if err != nil {
		return err
	}

	if len(rules) == 0 {
		v.Info("No automation rules found")
		return nil
	}

	headers := []string{"UUID", "NAME", "STATE", "LABELS"}
	rows := make([][]string, 0, len(rules))
	for _, r := range rules {
		labels := "-"
		if len(r.Labels) > 0 {
			labels = strings.Join(r.Labels, ", ")
		}
		if len(r.Tags) > 0 && labels == "-" {
			labels = strings.Join(r.Tags, ", ")
		}

		rows = append(rows, []string{
			r.Identifier(),
			view.Truncate(r.Name, 60),
			r.State,
			labels,
		})
	}

	if opts.Output == "json" {
		return v.JSON(rules)
	}

	return v.Table(headers, rows)
}
