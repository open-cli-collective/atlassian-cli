package automation

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/jira-ticket-cli/api"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
)

func newGetCmd(opts *root.Options) *cobra.Command {
	var full bool

	cmd := &cobra.Command{
		Use:   "get <rule-id>",
		Short: "Get automation rule details",
		Long: `Retrieve and display details for a specific automation rule.

Shows rule metadata and a summary of components. Use --full to see
component type details. Use -o json for the full rule object.

For the exact JSON needed for editing, use 'jtk auto export' instead.`,
		Example: `  jtk automation get 12345
  jtk auto get 12345 --full
  jtk auto get 12345 -o json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGet(opts, args[0], full)
		},
	}

	cmd.Flags().BoolVar(&full, "full", false, "Show component type details")

	return cmd
}

func runGet(opts *root.Options, ruleID string, full bool) error {
	v := opts.View()

	client, err := opts.APIClient()
	if err != nil {
		return err
	}

	rule, err := client.GetAutomationRule(ruleID)
	if err != nil {
		return err
	}

	if opts.Output == "json" {
		return v.JSON(rule)
	}

	v.Println("Name:        %s", rule.Name)
	v.Println("UUID:        %s", rule.Identifier())
	v.Println("State:       %s", rule.State)

	if rule.Description != "" {
		v.Println("Description: %s", rule.Description)
	}

	if len(rule.Labels) > 0 {
		v.Println("Labels:      %s", strings.Join(rule.Labels, ", "))
	}
	if len(rule.Tags) > 0 {
		v.Println("Tags:        %s", strings.Join(rule.Tags, ", "))
	}

	if len(rule.Projects) > 0 {
		projects := make([]string, 0, len(rule.Projects))
		for _, p := range rule.Projects {
			if p.ProjectKey != "" {
				projects = append(projects, p.ProjectKey)
			} else if p.ProjectName != "" {
				projects = append(projects, p.ProjectName)
			}
		}
		if len(projects) > 0 {
			v.Println("Projects:    %s", strings.Join(projects, ", "))
		}
	}

	v.Println("Components:  %s", summarizeComponents(rule.Components))

	if full && len(rule.Components) > 0 {
		v.Println("")
		v.Println("Component Details:")
		for i, c := range rule.Components {
			v.Println("  [%d] %s: %s", i+1, c.Component, c.Type)
		}
	}

	return nil
}

func summarizeComponents(components []api.RuleComponent) string {
	if len(components) == 0 {
		return "none"
	}

	triggers, conditions, actions := 0, 0, 0
	for _, c := range components {
		switch c.Component {
		case "TRIGGER":
			triggers++
		case "CONDITION":
			conditions++
		case "ACTION":
			actions++
		}
	}

	parts := make([]string, 0, 3)
	if triggers > 0 {
		parts = append(parts, fmt.Sprintf("%d trigger(s)", triggers))
	}
	if conditions > 0 {
		parts = append(parts, fmt.Sprintf("%d condition(s)", conditions))
	}
	if actions > 0 {
		parts = append(parts, fmt.Sprintf("%d action(s)", actions))
	}

	return fmt.Sprintf("%d total — %s", len(components), strings.Join(parts, ", "))
}
