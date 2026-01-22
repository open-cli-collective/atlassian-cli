package transitions

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/jira-ticket-cli/api"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
)

// Register registers the transitions commands
func Register(parent *cobra.Command, opts *root.Options) {
	cmd := &cobra.Command{
		Use:     "transitions",
		Aliases: []string{"transition", "tr"},
		Short:   "Manage issue transitions",
		Long:    "Commands for viewing and performing workflow transitions on issues.",
	}

	cmd.AddCommand(newListCmd(opts))
	cmd.AddCommand(newDoCmd(opts))

	parent.AddCommand(cmd)
}

func newListCmd(opts *root.Options) *cobra.Command {
	return &cobra.Command{
		Use:     "list <issue-key>",
		Short:   "List available transitions",
		Long:    "List the available workflow transitions for an issue.",
		Example: `  jira-ticket-cli transitions list PROJ-123`,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runList(opts, args[0])
		},
	}
}

func runList(opts *root.Options, issueKey string) error {
	v := opts.View()

	client, err := opts.APIClient()
	if err != nil {
		return err
	}

	transitions, err := client.GetTransitions(issueKey)
	if err != nil {
		return err
	}

	if len(transitions) == 0 {
		v.Info("No transitions available for %s", issueKey)
		return nil
	}

	if opts.Output == "json" {
		return v.JSON(transitions)
	}

	headers := []string{"ID", "NAME", "TO STATUS"}
	var rows [][]string

	for _, t := range transitions {
		rows = append(rows, []string{t.ID, t.Name, t.To.Name})
	}

	return v.Table(headers, rows)
}

func newDoCmd(opts *root.Options) *cobra.Command {
	var fields []string

	cmd := &cobra.Command{
		Use:   "do <issue-key> <transition>",
		Short: "Perform a transition",
		Long: `Perform a workflow transition on an issue. The transition can be specified by name or ID.

Some transitions require additional fields to be set. Use --field to provide them.`,
		Example: `  # Transition by name
  jira-ticket-cli transitions do PROJ-123 "In Progress"

  # Transition by ID
  jira-ticket-cli transitions do PROJ-123 21

  # Transition with required fields
  jira-ticket-cli transitions do PROJ-123 "In Progress" --field resolution=Done
  jira-ticket-cli transitions do PROJ-123 "Done" --field customfield_10001="some value"`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDo(opts, args[0], args[1], fields)
		},
	}

	cmd.Flags().StringArrayVarP(&fields, "field", "f", nil, "Fields to set during transition (key=value)")

	return cmd
}

func runDo(opts *root.Options, issueKey, transitionNameOrID string, fieldArgs []string) error {
	v := opts.View()

	client, err := opts.APIClient()
	if err != nil {
		return err
	}

	// Get available transitions
	transitions, err := client.GetTransitions(issueKey)
	if err != nil {
		return err
	}

	// Find the transition
	var transitionID string

	// First try by ID
	for _, t := range transitions {
		if t.ID == transitionNameOrID {
			transitionID = t.ID
			break
		}
	}

	// Then try by name
	if transitionID == "" {
		if t := api.FindTransitionByName(transitions, transitionNameOrID); t != nil {
			transitionID = t.ID
		}
	}

	if transitionID == "" {
		v.Error("Transition '%s' not found", transitionNameOrID)
		v.Info("Available transitions:")
		for _, t := range transitions {
			v.Info("  %s: %s -> %s", t.ID, t.Name, t.To.Name)
		}
		return fmt.Errorf("transition not found: %s", transitionNameOrID)
	}

	// Parse fields if provided
	var fields map[string]interface{}
	if len(fieldArgs) > 0 {
		fields = make(map[string]interface{})

		// Get field metadata for name resolution and type detection
		allFields, err := client.GetFields()
		if err != nil {
			return fmt.Errorf("failed to get field metadata: %w", err)
		}

		for _, f := range fieldArgs {
			parts := strings.SplitN(f, "=", 2)
			if len(parts) != 2 {
				return fmt.Errorf("invalid field format: %s (expected key=value)", f)
			}

			key, value := parts[0], parts[1]

			// Try to resolve field name to ID and get field info
			var fieldID string
			var field *api.Field
			if resolved := api.FindFieldByName(allFields, key); resolved != nil {
				fieldID = resolved.ID
				field = resolved
			} else if resolved := api.FindFieldByID(allFields, key); resolved != nil {
				fieldID = resolved.ID
				field = resolved
			} else {
				fieldID = key
			}

			// Format value based on field type
			fields[fieldID] = formatFieldValue(field, value)
		}
	}

	if err := client.DoTransition(issueKey, transitionID, fields); err != nil {
		return err
	}

	v.Success("Transitioned %s", issueKey)
	return nil
}

// formatFieldValue formats a field value based on its type
func formatFieldValue(field *api.Field, value string) interface{} {
	if field == nil {
		return value
	}

	// Handle different field types
	switch field.Schema.Type {
	case "option":
		// Single select fields need {"value": "..."} format
		return map[string]string{"value": value}
	case "array":
		// Multi-select options need [{"value": "..."}] format
		if field.Schema.Items == "option" {
			return []map[string]string{{"value": value}}
		}
		// Other arrays (like labels) are just string arrays
		return []string{value}
	case "user":
		// User fields need {"accountId": "..."} format
		return map[string]string{"accountId": value}
	default:
		return value
	}
}
